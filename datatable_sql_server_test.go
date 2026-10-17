package insyra

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

// These tests run against a real MySQL or PostgreSQL server, and only when
// its DSN is set, for example with Docker:
//
//	INSYRA_TEST_MYSQL_DSN='root:pw@tcp(127.0.0.1:33061)/shop?parseTime=true'
//	INSYRA_TEST_POSTGRES_DSN='postgres://postgres:pw@127.0.0.1:54321/shop?sslmode=disable'
//
// SQLite runs every path in the other tests; what only a server can show is
// its own SQL: information_schema lookups, $n placeholders, MySQL's implicit
// commits and RENAME TABLE, and the drivers' type mapping.
type sqlServer struct {
	name, driver, env, schema, placeholder string
	dialect                                SQLDialect
}

var sqlServers = []sqlServer{
	{"mysql", "mysql", "INSYRA_TEST_MYSQL_DSN", "shop", "?", SQLDialectMySQL},
	{"postgres", "pgx", "INSYRA_TEST_POSTGRES_DSN", "public", "$1", SQLDialectPostgres},
}

func eachSQLServer(t *testing.T, run func(t *testing.T, s sqlServer, db *sql.DB)) {
	ran := false
	for _, s := range sqlServers {
		dsn := os.Getenv(s.env)
		if dsn == "" {
			continue
		}
		ran = true
		t.Run(s.name, func(t *testing.T) {
			db, err := sql.Open(s.driver, dsn)
			require.NoError(t, err)
			t.Cleanup(func() { _ = db.Close() })
			require.NoError(t, db.Ping())
			run(t, s, db)
		})
	}
	if !ran {
		t.Skip("no INSYRA_TEST_MYSQL_DSN or INSYRA_TEST_POSTGRES_DSN set")
	}
}

// serverTable gives each test a table name of its own and drops it afterwards.
func serverTable(t *testing.T, s sqlServer, db *sql.DB, names ...string) {
	t.Helper()
	drop := func() {
		for _, n := range names {
			_, _ = db.Exec("DROP TABLE IF EXISTS " + quoteSQLIdent(string(s.dialect), n))
		}
	}
	drop()
	t.Cleanup(drop)
}

func serverCount(t *testing.T, s sqlServer, db *sql.DB, table string) int {
	t.Helper()
	var n int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+quoteSQLIdent(string(s.dialect), table)).Scan(&n))
	return n
}

func TestSQLServerRoundTrip(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra it round"
		serverTable(t, s, db, table)
		dt := NewDataTable(
			NewDataList(1, 2, 3).SetName("id"),
			NewDataList(1.5, 2.5, 3.5).SetName("amount"),
			NewDataList("a", `b"q`, "c`d").SetName("label"),
		)
		require.NoError(t, dt.ToSQL(db, s.dialect, table, ToSQLOptions{Schema: s.schema}))

		got, err := ReadSQL(db, s.dialect, table, ReadSQLOptions{Schema: s.schema, OrderBy: "id"})
		require.NoError(t, err)
		require.Equal(t, 3, got.NumRows())
		require.EqualValues(t, 2, got.GetColByName("id").Get(1))
		require.Equal(t, 2.5, got.GetColByName("amount").Get(1))
		require.Equal(t, `b"q`, got.GetColByName("label").Get(1))

		// A custom query uses the driver's own placeholder.
		q, err := ReadSQL(db, s.dialect, "", ReadSQLOptions{
			Query:  "SELECT label FROM " + quoteSQLIdent(string(s.dialect), table) + " WHERE id = " + s.placeholder,
			Params: []any{3},
		})
		require.NoError(t, err)
		require.Equal(t, 1, q.NumRows())
		require.Equal(t, "c`d", q.GetColByNumber(0).Get(0))

		chunks := 0
		for c, err := range ReadSQLStream(context.Background(), db, s.dialect, table, ReadSQLOptions{ChunkSize: 2}) {
			require.NoError(t, err)
			require.NotNil(t, c)
			chunks++
		}
		require.Equal(t, 2, chunks)

		err = dt.ToSQL(db, s.dialect, table)
		require.ErrorContains(t, err, "already exists")
	})
}

func TestSQLServerAppendAddsAColumn(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra_it_append"
		serverTable(t, s, db, table)
		require.NoError(t, NewDataTable(NewDataList(1).SetName("a")).ToSQL(db, s.dialect, table))
		more := NewDataTable(NewDataList(2).SetName("a"), NewDataList("x").SetName("b"))
		require.NoError(t, more.ToSQL(db, s.dialect, table, ToSQLOptions{IfExists: SQLActionIfTableExistsAppend}))
		got, err := ReadSQL(db, s.dialect, table, ReadSQLOptions{OrderBy: "a"})
		require.NoError(t, err)
		require.Equal(t, 2, got.NumRows())
		require.Nil(t, got.GetColByName("b").Get(0))
		require.Equal(t, "x", got.GetColByName("b").Get(1))
	})
}

func TestSQLServerReplace(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra_it_replace"
		serverTable(t, s, db, table)
		require.NoError(t, NewDataTable(NewDataList(1, 2, 3).SetName("id")).ToSQL(db, s.dialect, table))
		require.NoError(t, NewDataTable(NewDataList(7).SetName("id")).ToSQL(db, s.dialect, table,
			ToSQLOptions{IfExists: SQLActionIfTableExistsReplace, Schema: s.schema}))
		require.Equal(t, 1, serverCount(t, s, db, table))
		requireNoLeftoverTables(t, s, db)
	})
}

// A write that fails on the server, here an INSERT with more placeholders
// than the protocol allows, leaves the old table and its rows as they were.
func TestSQLServerReplaceKeepsOldTableWhenWriteFails(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		const table = "insyra_it_keep"
		serverTable(t, s, db, table)
		require.NoError(t, NewDataTable(NewDataList(1, 2, 3).SetName("id")).ToSQL(db, s.dialect, table))

		n := 70000
		vals := make([]any, n)
		for i := range vals {
			vals[i] = i
		}
		big := NewDataTable(NewDataList(vals...).SetName("id"))
		// The first batch fits, the second passes 65,535 placeholders.
		err := big.ToSQL(db, s.dialect, table, ToSQLOptions{IfExists: SQLActionIfTableExistsReplace, BatchSize: 66000})
		require.Error(t, err)
		require.Equal(t, 3, serverCount(t, s, db, table))
		requireNoLeftoverTables(t, s, db)
	})
}

func TestSQLServerReplaceRefusesAReferencedTableOnMySQL(t *testing.T) {
	eachSQLServer(t, func(t *testing.T, s sqlServer, db *sql.DB) {
		if s.dialect != SQLDialectMySQL {
			t.Skip("the staged replace runs on MySQL only")
		}
		serverTable(t, s, db, "insyra_it_child", "insyra_it_parent")
		_, err := db.Exec("CREATE TABLE insyra_it_parent (id INT PRIMARY KEY)")
		require.NoError(t, err)
		_, err = db.Exec("INSERT INTO insyra_it_parent VALUES (1)")
		require.NoError(t, err)
		_, err = db.Exec("CREATE TABLE insyra_it_child (pid INT, FOREIGN KEY (pid) REFERENCES insyra_it_parent(id))")
		require.NoError(t, err)
		err = NewDataTable(NewDataList(5).SetName("id")).ToSQL(db, s.dialect, "insyra_it_parent",
			ToSQLOptions{IfExists: SQLActionIfTableExistsReplace})
		require.ErrorContains(t, err, "referenced by a foreign key from insyra_it_child")
		require.Equal(t, 1, serverCount(t, s, db, "insyra_it_parent"))
	})
}

func requireNoLeftoverTables(t *testing.T, s sqlServer, db *sql.DB) {
	t.Helper()
	q := "SELECT table_name FROM information_schema.tables WHERE table_schema = " + bindVars(string(s.dialect), "?")
	rows, err := db.Query(q, s.schema)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		require.False(t, strings.HasPrefix(name, "insyra_new_") || strings.HasPrefix(name, "insyra_old_"), "left behind %s", name)
	}
	require.NoError(t, rows.Err())
}
