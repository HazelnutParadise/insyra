package insyra

import (
	"fmt"
	"reflect"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"gorm.io/gorm"
)

type ToSQLOptions struct {
	IfExists    SQLActionIfTableExists // "fail", "replace", "append"
	RowNames    bool
	ColumnTypes map[string]string // 自訂型別
}

type SQLActionIfTableExists int

const (
	FailIfExists SQLActionIfTableExists = iota
	ReplaceIfExists
	AppendIfExists
)

func (dt *DataTable) ToSQL(db *gorm.DB, tableName string) error {
	if dt == nil {
		return fmt.Errorf("dt is nil")
	}

	var dataMap []map[string]any
	numRow, numCol := dt.Size()
	for i := range numRow {
		row := dt.GetRow(i)
		rowMap := make(map[string]any)
		for j := 0; j < numCol; j++ {
			colName := dt.columns[j].GetName()
			rowMap[colName] = row.Get(j)
		}
		dataMap = append(dataMap, rowMap)
	}
	if len(dataMap) == 0 {
		return fmt.Errorf("data is empty")
	}

	// 儲存資料到資料庫
	if err := saveDataMapToDB(db, tableName, dataMap); err != nil {
		return fmt.Errorf("failed to save data to DB: %w", err)
	}
	return nil
}

func saveDataMapToDB(db *gorm.DB, tableName string, data []map[string]interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("data is empty")
	}

	// 取得欄位與型別
	columnTypes := map[string]string{}
	for k, v := range data[0] {
		columnTypes[k] = inferSQLType(reflect.TypeOf(v), db.Dialector.Name())
	}

	// 產生建表語法
	colDefs := []string{}
	for col, typ := range columnTypes {
		colDefs = append(colDefs, fmt.Sprintf("%s %s", col, typ))
	}
	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", tableName, strings.Join(colDefs, ", "))
	if err := db.Exec(createSQL).Error; err != nil {
		return err
	}

	// 插入資料
	for _, row := range data {
		builder := sq.Insert(tableName).PlaceholderFormat(sq.Question)
		cols := []string{}
		vals := []interface{}{}
		for col, val := range row {
			cols = append(cols, col)
			vals = append(vals, val)
		}
		builder = builder.Columns(cols...).Values(vals...)

		sqlStr, args, err := builder.ToSql()
		if err != nil {
			return err
		}

		if err := db.Exec(sqlStr, args...).Error; err != nil {
			return err
		}
	}

	return nil
}

func inferSQLType(t reflect.Type, dialect string) string {
	kind := t.Kind()
	switch dialect {
	case "mysql":
		switch kind {
		case reflect.Int, reflect.Int64:
			return "BIGINT"
		case reflect.Float64:
			return "DOUBLE"
		case reflect.Bool:
			return "BOOLEAN"
		case reflect.String:
			return "VARCHAR(255)"
		default:
			return "TEXT"
		}
	case "postgres":
		switch kind {
		case reflect.Int, reflect.Int64:
			return "INTEGER"
		case reflect.Float64:
			return "DOUBLE PRECISION"
		case reflect.Bool:
			return "BOOLEAN"
		case reflect.String:
			return "TEXT"
		default:
			return "TEXT"
		}
	default: // SQLite
		switch kind {
		case reflect.Int, reflect.Int64:
			return "INTEGER"
		case reflect.Float64:
			return "REAL"
		case reflect.Bool:
			return "BOOLEAN"
		case reflect.String:
			return "TEXT"
		default:
			return "TEXT"
		}
	}
}
