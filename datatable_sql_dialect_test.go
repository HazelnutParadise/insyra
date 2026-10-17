package insyra

import (
	"context"
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

	dt := NewDataTable(NewDataList(9).SetName("n"))
	err := dt.ToSQL(db, SQLDialect("oracle"), "t", ToSQLOptions{IfExists: SQLActionIfTableExistsReplace})
	require.ErrorContains(t, err, `"oracle"`)
	require.Equal(t, []int64{1}, queryInt64s(t, db, "SELECT n FROM t"))

	_, err = ReadSQL(db, SQLDialect("oracle"), "t")
	require.ErrorContains(t, err, `"oracle"`)

	yields := 0
	for table, err := range ReadSQLStream(context.Background(), db, SQLDialect(""), "t") {
		yields++
		require.Nil(t, table)
		require.ErrorContains(t, err, "dialect")
	}
	require.Equal(t, 1, yields)
}

func TestSQLRoundTripThroughDatabaseSQL(t *testing.T) {
	db := newTestSQLite(t)
	dt := NewDataTable(
		NewDataList(1, 2, 3).SetName("id"),
		NewDataList("a", "b", "c").SetName("label"),
	)
	require.NoError(t, dt.ToSQL(db, SQLDialectSQLite, "t"))
	got, err := ReadSQL(db, SQLDialectSQLite, "t", ReadSQLOptions{
		Query:  "SELECT id, label FROM t WHERE id = ?",
		Params: []any{2},
	})
	require.NoError(t, err)
	require.Equal(t, 1, got.NumRows())
	require.Equal(t, "b", got.GetColByName("label").Get(0))
}

// The point of taking *sql.DB: a program that uses DataTable does not link
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
