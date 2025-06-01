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

func (dt *DataTable) ToSQL(db *gorm.DB, tableName string, options ...ToSQLOptions) error {
	if dt == nil {
		return fmt.Errorf("dt is nil")
	}

	// 設定默認選項
	var opts ToSQLOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = ToSQLOptions{
			IfExists:    FailIfExists,
			RowNames:    false,
			ColumnTypes: make(map[string]string),
		}
	}
	var dataMap []map[string]any
	numRow, numCol := dt.Size()
	for i := range numRow {
		row := dt.GetRow(i)
		rowMap := make(map[string]any)

		// 如果啟用行名稱，則將其添加到資料中
		if opts.RowNames {
			rowName := dt.GetRowNameByIndex(i)
			if rowName != "" {
				rowMap["row_name"] = rowName
			}
		}

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
	if err := saveDataMapToDB(db, tableName, dataMap, opts); err != nil {
		return fmt.Errorf("failed to save data to DB: %w", err)
	}
	return nil
}

func saveDataMapToDB(db *gorm.DB, tableName string, data []map[string]interface{}, opts ToSQLOptions) error {
	if len(data) == 0 {
		return fmt.Errorf("data is empty")
	}

	// 取得欄位與型別
	columnTypes := map[string]string{}

	// 使用自訂欄位型別（如果有提供）
	if len(opts.ColumnTypes) > 0 {
		columnTypes = opts.ColumnTypes
	} else {
		// 自動推斷型別
		for k, v := range data[0] {
			columnTypes[k] = inferSQLType(reflect.TypeOf(v), db.Dialector.Name())
		}
	}
	// 處理表格已存在的情況
	var tableExists bool
	dialect := db.Dialector.Name()

	// 根據數據庫類型選擇檢查表格是否存在的查詢
	var result *gorm.DB
	var count int

	switch dialect {
	case "mysql":
		result = db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = (SELECT DATABASE())", tableName)
	case "postgres":
		result = db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ? AND table_schema = current_schema()", tableName)
	case "sqlite":
		// SQLite 特有的查詢方式
		result = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName)
	default:
		// 未知數據庫，使用一般性方法
		result = db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName)
	}

	if err := result.Scan(&count).Error; err != nil {
		// 如果無法檢查表格存在，假設表格不存在
		tableExists = false
	} else {
		tableExists = count > 0
	}

	if tableExists {
		switch opts.IfExists {
		case FailIfExists:
			return fmt.Errorf("table %s already exists", tableName)
		case ReplaceIfExists:
			// 刪除現有表格
			if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)).Error; err != nil {
				return err
			}
			tableExists = false
		case AppendIfExists:
			// 保留現有表格，繼續執行
		}
	} // 如果表格不存在，則建立表格
	if !tableExists {
		// 確保啟用行名稱時，row_name 欄位存在
		if opts.RowNames {
			columnTypes["row_name"] = "TEXT"
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
	} else if opts.IfExists == AppendIfExists {
		// 檢查現有表格是否需要新增欄位
		var existingCols []string

		// 根據數據庫方言選擇正確的查詢方式
		var columnsQuery string
		var args []interface{}

		switch dialect {
		case "mysql":
			columnsQuery = "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND TABLE_SCHEMA = (SELECT DATABASE());"
			args = []interface{}{tableName}
		case "postgres":
			columnsQuery = "SELECT column_name FROM information_schema.columns WHERE table_name = ? AND table_schema = current_schema();"
			args = []interface{}{tableName}
		case "sqlite":
			columnsQuery = fmt.Sprintf("PRAGMA table_info(%s);", tableName)
			rows, err := db.Raw(columnsQuery).Rows()
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var cid int
					var name string
					var type_name string
					var notnull int
					var dflt_value *string
					var pk int
					if err := rows.Scan(&cid, &name, &type_name, &notnull, &dflt_value, &pk); err == nil {
						existingCols = append(existingCols, name)
					}
				}
			}
		default:
			// 對於未知數據庫，假設是SQLite
			columnsQuery = fmt.Sprintf("PRAGMA table_info(%s);", tableName)
			rows, err := db.Raw(columnsQuery).Rows()
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var cid int
					var name string
					var type_name string
					var notnull int
					var dflt_value *string
					var pk int
					if err := rows.Scan(&cid, &name, &type_name, &notnull, &dflt_value, &pk); err == nil {
						existingCols = append(existingCols, name)
					}
				}
			}
		}

		// 如果是MySQL或PostgreSQL，需要執行查詢獲取列名
		if dialect == "mysql" || dialect == "postgres" {
			rows, err := db.Raw(columnsQuery, args...).Rows()
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var name string
					if err := rows.Scan(&name); err == nil {
						existingCols = append(existingCols, name)
					}
				}
			}
		}
		// 檢查每個欄位是否存在，如果不存在則添加
		existingColsMap := make(map[string]bool)
		for _, col := range existingCols {
			existingColsMap[strings.ToLower(col)] = true
		}

		// 如果啟用了行名稱功能，確保 row_name 欄位存在
		if opts.RowNames && !existingColsMap["row_name"] {
			alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tableName, "row_name", "TEXT")
			if err := db.Exec(alterSQL).Error; err != nil {
				LogWarning("DataTable", "ToSQL", "Failed to add column row_name to table %s: %v", tableName, err)
			} else {
				LogInfo("DataTable", "ToSQL", "Added column row_name to table %s", tableName)
				existingColsMap["row_name"] = true
			}
		}

		for col, typ := range columnTypes {
			if !existingColsMap[strings.ToLower(col)] {
				// 添加新列
				alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tableName, col, typ)
				if err := db.Exec(alterSQL).Error; err != nil {
					LogWarning("DataTable", "ToSQL", "Failed to add column %s to table %s: %v", col, tableName, err)
				} else {
					LogInfo("DataTable", "ToSQL", "Added column %s to table %s", col, tableName)
				}
			}
		}
	}
	// 插入資料
	for _, row := range data {
		// 檢查此行的欄位是否都存在於資料表中
		// 如果表格是使用AppendIfExists模式，且發現需要添加新欄位，則重新檢查表格結構
		if opts.IfExists == AppendIfExists {
			// 獲取當前表格的列
			var existingCols []string
			switch dialect {
			case "mysql":
				rows, err := db.Raw("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND TABLE_SCHEMA = (SELECT DATABASE());", tableName).Rows()
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var name string
						if err := rows.Scan(&name); err == nil {
							existingCols = append(existingCols, strings.ToLower(name))
						}
					}
				}
			case "postgres":
				rows, err := db.Raw("SELECT column_name FROM information_schema.columns WHERE table_name = ? AND table_schema = current_schema();", tableName).Rows()
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var name string
						if err := rows.Scan(&name); err == nil {
							existingCols = append(existingCols, strings.ToLower(name))
						}
					}
				}
			default: // sqlite
				rows, err := db.Raw(fmt.Sprintf("PRAGMA table_info(%s);", tableName)).Rows()
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var cid int
						var name string
						var type_name string
						var notnull int
						var dflt_value *string
						var pk int
						if err := rows.Scan(&cid, &name, &type_name, &notnull, &dflt_value, &pk); err == nil {
							existingCols = append(existingCols, strings.ToLower(name))
						}
					}
				}
			}

			// 將現有列轉換為 map 以便快速查找
			existingColsMap := make(map[string]bool)
			for _, col := range existingCols {
				existingColsMap[strings.ToLower(col)] = true
			}

			// 檢查此行的每個列是否存在
			missingCols := []string{}
			for col := range row {
				if !existingColsMap[strings.ToLower(col)] {
					missingCols = append(missingCols, col)
				}
			}

			// 如果有缺失的列，則添加它們
			for _, col := range missingCols {
				// 根據值推斷類型
				value := row[col]
				typ := inferSQLType(reflect.TypeOf(value), dialect)

				// 添加新列
				alterSQL := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s;", tableName, col, typ)
				if err := db.Exec(alterSQL).Error; err != nil {
					LogWarning("Failed to add column %s to table %s: %v", col, tableName, err)
				} else {
					LogInfo("Added column %s to table %s", col, tableName)
					// 將此列添加到存在列映射中
					existingColsMap[strings.ToLower(col)] = true
				}
			}
		}

		// 只包含表格中存在的列
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
