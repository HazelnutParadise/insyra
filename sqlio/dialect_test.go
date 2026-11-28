package sqlio

import (
	"context"
	"github.com/HazelnutParadise/insyra"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The dialect has no default: one the functions do not know is refused
// before any statement runs, instead of being read as SQLite.
func TestSQLRefusesAnUnknownDialect(t *testing.T) {
	db := newTestSQLite(t)
	execSQL(t, db, "CREATE TABLE t (n INTEGER)")
	execSQL(t, db, "INSERT INTO t VALUES (1)")

	dt := insyra.NewDataTable(insyra.NewDataList(9).SetName("n"))
	err := Write(context.Background(), dt, db, Dialect("oracle"), "t", WriteOptions{IfTableExists: TableExistsReplace})
	require.ErrorContains(t, err, `"oracle"`)
	require.Equal(t, []int64{1}, queryInt64s(t, db, "SELECT n FROM t"))

	_, err = Read(context.Background(), db, Dialect("oracle"), "t")
	require.ErrorContains(t, err, `"oracle"`)

	yields := 0
	for table, err := range Stream(context.Background(), db, Dialect(""), "t", 1000) {
		yields++
		require.Nil(t, table)
		require.ErrorContains(t, err, "dialect")
	}
	require.Equal(t, 1, yields)
}

func TestSQLRoundTripThroughDatabaseSQL(t *testing.T) {
	db := newTestSQLite(t)
	dt := insyra.NewDataTable(
		insyra.NewDataList(1, 2, 3).SetName("id"),
		insyra.NewDataList("a", "b", "c").SetName("label"),
	)
	require.NoError(t, Write(context.Background(), dt, db, SQLite, "t"))
	got, err := Query(context.Background(), db, SQLite, "SELECT id, label FROM t WHERE id = ?", QueryOptions{
		Params: []any{2},
	})
	require.NoError(t, err)
	require.Equal(t, 1, got.NumRows())
	require.Equal(t, "b", got.GetColByName("label").Get(0))
}

// The point of taking *sql.DB: a program that uses insyra.DataTable does not link
// gorm or the query builder the gorm version needed.
func TestRootPackageDoesNotLinkGorm(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command not available")
	}
	out, err := exec.Command(goBin, "list", "-deps", ".").Output()
	require.NoError(t, err)
	for _, pkg := range strings.Fields(string(out)) {
		require.False(t, strings.HasPrefix(pkg, "gorm.io/"), "root package depends on %s", pkg)
		require.False(t, strings.HasPrefix(pkg, "github.com/Masterminds/squirrel"), "root package depends on %s", pkg)
	}
}

// A column with no name is written under its Excel-style letter, the name
// insyra already uses for it, because MySQL and PostgreSQL refuse an empty
// identifier.
func TestToSQLWritesAnUnnamedColumnUnderItsLetter(t *testing.T) {
	db := newTestSQLite(t)
	dt := insyra.NewDataTable(insyra.NewDataList(1, 2), insyra.NewDataList("x", "y").SetName("label"), insyra.NewDataList(3.5, 4.5))
	require.NoError(t, Write(context.Background(), dt, db, SQLite, "t"))
	got, err := Read(context.Background(), db, SQLite, "t")
	require.NoError(t, err)
	require.Equal(t, []string{"A", "label", "C"}, got.ColNames())
	require.EqualValues(t, 2, got.GetColByName("A").Get(1))
	require.Equal(t, 4.5, got.GetColByName("C").Get(1))
}

// The letter must not land on another column's name: the write is refused
// before anything is created.
func TestToSQLRefusesAnUnnamedColumnWhoseLetterIsTaken(t *testing.T) {
	db := newTestSQLite(t)
	dt := insyra.NewDataTable(insyra.NewDataList(1), insyra.NewDataList(2).SetName("a"))
	err := Write(context.Background(), dt, db, SQLite, "t")
	require.ErrorContains(t, err, "column A has no name")
	var n int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE name = 't'").Scan(&n))
	require.Zero(t, n)
}
