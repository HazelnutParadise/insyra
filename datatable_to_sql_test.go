package insyra

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	_ "modernc.org/sqlite"
)

func TestToSQL_TransactionRollbackOnInsertError(t *testing.T) {
	// 使用 pure-Go sqlite (modernc) in-memory 測試 transaction rollback (無需 cgo)
	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer func() { _ = sqlDB.Close() }()

	db, err := gorm.Open(sqlite.New(sqlite.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)

	// 建立表格，並加上 CHECK constraint 使插入會失敗
	require.NoError(t, db.Exec("CREATE TABLE test_tx (id INTEGER PRIMARY KEY AUTOINCREMENT, value INTEGER CHECK(value > 0));").Error)

	// 建立 DataTable，帶入違反約束的值
	valueCol := NewDataList(-1).SetName("value")
	dt := NewDataTable(valueCol)

	// 使用 AppendIfExists，因為表格已存在
	err = dt.ToSQL(db, "test_tx", ToSQLOptions{IfExists: SQLActionIfTableExistsAppend})
	require.Error(t, err, "expected ToSQL to return an error due to CHECK constraint violation")

	// 檢查是否沒有插入任何資料（即 transaction 已回滾）
	var cnt int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM test_tx;").Scan(&cnt).Error)
	require.Equal(t, int64(0), cnt, "expected zero rows after rollback")
}
