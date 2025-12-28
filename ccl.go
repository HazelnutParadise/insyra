package insyra

import (
	"fmt"
	"reflect"

	"github.com/HazelnutParadise/insyra/internal/ccl"
)

func resetCCLEvalDepth() {
	// Reset the global evaluation depth variable to 0
	ccl.ResetEvalDepth()
}

func resetCCLFuncCallDepth() {
	// Reset the global function call depth variable to 0
	ccl.ResetFuncCallDepth()
}

// dataTableContext implements ccl.Context for DataTable
type dataTableContext struct {
	row        []any
	tableData  [][]any
	rowNameMap map[string]int
	colNameMap map[string]int
	rowIndex   int
}

func (c *dataTableContext) GetCol(index int) any {
	if index >= len(c.row) {
		return nil
	}
	return c.row[index]
}

func (c *dataTableContext) GetColByName(name string) (any, error) {
	if c.colNameMap == nil {
		return nil, fmt.Errorf("column name reference ['%s'] used but no column name mapping provided", name)
	}
	idx, ok := c.colNameMap[name]
	if !ok {
		return nil, fmt.Errorf("column name '%s' not found", name)
	}
	if idx >= len(c.row) {
		return nil, fmt.Errorf("column ['%s'] (index %d) out of range", name, idx)
	}
	return c.row[idx], nil
}

func (c *dataTableContext) GetRowIndex() int {
	return c.rowIndex
}

func (c *dataTableContext) GetCurrentRow() any {
	return c.row
}

func (c *dataTableContext) GetCell(colIndex, rowIndex int) (any, error) {
	if colIndex < 0 || colIndex >= len(c.tableData) {
		return nil, fmt.Errorf("column index %d out of range", colIndex)
	}
	if rowIndex < 0 || rowIndex >= len(c.tableData[colIndex]) {
		return nil, fmt.Errorf("row index %d out of range for column %d", rowIndex, colIndex)
	}
	return c.tableData[colIndex][rowIndex], nil
}

func (c *dataTableContext) GetCellByName(colName string, rowIndex int) (any, error) {
	if c.colNameMap == nil {
		return nil, fmt.Errorf("column name reference used but no column name mapping provided")
	}
	idx, ok := c.colNameMap[colName]
	if !ok {
		return nil, fmt.Errorf("column name '%s' not found", colName)
	}
	return c.GetCell(idx, rowIndex)
}

func (c *dataTableContext) GetRowAt(rowIndex int) (any, error) {
	if rowIndex < 0 {
		return nil, fmt.Errorf("row index %d out of range", rowIndex)
	}
	// Construct row from tableData
	row := make([]any, len(c.tableData))
	for i := range c.tableData {
		if rowIndex < len(c.tableData[i]) {
			row[i] = c.tableData[i][rowIndex]
		} else {
			row[i] = nil
		}
	}
	return row, nil
}

func (c *dataTableContext) GetRowIndexByName(rowName string) (int, error) {
	if c.rowNameMap == nil {
		return -1, fmt.Errorf("row name '%s' used but no row name mapping provided", rowName)
	}
	idx, ok := c.rowNameMap[rowName]
	if !ok {
		return -1, fmt.Errorf("row name '%s' not found", rowName)
	}
	return idx, nil
}

func (c *dataTableContext) GetColIndexByName(colName string) (int, error) {
	if c.colNameMap == nil {
		return -1, fmt.Errorf("column name reference used but no column name mapping provided")
	}
	idx, ok := c.colNameMap[colName]
	if !ok {
		return -1, fmt.Errorf("column name '%s' not found", colName)
	}
	return idx, nil
}

func (c *dataTableContext) GetColData(index int) ([]any, error) {
	if index < 0 || index >= len(c.tableData) {
		return nil, fmt.Errorf("column index %d out of range", index)
	}
	// Return a copy to avoid external modification?
	// Or just return the slice. ccl_evaluator used to copy.
	// Let's return a copy to be safe and consistent with previous behavior.
	res := make([]any, len(c.tableData[index]))
	copy(res, c.tableData[index])
	return res, nil
}

func (c *dataTableContext) GetColDataByName(name string) ([]any, error) {
	if c.colNameMap == nil {
		return nil, fmt.Errorf("column name reference used but no column name mapping provided")
	}
	idx, ok := c.colNameMap[name]
	if !ok {
		return nil, fmt.Errorf("column name '%s' not found", name)
	}
	return c.GetColData(idx)
}

func (c *dataTableContext) GetRowCount() int {
	if len(c.tableData) == 0 {
		return 0
	}
	return len(c.tableData[0])
}

func (c *dataTableContext) SetRowIndex(index int) error {
	if index < 0 {
		return fmt.Errorf("row index cannot be negative")
	}
	// We don't strictly check upper bound here to allow setting index for next iteration check?
	// But usually we should check.
	// Let's check against max row count if possible, but tableData might be jagged?
	// Assuming rectangular for now based on GetRowCount.
	if len(c.tableData) > 0 && index >= len(c.tableData[0]) {
		return fmt.Errorf("row index %d out of range", index)
	}

	c.rowIndex = index
	// Also update the row slice cache
	for j := range len(c.tableData) {
		if index < len(c.tableData[j]) {
			c.row[j] = c.tableData[j][index]
		} else {
			c.row[j] = nil
		}
	}
	return nil
}

func (c *dataTableContext) GetAllData() ([]any, error) {
	var allData []any
	// Estimate capacity to avoid frequent reallocations
	totalSize := 0
	if len(c.tableData) > 0 {
		totalSize = len(c.tableData) * len(c.tableData[0])
	}
	allData = make([]any, 0, totalSize)

	for _, col := range c.tableData {
		allData = append(allData, col...)
	}
	return allData, nil
}

// applyCCLOnDataTable evaluates the expression on each row of a DataTable.
// Optimized: compiles expression once and reuses AST for all rows.
func applyCCLOnDataTable(table *DataTable, expression string) ([]any, error) {
	var result []any
	var err error

	// 預先編譯表達式（只做一次 tokenize + parse）
	ast, err := ccl.CompileExpression(expression)
	if err != nil {
		return nil, err
	}

	table.AtomicDo(func(table *DataTable) {
		numRow, numCol := table.getMaxColLength(), len(table.columns)
		result = make([]any, numRow)

		// 建立欄位名稱到索引的映射（支援 ['colName'] 語法）
		colNameMap := make(map[string]int, numCol)
		for j := range numCol {
			if table.columns[j].name != "" {
				colNameMap[table.columns[j].name] = j
			}
		}

		// Bind AST to table columns to resolve indices at compile time
		boundAST, err2 := ccl.Bind(ast, colNameMap)
		if err2 != nil {
			err = err2
			return
		}

		// 準備 tableData 和 rowNameMap 以支援 . 運算符
		tableData := make([][]any, numCol)
		for j := range numCol {
			tableData[j] = make([]any, len(table.columns[j].data))
			copy(tableData[j], table.columns[j].data)
		}
		rowNameMap := table.rowNames

		// 預分配 row slice，避免每行都重新分配
		row := make([]any, numCol)

		// Create context
		ctx := &dataTableContext{
			row:        row,
			tableData:  tableData,
			rowNameMap: rowNameMap,
			colNameMap: colNameMap,
		}

		if ccl.IsRowDependent(ccl.GetExpressionNode(boundAST)) {
			for i := range numRow {
				// 填充第 i 行的資料（重用 row slice）
				for j := range numCol {
					if i < len(table.columns[j].data) {
						row[j] = table.columns[j].data[i]
					} else {
						row[j] = nil
					}
				}
				// Update context
				ctx.rowIndex = i

				// 直接使用預編譯且綁定的 AST
				val, err2 := ccl.Evaluate(boundAST, ctx)
				if err2 != nil {
					err = err2
					return
				}
				result[i] = val
			}
		} else {
			// 非行依賴表達式，只需計算一次
			for j := range numCol {
				if len(table.columns[j].data) > 0 {
					row[j] = table.columns[j].data[0]
				} else {
					row[j] = nil
				}
			}
			ctx.rowIndex = 0
			val, err2 := ccl.Evaluate(boundAST, ctx)
			if err2 != nil {
				err = err2
				return
			}

			if rv := reflect.ValueOf(val); val != nil && rv.Kind() == reflect.Slice && rv.Len() == numRow {
				result = make([]any, rv.Len())
				for i := 0; i < rv.Len(); i++ {
					result[i] = rv.Index(i).Interface()
				}
			} else {
				for i := range numRow {
					result[i] = val
				}
			}
		}
	})
	return result, err
}

// InitCCLFunctions registers default functions for use with CCL.
func initCCLFunctions() {
	ccl.RegisterStandardFunctions()
}
