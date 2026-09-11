package insyra

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/internal/ccl"
	"github.com/HazelnutParadise/insyra/internal/core"
)

func (dt *DataTable) AddColUsingCCL(newColName, cclFormula string) (result *DataTable) {
	result = dt
	// 添加 recover 以防止程序崩潰
	defer func() {
		if r := recover(); r != nil {
			dt.fail("AddColUsingCCL", "Panic recovered: %v", r)
		}
	}()

	dt.AtomicDo(func(dt *DataTable) {
		// 優先記錄表達式開始評估的時間
		startTime := time.Now()
		LogDebug("DataTable", "AddColUsingCCL", "Starting CCL evaluation for %s: %s", newColName, cclFormula)

		result, err := applyCCLOnDataTable(dt, cclFormula)
		if err != nil {
			// The error already says whether it could not compile or failed on
			// a particular row; an elapsed time adds nothing to either.
			dt.failErr("AddColUsingCCL", err)
		} else {
			elapsed := time.Since(startTime)
			LogDebug("DataTable", "AddColUsingCCL", "CCL evaluation completed in %v", elapsed)
			// fmt.Printf("DEBUG: AddColUsingCCL result[0]: %v (type %T)\n", result[0], result[0])
			// 使用 NewDataList(result...) 會展開 slice，如果 result 本身就是我們想要的資料，
			// 且我們不希望它被進一步展開（例如 result 已經是 []any），
			// 我們應該直接將其包裝在一個 slice 中傳遞給 NewDataList，
			// 或者直接手動創建 DataList。
			// 這裡我們選擇將 result 作為單個參數傳遞，以避免 NewDataList 展開它。
			// 但 applyCCLOnDataTable 返回的是 []any，代表整欄的資料。
			// 如果我們直接傳遞 result...，NewDataList 會展開它。
			// 如果 result 裡面包含 slice，NewDataList 會遞迴展開。

			// 解決方案：直接手動創建 DataList 並設置數據，繞過 NewDataList 的自動展開邏輯。
			newCol := &DataList{
				data: result,
				name: newColName,
			}
			timestamp := time.Now().Unix()
			newCol.creationTimestamp = timestamp
			newCol.lastModifiedTimestamp.Store(timestamp)
			dt.AppendCols(newCol)
		}
	})
	return result
}

// EditColByIndexUsingCCL modifies an existing column at the specified index using a CCL expression.
// The column index uses Excel-style letters (A, B, C, ..., AA, AB, etc.) where A = first column.
// Returns the modified DataTable.
func (dt *DataTable) EditColByIndexUsingCCL(colIndex, cclFormula string) (result *DataTable) {
	result = dt
	defer func() {
		if r := recover(); r != nil {
			dt.fail("EditColByIndexUsingCCL", "Panic recovered: %v", r)
		}
	}()

	dt.AtomicDo(func(dt *DataTable) {
		startTime := time.Now()
		LogDebug("DataTable", "EditColByIndexUsingCCL", "Starting CCL evaluation for column %s: %s", colIndex, cclFormula)

		// 解析欄位索引
		targetColIdx, ok := ParseColIndex(colIndex)
		if !ok || targetColIdx < 0 || targetColIdx >= len(dt.columns) {
			dt.fail("EditColByIndexUsingCCL", "Column index '%s' out of range", colIndex)
			return
		}

		result, err := applyCCLOnDataTable(dt, cclFormula)
		if err != nil {
			// The error already says whether it could not compile or failed on
			// a particular row; an elapsed time adds nothing to either.
			dt.failErr("EditColByIndexUsingCCL", err)
		} else {
			elapsed := time.Since(startTime)
			LogDebug("DataTable", "EditColByIndexUsingCCL", "CCL evaluation completed in %v", elapsed)
			dt.columns[targetColIdx].data = result
		}
	})
	return result
}

// EditColByNameUsingCCL modifies an existing column with the specified name using a CCL expression.
// Returns the modified DataTable. If the column name is not found, a warning is logged.
func (dt *DataTable) EditColByNameUsingCCL(colName, cclFormula string) (result *DataTable) {
	result = dt
	defer func() {
		if r := recover(); r != nil {
			dt.fail("EditColByNameUsingCCL", "Panic recovered: %v", r)
		}
	}()

	dt.AtomicDo(func(dt *DataTable) {
		startTime := time.Now()
		LogDebug("DataTable", "EditColByNameUsingCCL", "Starting CCL evaluation for column '%s': %s", colName, cclFormula)

		// 查找欄位索引
		targetColIdx := -1
		for i, col := range dt.columns {
			if col.name == colName {
				targetColIdx = i
				break
			}
		}

		if targetColIdx < 0 {
			dt.fail("EditColByNameUsingCCL", "Column '%s' not found", colName)
			return
		}

		result, err := applyCCLOnDataTable(dt, cclFormula)
		if err != nil {
			// The error already says whether it could not compile or failed on
			// a particular row; an elapsed time adds nothing to either.
			dt.failErr("EditColByNameUsingCCL", err)
		} else {
			elapsed := time.Since(startTime)
			LogDebug("DataTable", "EditColByNameUsingCCL", "CCL evaluation completed in %v", elapsed)
			dt.columns[targetColIdx].data = result
		}
	})
	return result
}

// ExecuteCCL executes multi-line CCL statements on the DataTable.
// It supports assignment syntax (e.g., A=B+C) and NEW('colName', expr) for creating new columns.
// Multiple statements can be separated by ; or newline.
// Assignment operations modify existing columns; if the target column doesn't exist, an error is returned.
// Returns the modified DataTable.
func (dt *DataTable) ExecuteCCL(cclStatements string) (result *DataTable) {
	result = dt
	// 添加 recover 以防止程序崩潰
	defer func() {
		if r := recover(); r != nil {
			dt.fail("ExecuteCCL", "Panic recovered: %v", r)
		}
	}()

	dt.AtomicDo(func(dt *DataTable) {
		startTime := time.Now()
		LogDebug("DataTable", "ExecuteCCL", "Starting CCL execution: %s", cclStatements)

		// 編譯多行 CCL 語句，並保留每條語句的原文，讓執行期錯誤指得出是哪一條
		stmts, err := ccl.CompileMultilineStatements(cclStatements)
		if err != nil {
			dt.failErr("ExecuteCCL", err)
			return
		}

		// Run the whole script against a private working copy and write it
		// back only when every statement succeeds. Applying the statements to
		// dt one at a time left the first n-1 in place when statement n
		// failed, and nothing told the caller the table was half-changed.
		work := dt.cclWorkingCopy()
		numRow, numCol := work.getMaxColLength(), len(work.columns)

		colNameMap := make(map[string]int, numCol)
		for j := range numCol {
			if work.columns[j].name != "" {
				colNameMap[work.columns[j].name] = j
			}
		}

		// Every statement sees a snapshot of the table as the previous
		// statements left it.
		tableData := make([][]any, len(work.columns))
		for j := range len(work.columns) {
			tableData[j] = make([]any, len(work.columns[j].data))
			copy(tableData[j], work.columns[j].data)
		}
		rowNameMap := work.rowNames

		for _, stmt := range stmts {
			if err := executeCCLNode(work, stmt.Node, numRow, colNameMap, tableData, rowNameMap); err != nil {
				dt.failErr("ExecuteCCL", ccl.AttachExpr(stmt.Src, err))
				return
			}
			numCol = len(work.columns)
			for j := range numCol {
				if work.columns[j].name != "" {
					colNameMap[work.columns[j].name] = j
				}
			}
			tableData = make([][]any, len(work.columns))
			for j := range len(work.columns) {
				tableData[j] = make([]any, len(work.columns[j].data))
				copy(tableData[j], work.columns[j].data)
			}
			rowNameMap = work.rowNames
		}

		dt.commitCCLWorkingCopy(work)

		elapsed := time.Since(startTime)
		LogDebug("DataTable", "ExecuteCCL", "CCL execution completed in %v", elapsed)
	})
	return result
}

// executeCCLNode executes a single CCL node on the DataTable
func executeCCLNode(dt *DataTable, node ccl.CCLNode, numRow int, colNameMap map[string]int, tableData [][]any, rowNameMap *core.BiIndex) error {
	// 檢查是否為賦值語句
	if ccl.IsAssignmentNode(node) {
		target, _ := ccl.GetAssignmentTarget(node)
		return executeAssignment(dt, node, target, numRow, colNameMap, tableData, rowNameMap)
	}

	// 檢查是否為 NEW 函數
	if ccl.IsNewColNode(node) {
		newColName, _, _ := ccl.GetNewColInfo(node)
		return executeNewColumn(dt, node, newColName, numRow, colNameMap, tableData, rowNameMap)
	}

	// 普通表達式，不做任何操作（只有在 slice 模式下才有意義）
	return nil
}

// executeAssignment executes an assignment CCL statement
func executeAssignment(dt *DataTable, node ccl.CCLNode, target string, numRow int, colNameMap map[string]int, tableData [][]any, rowNameMap *core.BiIndex) error {
	// 確定目標列索引
	var targetColIdx int

	if strings.HasPrefix(target, "'") && strings.HasSuffix(target, "'") {
		// 欄位名稱形式 'colName'
		colName := target[1 : len(target)-1]
		idx, ok := colNameMap[colName]
		if !ok {
			return fmt.Errorf("assignment target column '%s' does not exist", colName)
		}
		targetColIdx = idx
	} else {
		// 欄位索引形式 A, B, C 或直接是欄位名稱
		// 先嘗試作為索引解析
		idx, ok := ParseColIndex(target)
		if ok && idx >= 0 && idx < len(dt.columns) {
			targetColIdx = idx
		} else {
			// 嘗試作為欄位名稱
			colIdx, ok := colNameMap[target]
			if !ok {
				return fmt.Errorf("assignment target column '%s' does not exist", target)
			}
			targetColIdx = colIdx
		}
	}

	// Bind the node first
	boundNode, err := ccl.Bind(node, colNameMap)
	if err != nil {
		return err
	}
	if err := checkCCLColRange(boundNode, len(dt.columns)); err != nil {
		return err
	}

	// 預分配 row slice
	row := make([]any, len(dt.columns))

	ctx := &dataTableContext{
		row:        row,
		tableData:  tableData,
		rowNameMap: rowNameMap,
		colNameMap: colNameMap,
	}

	// 計算結果
	var results []any
	if ccl.IsRowDependent(ccl.GetExpressionNode(boundNode)) {
		results = make([]any, numRow)
		for i := range numRow {
			// 填充第 i 行的資料
			for j := range len(dt.columns) {
				if i < len(dt.columns[j].data) {
					row[j] = dt.columns[j].data[i]
				} else {
					row[j] = nil
				}
			}
			ctx.rowIndex = i

			// 評估表達式
			evalResult, err := ccl.EvaluateStatement(boundNode, ctx)
			if err != nil {
				// The statement's source text is filled in by ExecuteCCL,
				// which is the only place that knows which line is running.
				return &ccl.EvalError{Row: i, Err: err}
			}
			results[i] = evalResult.Value
		}
	} else {
		// 非行依賴表達式，只需計算一次
		// 使用第一行的資料作為上下文（雖然不應該依賴它）
		for j := range len(dt.columns) {
			if len(dt.columns[j].data) > 0 {
				row[j] = dt.columns[j].data[0]
			} else {
				row[j] = nil
			}
		}
		ctx.rowIndex = 0
		evalResult, err := ccl.EvaluateStatement(boundNode, ctx)
		if err != nil {
			return &ccl.EvalError{Row: -1, Err: err}
		}
		val := evalResult.Value

		// 檢查結果是否為 slice
		if rv := reflect.ValueOf(val); val != nil && rv.Kind() == reflect.Slice {
			// 如果是 slice，直接將其作為整欄資料
			results = make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				results[i] = rv.Index(i).Interface()
			}
		} else {
			// 如果是純量，則重複填充整欄
			results = make([]any, numRow)
			for i := range numRow {
				results[i] = val
			}
		}
	}

	// 更新目標列的資料
	dt.columns[targetColIdx].data = results

	return nil
}

// executeNewColumn executes a NEW column creation CCL statement
func executeNewColumn(dt *DataTable, node ccl.CCLNode, newColName string, numRow int, colNameMap map[string]int, tableData [][]any, rowNameMap *core.BiIndex) error {
	// Bind the node first
	boundNode, err := ccl.Bind(node, colNameMap)
	if err != nil {
		return err
	}
	if err := checkCCLColRange(boundNode, len(dt.columns)); err != nil {
		return err
	}

	// 預分配 row slice
	row := make([]any, len(dt.columns))

	ctx := &dataTableContext{
		row:        row,
		tableData:  tableData,
		rowNameMap: rowNameMap,
		colNameMap: colNameMap,
	}

	// 計算結果
	var results []any
	if ccl.IsRowDependent(ccl.GetExpressionNode(boundNode)) {
		results = make([]any, numRow)
		for i := range numRow {
			// 填充第 i 行的資料
			for j := range len(dt.columns) {
				if i < len(dt.columns[j].data) {
					row[j] = dt.columns[j].data[i]
				} else {
					row[j] = nil
				}
			}
			ctx.rowIndex = i

			// 評估表達式
			evalResult, err := ccl.EvaluateStatement(boundNode, ctx)
			if err != nil {
				// The statement's source text is filled in by ExecuteCCL,
				// which is the only place that knows which line is running.
				return &ccl.EvalError{Row: i, Err: err}
			}
			results[i] = evalResult.Value
		}
	} else {
		// 非行依賴表達式，只需計算一次
		for j := range len(dt.columns) {
			if len(dt.columns[j].data) > 0 {
				row[j] = dt.columns[j].data[0]
			} else {
				row[j] = nil
			}
		}
		ctx.rowIndex = 0
		evalResult, err := ccl.EvaluateStatement(boundNode, ctx)
		if err != nil {
			return &ccl.EvalError{Row: -1, Err: err}
		}
		val := evalResult.Value

		if rv := reflect.ValueOf(val); val != nil && rv.Kind() == reflect.Slice {
			// LogDebug("DataTable", "executeNewColumn", "Unwrapping slice of length %d matching numRow %d", rv.Len(), numRow)
			results = make([]any, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				results[i] = rv.Index(i).Interface()
			}
		} else {
			// LogDebug("DataTable", "executeNewColumn", "Treating value as scalar (val type: %T, numRow: %d)", val, numRow)
			results = make([]any, numRow)
			for i := range numRow {
				results[i] = val
			}
		}
	}

	// 創建新列並添加到 DataTable
	newCol := &DataList{
		data: results,
		name: newColName,
	}
	timestamp := time.Now().Unix()
	newCol.creationTimestamp = timestamp
	newCol.lastModifiedTimestamp.Store(timestamp)
	dt.AppendCols(newCol)

	return nil
}

// cclWorkingCopy returns a private copy of dt's columns and row names for
// ExecuteCCL to run a script against. Nothing else can reach it, so it needs no
// locking of its own.
func (dt *DataTable) cclWorkingCopy() *DataTable {
	work := NewDataTable()
	work.columns = make([]*DataList, len(dt.columns))
	for i, col := range dt.columns {
		work.columns[i] = &DataList{name: col.name, data: slices.Clone(col.data)}
	}
	if dt.rowNames != nil {
		work.rowNames = dt.rowNames.Clone()
	}
	return work
}

// commitCCLWorkingCopy writes a successful script's result back into dt. The
// existing columns keep their DataList objects and take the new data, so
// anything inside the library already holding one stays attached to the table;
// the columns the script created are appended after them.
func (dt *DataTable) commitCCLWorkingCopy(work *DataTable) {
	orig := len(dt.columns)
	for i := 0; i < orig; i++ {
		dt.columns[i].data = work.columns[i].data
	}
	for _, col := range work.columns[orig:] {
		dt.AppendCols(col)
	}
}
