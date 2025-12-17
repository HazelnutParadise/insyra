package insyra

import (
	"fmt"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/internal/ccl"
	"github.com/HazelnutParadise/insyra/internal/utils"
)

func (dt *DataTable) AddColUsingCCL(newColName, cclFormula string) *DataTable {
	// 添加 recover 以防止程序崩潰
	defer func() {
		if r := recover(); r != nil {
			LogWarning("DataTable", "AddColUsingCCL", "Panic recovered: %v", r)
		}
	}()

	// 重設遞歸深度和調用深度計數器
	resetCCLEvalDepth()
	resetCCLFuncCallDepth()

	resultDtChan := make(chan *DataTable, 1)

	dt.AtomicDo(func(dt *DataTable) {
		// 優先記錄表達式開始評估的時間
		startTime := time.Now()
		LogDebug("DataTable", "AddColUsingCCL", "Starting CCL evaluation for %s: %s", newColName, cclFormula)

		result, err := applyCCLOnDataTable(dt, cclFormula)
		if err != nil {
			elapsed := time.Since(startTime)
			LogWarning("DataTable", "AddColUsingCCL", "Failed to apply CCL on DataTable after %v: %v", elapsed, err)
		} else {
			elapsed := time.Since(startTime)
			LogDebug("DataTable", "AddColUsingCCL", "CCL evaluation completed in %v", elapsed)
			dt.AppendCols(NewDataList(result...).SetName(newColName))
		}
		resultDtChan <- dt
	})
	return <-resultDtChan
}

// ExecuteCCL executes multi-line CCL statements on the DataTable.
// It supports assignment syntax (e.g., A=B+C) and NEW('colName', expr) for creating new columns.
// Multiple statements can be separated by ; or newline.
// Assignment operations modify existing columns; if the target column doesn't exist, an error is returned.
// Returns the modified DataTable.
func (dt *DataTable) ExecuteCCL(cclStatements string) *DataTable {
	// 添加 recover 以防止程序崩潰
	defer func() {
		if r := recover(); r != nil {
			LogWarning("DataTable", "ExecuteCCL", "Panic recovered: %v", r)
		}
	}()

	// 重設遞歸深度和調用深度計數器
	resetCCLEvalDepth()
	resetCCLFuncCallDepth()

	resultDtChan := make(chan *DataTable, 1)

	dt.AtomicDo(func(dt *DataTable) {
		startTime := time.Now()
		LogDebug("DataTable", "ExecuteCCL", "Starting CCL execution: %s", cclStatements)

		// 編譯多行 CCL 語句
		nodes, err := compileMultilineCCL(cclStatements)
		if err != nil {
			LogWarning("DataTable", "ExecuteCCL", "Failed to parse CCL statements: %v", err)
			resultDtChan <- dt
			return
		}

		numRow, numCol := dt.getMaxColLength(), len(dt.columns)

		// 建立欄位名稱到索引的映射
		colNameMap := make(map[string]int, numCol)
		for j := range numCol {
			if dt.columns[j].name != "" {
				colNameMap[dt.columns[j].name] = j
			}
		}

		// 執行每個 CCL 語句
		for _, node := range nodes {
			if err := executeCCLNode(dt, node, numRow, colNameMap); err != nil {
				LogWarning("DataTable", "ExecuteCCL", "Failed to execute CCL statement: %v", err)
				resultDtChan <- dt
				return
			}
			// 更新 numCol 和 colNameMap（如果添加了新列）
			numCol = len(dt.columns)
			for j := range numCol {
				if dt.columns[j].name != "" {
					colNameMap[dt.columns[j].name] = j
				}
			}
		}

		elapsed := time.Since(startTime)
		LogDebug("DataTable", "ExecuteCCL", "CCL execution completed in %v", elapsed)
		resultDtChan <- dt
	})
	return <-resultDtChan
}

// executeCCLNode executes a single CCL node on the DataTable
func executeCCLNode(dt *DataTable, node ccl.CCLNode, numRow int, colNameMap map[string]int) error {
	// 檢查是否為賦值語句
	if ccl.IsAssignmentNode(node) {
		target, _ := ccl.GetAssignmentTarget(node)
		return executeAssignment(dt, node, target, numRow, colNameMap)
	}

	// 檢查是否為 NEW 函數
	if ccl.IsNewColNode(node) {
		newColName, _, _ := ccl.GetNewColInfo(node)
		return executeNewColumn(dt, node, newColName, numRow, colNameMap)
	}

	// 普通表達式，不做任何操作（只有在 slice 模式下才有意義）
	return nil
}

// executeAssignment executes an assignment CCL statement
func executeAssignment(dt *DataTable, node ccl.CCLNode, target string, numRow int, colNameMap map[string]int) error {
	// 確定目標列索引
	targetColIdx := -1

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
		idx := utils.ParseColIndex(target)
		if idx >= 0 && idx < len(dt.columns) {
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

	// 預分配 row slice
	row := make([]any, len(dt.columns))

	// 計算每一行的結果
	results := make([]any, numRow)
	for i := range numRow {
		// 填充第 i 行的資料
		for j := range len(dt.columns) {
			if i < len(dt.columns[j].data) {
				row[j] = dt.columns[j].data[i]
			} else {
				row[j] = nil
			}
		}

		// 評估表達式
		evalResult, err := ccl.EvaluateStatement(node, row, colNameMap)
		if err != nil {
			return err
		}
		results[i] = evalResult.Value
	}

	// 更新目標列的資料
	dt.columns[targetColIdx].data = results

	return nil
}

// executeNewColumn executes a NEW column creation CCL statement
func executeNewColumn(dt *DataTable, node ccl.CCLNode, newColName string, numRow int, colNameMap map[string]int) error {
	// 預分配 row slice
	row := make([]any, len(dt.columns))

	// 計算每一行的結果
	results := make([]any, numRow)
	for i := range numRow {
		// 填充第 i 行的資料
		for j := range len(dt.columns) {
			if i < len(dt.columns[j].data) {
				row[j] = dt.columns[j].data[i]
			} else {
				row[j] = nil
			}
		}

		// 評估表達式
		evalResult, err := ccl.EvaluateStatement(node, row, colNameMap)
		if err != nil {
			return err
		}
		results[i] = evalResult.Value
	}

	// 創建新列並添加到 DataTable
	newCol := NewDataList(results...).SetName(newColName)
	dt.AppendCols(newCol)

	return nil
}
