package insyra

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type ReadSQLOptions struct {
	RowNameColumn string // 指定哪個欄位作為行名稱，預設為 "row_name"
	Query         string // 自訂 SQL 查詢，如果提供則忽略 tableName
	Limit         int    // 限制讀取的行數
	Offset        int    // 讀取的起始位置
	WhereClause   string // WHERE 子句
	OrderBy       string // ORDER BY 子句
}

// ReadSQL 從資料庫讀取表格資料並轉換為 DataTable 物件
// 如果 options 中提供自訂 Query，則使用自訂查詢
// 否則使用 tableName 和其他選項來構建 SQL 查詢
func ReadSQL(db *gorm.DB, tableName string, options ...ReadSQLOptions) (*DataTable, error) {
	if db == nil {
		return nil, fmt.Errorf("db 參數不能為空")
	}

	// 設定默認選項
	var opts ReadSQLOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = ReadSQLOptions{
			RowNameColumn: "row_name",
			Limit:         0, // 0 表示不限制
			Offset:        0,
		}
	}

	// 確定使用的查詢語句
	var query string
	var params []interface{}

	if opts.Query != "" {
		// 使用自訂查詢
		query = opts.Query
	} else {
		// 檢查表格是否存在
		dialect := db.Name()
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
			return nil, fmt.Errorf("檢查表格 %s 存在時發生錯誤: %w", tableName, err)
		}

		if count == 0 {
			return nil, fmt.Errorf("表格 %s 不存在", tableName)
		}

		// 構建 SQL 查詢
		query = fmt.Sprintf("SELECT * FROM %s", tableName)
		if opts.WhereClause != "" {
			query += " WHERE " + opts.WhereClause
		}
		if opts.OrderBy != "" {
			query += " ORDER BY " + opts.OrderBy
		}
		if opts.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", opts.Limit)
		}
		if opts.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", opts.Offset)
		}
	}
	// 執行查詢
	LogDebug("core", "ReadSQL", "執行SQL查詢: %s", query)
	rows, err := db.Raw(query, params...).Rows()
	if err != nil {
		return nil, fmt.Errorf("執行查詢時發生錯誤: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// 獲取列名
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("獲取列名時發生錯誤: %w", err)
	}

	// 確定行名列的索引（如果有）
	rowNameColIndex := -1
	for i, colName := range columnNames {
		if strings.EqualFold(colName, opts.RowNameColumn) {
			rowNameColIndex = i
			break
		}
	}

	// 為每一列創建 DataList
	dataLists := make([]*DataList, len(columnNames))
	for i, colName := range columnNames {
		if i != rowNameColIndex || rowNameColIndex == -1 {
			dl := NewDataList()
			dl.SetName(colName) // 明確設置列名為資料庫中的列名
			dataLists[i] = dl
		}
	}
	// 處理行
	rowValues := make([]interface{}, len(columnNames))
	scanArgs := make([]interface{}, len(columnNames))
	for i := range rowValues {
		scanArgs[i] = &rowValues[i]
	}

	// 用來保存行名稱，之後再設置
	rowNames := make(map[int]string)
	var rowIndex = 0

	for rows.Next() {
		// 讀取數據
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("讀取行數據時發生錯誤: %w", err)
		}

		// 保存行名（如果有）
		if rowNameColIndex >= 0 {
			if rowValues[rowNameColIndex] != nil {
				rowName := fmt.Sprintf("%v", rowValues[rowNameColIndex])
				rowNames[rowIndex] = rowName
			}
		}

		// 將數據添加到相應的列
		for i, val := range rowValues {
			// 跳過行名列，因為它已經被處理了
			if i != rowNameColIndex || rowNameColIndex == -1 {
				if dataLists[i] != nil {
					// 對 SQL NULL 值的處理
					if val == nil {
						dataLists[i].Append(nil)
					} else {
						// 處理不同類型的數據
						switch v := val.(type) {
						case []byte:
							// 將 []byte 轉換為字符串
							dataLists[i].Append(string(v))
						default:
							dataLists[i].Append(v)
						}
					}
				}
			}
		}
		rowIndex++
	} // 檢查是否有任何迭代錯誤
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代行時發生錯誤: %w", err)
	}

	// 將列添加到 DataTable 中
	// 確保列顯示順序與資料庫表格相同
	dt := NewDataTable()
	validColumns := make([]*DataList, 0, len(dataLists))
	for i, dl := range dataLists {
		if i != rowNameColIndex || rowNameColIndex == -1 {
			if dl != nil && len(dl.data) > 0 {
				validColumns = append(validColumns, dl)
			}
		}
	}
	if len(validColumns) > 0 {
		dt.AppendCols(validColumns...)

		// 在這裡設置行名
		if rowNameColIndex >= 0 {
			for i, name := range rowNames {
				dt.SetRowNameByIndex(i, name)
			}
		}
	}

	return dt, nil
}
