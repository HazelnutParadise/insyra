package insyra

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newFileSQLite opens a SQLite database in a file, so every connection the
// pool opens sees the same tables; an in-memory database is one per
// connection, and a cleanup that happened to get a second connection would
// look at an empty database. A busy timeout makes a statement wait for a lock
// the way MySQL waits for a metadata lock, instead of failing at once with
// SQLITE_BUSY while a cancelled transaction is still being rolled back, and
// WAL lets one connection write while another reads.
func newFileSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "replace.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	sqlDB, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// seedSales creates the table a Replace is about to overwrite.
func seedSales(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE sales (id INTEGER, amount REAL)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sales VALUES (1, 10), (2, 20), (3, 30)`).Error)
}

func newSalesTable() *DataTable {
	return NewDataTable(
		NewDataList(101, 102, 103, 104, 105).SetName("id"),
		NewDataList(1.5, 2.5, 3.5, 4.5, 5.5).SetName("amount"),
	)
}

func salesIDs(t *testing.T, db *gorm.DB, table string) []int64 {
	t.Helper()
	var ids []int64
	require.NoError(t, db.Raw(`SELECT id FROM "`+table+`" ORDER BY id`).Scan(&ids).Error)
	return ids
}

func tableNamesIn(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var names []string
	require.NoError(t, db.Raw(`SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`).Scan(&names).Error)
	return names
}

// failStatement makes the nth statement containing fragment fail before it
// reaches the database.
func failStatement(t *testing.T, db *gorm.DB, fragment string, nth int) {
	t.Helper()
	seen := 0
	require.NoError(t, db.Callback().Raw().Before("gorm:raw").Register("test:fail-"+fragment, func(d *gorm.DB) {
		if strings.Contains(d.Statement.SQL.String(), fragment) {
			seen++
			if seen == nth {
				_ = d.AddError(errors.New("forced failure"))
			}
		}
	}))
}

// The dialects with transactional DDL keep the old table because the whole
// replace runs in one transaction: a failed second batch rolls back the DROP.
func TestToSQLReplaceKeepsOldTableWhenWriteFails(t *testing.T) {
	db := newFileSQLite(t)
	seedSales(t, db)
	failStatement(t, db, "INSERT INTO", 2) // BatchSize 2: five rows are three INSERTs
	err := newSalesTable().ToSQL(db, "sales", ToSQLOptions{IfExists: SQLActionIfTableExistsReplace, BatchSize: 2})
	require.ErrorContains(t, err, "forced failure")
	require.Equal(t, []int64{1, 2, 3}, salesIDs(t, db, "sales"))
	require.Equal(t, []string{"sales"}, tableNamesIn(t, db))
}

// MySQL commits every DDL statement at once, so a transaction cannot bring a
// dropped table back. Replace there writes a staging table and renames it in.
// Run on SQLite through the same path, a write that fails on its second
// batch leaves the old table as it was and no staging table behind.
func TestToSQLReplaceByRenameKeepsOldTableWhenWriteFails(t *testing.T) {
	db := newFileSQLite(t)
	seedSales(t, db)
	failStatement(t, db, "INSERT INTO", 2)
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace, BatchSize: 2}, true)
	require.ErrorContains(t, err, "forced failure")
	require.Equal(t, []int64{1, 2, 3}, salesIDs(t, db, "sales"))
	require.Equal(t, []string{"sales"}, tableNamesIn(t, db))
}

func TestToSQLReplaceByRenameSwapsInTheNewTable(t *testing.T) {
	db := newFileSQLite(t)
	seedSales(t, db)
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace, BatchSize: 2}, true)
	require.NoError(t, err)
	require.Equal(t, []int64{101, 102, 103, 104, 105}, salesIDs(t, db, "sales"))
	require.Equal(t, []string{"sales"}, tableNamesIn(t, db))
}

// A table that does not exist yet is created the ordinary way.
func TestToSQLReplaceByRenameCreatesAMissingTable(t *testing.T) {
	db := newFileSQLite(t)
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace}, true)
	require.NoError(t, err)
	require.Equal(t, []int64{101, 102, 103, 104, 105}, salesIDs(t, db, "sales"))
	require.Equal(t, []string{"sales"}, tableNamesIn(t, db))
}

// Cancelling the write does not stop the cleanup: the staging table is
// removed with a context that outlives the caller's.
func TestToSQLReplaceByRenameCleansUpAfterCancel(t *testing.T) {
	db := newFileSQLite(t)
	seedSales(t, db)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.NoError(t, db.Callback().Raw().After("gorm:raw").Register("test:cancel", func(d *gorm.DB) {
		if strings.Contains(d.Statement.SQL.String(), "INSERT INTO") {
			cancel()
		}
	}))
	err := newSalesTable().toSQL(ctx, db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace, BatchSize: 2}, true)
	require.Error(t, err)
	require.Equal(t, []int64{1, 2, 3}, salesIDs(t, db, "sales"))
	require.Equal(t, []string{"sales"}, tableNamesIn(t, db))
}

func TestToSQLReplaceByRenameReportsAFailedSwap(t *testing.T) {
	db := newFileSQLite(t)
	seedSales(t, db)
	failStatement(t, db, "RENAME", 1)
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace}, true)
	require.ErrorContains(t, err, "forced failure")
	require.Equal(t, []int64{1, 2, 3}, salesIDs(t, db, "sales"))
	require.Equal(t, []string{"sales"}, tableNamesIn(t, db))
}

// Where the swap takes two statements, a failure in the second undoes the
// first, so the old table keeps its name.
func TestToSQLReplaceByRenameUndoesAHalfDoneSwap(t *testing.T) {
	db := newFileSQLite(t)
	seedSales(t, db)
	failStatement(t, db, "RENAME", 2)
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace}, true)
	require.ErrorContains(t, err, "forced failure")
	require.Equal(t, []int64{1, 2, 3}, salesIDs(t, db, "sales"))
	require.Equal(t, []string{"sales"}, tableNamesIn(t, db))
}

// A staging table that cannot be created, as when the account may not create
// tables in the database, stops the replace before anything changes, and the
// error says which privileges are needed.
func TestToSQLReplaceByRenameReportsAStagingTableItCannotCreate(t *testing.T) {
	db := newFileSQLite(t)
	seedSales(t, db)
	failStatement(t, db, "CREATE TABLE", 1)
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace}, true)
	require.ErrorContains(t, err, "forced failure")
	require.ErrorContains(t, err, "CREATE, INSERT, ALTER and DROP privileges")
	require.NotContains(t, err.Error(), "could not drop")
	require.Equal(t, []int64{1, 2, 3}, salesIDs(t, db, "sales"))
	require.Equal(t, []string{"sales"}, tableNamesIn(t, db))
}

// A table another table references by foreign key is refused before anything
// changes: the rename would move the reference to the old copy, which the
// drop then either fails on or leaves pointing at nothing.
func TestToSQLReplaceByRenameRefusesAReferencedTable(t *testing.T) {
	db := newFileSQLite(t)
	seedSales(t, db)
	require.NoError(t, db.Exec(`CREATE TABLE refunds (sale_id INTEGER REFERENCES sales(id))`).Error)
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace}, true)
	require.ErrorContains(t, err, "referenced by a foreign key from refunds")
	require.Equal(t, []int64{1, 2, 3}, salesIDs(t, db, "sales"))
	require.Equal(t, []string{"refunds", "sales"}, tableNamesIn(t, db))
}

// A reference from the table to itself moves with it and is dropped with it,
// so it does not stop the replace.
func TestToSQLReplaceByRenameAllowsASelfReference(t *testing.T) {
	db := newFileSQLite(t)
	require.NoError(t, db.Exec(`CREATE TABLE sales (id INTEGER PRIMARY KEY, parent INTEGER REFERENCES sales(id))`).Error)
	require.NoError(t, db.Exec(`INSERT INTO sales VALUES (1, NULL)`).Error)
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace}, true)
	require.NoError(t, err)
	require.Equal(t, []int64{101, 102, 103, 104, 105}, salesIDs(t, db, "sales"))
}

// A table that appears after the first look found none is not dropped: the
// create path refuses it instead of falling back to DROP, the step MySQL
// cannot take back.
func TestToSQLReplaceByRenameRefusesATableThatAppearsMeanwhile(t *testing.T) {
	db := newFileSQLite(t)
	looked := false
	require.NoError(t, db.Callback().Row().After("gorm:row").Register("test:appear", func(d *gorm.DB) {
		if !looked && strings.Contains(d.Statement.SQL.String(), "sqlite_master") {
			looked = true
			require.NoError(t, db.Exec(`CREATE TABLE sales (id INTEGER, amount REAL)`).Error)
			require.NoError(t, db.Exec(`INSERT INTO sales VALUES (7, 70)`).Error)
		}
	}))
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace}, true)
	require.ErrorContains(t, err, "already exists")
	require.Equal(t, []int64{7}, salesIDs(t, db, "sales"))
}

// After a rename that reported an error, the replace looks at which tables
// exist to tell whether it took effect.
func TestRenameSwapHappened(t *testing.T) {
	db := newFileSQLite(t)
	require.NoError(t, db.Exec(`CREATE TABLE insyra_new_x (a INTEGER)`).Error)
	happened, err := renameSwapHappened(db, "sqlite", "", "insyra_new_x", "insyra_old_x")
	require.NoError(t, err)
	require.False(t, happened, "the staging table is still there")
	require.NoError(t, db.Exec(`ALTER TABLE insyra_new_x RENAME TO insyra_old_x`).Error)
	happened, err = renameSwapHappened(db, "sqlite", "", "insyra_new_x", "insyra_old_x")
	require.NoError(t, err)
	require.True(t, happened, "the staging table is gone and the backup exists")
}

// When the new table is in place but the old one cannot be dropped, the call
// says so and names the table still holding the old rows.
func TestToSQLReplaceByRenameReportsTheLeftoverOldTable(t *testing.T) {
	db := newFileSQLite(t)
	seedSales(t, db)
	failStatement(t, db, "DROP TABLE", 1)
	err := newSalesTable().toSQL(context.Background(), db, "sales",
		ToSQLOptions{IfExists: SQLActionIfTableExistsReplace}, true)
	require.ErrorContains(t, err, "forced failure")
	require.Equal(t, []int64{101, 102, 103, 104, 105}, salesIDs(t, db, "sales"))
	names := tableNamesIn(t, db)
	require.Len(t, names, 2)
	var backup string
	for _, n := range names {
		if n != "sales" {
			backup = n
		}
	}
	require.True(t, strings.HasPrefix(backup, "insyra_old_"), "leftover table %q", backup)
	require.ErrorContains(t, err, backup)
	require.Equal(t, []int64{1, 2, 3}, salesIDs(t, db, backup))
}

// On MySQL the swap is one RENAME TABLE statement, which MySQL runs
// atomically: no other session sees a moment with no table under the name.
func TestRenameSwapStatementsForMySQL(t *testing.T) {
	require.Equal(t,
		[]string{"RENAME TABLE `shop`.`sales` TO `shop`.`insyra_old_x`, `shop`.`insyra_new_x` TO `shop`.`sales`"},
		renameSwapStatements("mysql", "shop", "sales", "insyra_new_x", "insyra_old_x"))
	require.Equal(t,
		[]string{"RENAME TABLE `sales` TO `insyra_old_x`, `insyra_new_x` TO `sales`"},
		renameSwapStatements("mysql", "", "sales", "insyra_new_x", "insyra_old_x"))
	require.Equal(t,
		[]string{`ALTER TABLE "sales" RENAME TO "insyra_old_x"`, `ALTER TABLE "insyra_new_x" RENAME TO "sales"`},
		renameSwapStatements("sqlite", "", "sales", "insyra_new_x", "insyra_old_x"))
}
