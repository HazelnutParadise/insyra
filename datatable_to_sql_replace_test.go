package insyra

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToSQLReplaceKeepsSQLiteTransactionalBehavior(t *testing.T) {
	db := newTestSQLite(t)
	require.NoError(t, db.Exec("CREATE TABLE sales (id INTEGER, amount REAL)").Error)
	require.NoError(t, db.Exec("INSERT INTO sales VALUES (1, 10), (2, 20)").Error)
	dt := NewDataTable(
		NewDataList(101, 102).SetName("id"),
		NewDataList(1.5, 2.5).SetName("amount"),
	)
	require.NoError(t, dt.ToSQL(db, "sales", ToSQLOptions{IfExists: SQLActionIfTableExistsReplace}))
	var ids []int
	require.NoError(t, db.Raw("SELECT id FROM sales ORDER BY id").Scan(&ids).Error)
	require.Equal(t, []int{101, 102}, ids)
	var count int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name LIKE 'insyra_%'").Scan(&count).Error)
	require.Zero(t, count, "SQLite should keep the existing transactional replacement path")
}
