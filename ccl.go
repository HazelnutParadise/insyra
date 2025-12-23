package insyra

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/Go-Utils/conv"
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

// compileCCLExpression compiles a CCL expression string into an AST for reuse.
// This avoids repeated tokenization and parsing for each row.
// Returns an error if assignment syntax (=) or NEW function is detected.
func compileCCLExpression(expression string) (ccl.CCLNode, error) {
	tokens, err := ccl.Tokenize(expression)
	if err != nil {
		return nil, err
	}

	// 檢查是否包含賦值語法或 NEW 函數（表達式模式不允許）
	if err := ccl.CheckExpressionMode(tokens); err != nil {
		return nil, err
	}

	return ccl.ParseExpression(tokens)
}

// compileMultilineCCL compiles multi-line CCL statements separated by ; or newline into ASTs.
// This supports assignment syntax (e.g., A=B+C) and NEW function.
func compileMultilineCCL(cclStatements string) ([]ccl.CCLNode, error) {
	// 按 ; 或換行分割語句
	lines := strings.FieldsFunc(cclStatements, func(r rune) bool {
		return r == ';' || r == '\n'
	})

	nodes := make([]ccl.CCLNode, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		tokens, err := ccl.Tokenize(line)
		if err != nil {
			return nil, err
		}
		node, err := ccl.ParseStatement(tokens)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// applyCCLOnDataTable evaluates the expression on each row of a DataTable.
// Optimized: compiles expression once and reuses AST for all rows.
func applyCCLOnDataTable(table *DataTable, expression string) ([]any, error) {
	var result []any
	var err error

	// 預先編譯表達式（只做一次 tokenize + parse）
	ast, err := compileCCLExpression(expression)
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

		// 準備 tableData 和 rowNameMap 以支援 . 運算符
		tableData := make([][]any, numCol)
		for j := range numCol {
			tableData[j] = table.columns[j].data
		}
		rowNameMap := table.rowNames

		// 預分配 row slice，避免每行都重新分配
		row := make([]any, numCol)

		if ccl.IsRowDependent(ccl.GetExpressionNode(ast)) {
			for i := range numRow {
				// 填充第 i 行的資料（重用 row slice）
				for j := range numCol {
					if i < len(table.columns[j].data) {
						row[j] = table.columns[j].data[i]
					} else {
						row[j] = nil
					}
				}
				// 直接使用預編譯的 AST
				val, err2 := ccl.Evaluate(ast, row, tableData, rowNameMap, i, colNameMap)
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
			val, err2 := ccl.Evaluate(ast, row, tableData, rowNameMap, 0, colNameMap)
			if err2 != nil {
				err = err2
				return
			}

			if rv := reflect.ValueOf(val); val != nil && rv.Kind() == reflect.Slice {
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
	ccl.RegisterFunction("IF", func(args ...any) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("IF requires 3 arguments")
		}

		// 嘗試將第一個參數轉換為布林值
		var cond bool
		switch v := args[0].(type) {
		case bool:
			cond = v
		case int, int32, int64:
			cond = conv.ParseInt(v) != 0
		case float32, float64:
			cond = conv.ParseF64(v) != 0
		case string:
			cond = v != "" && v != "0" && v != "false"
		case nil:
			cond = false
		default:
			return nil, fmt.Errorf("first argument to IF cannot be converted to boolean: %T", args[0])
		}

		if cond {
			return args[1], nil
		}
		return args[2], nil
	})
	ccl.RegisterFunction("AND", func(args ...any) (any, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("AND requires at least 2 arguments")
		}
		for _, arg := range args {
			if cond, ok := arg.(bool); !ok || !cond {
				return false, nil
			}
		}
		return true, nil
	})
	ccl.RegisterFunction("OR", func(args ...any) (any, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("OR requires at least 2 arguments")
		}
		for _, arg := range args {
			if cond, ok := arg.(bool); ok && cond {
				return true, nil
			}
		}
		return false, nil
	})
	ccl.RegisterFunction("CONCAT", func(args ...any) (any, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("CONCAT requires at least 2 arguments")
		}
		var result string
		for _, arg := range args {
			if str, ok := arg.(string); ok {
				result += str
			} else {
				argStr := conv.ToString(arg)
				result += argStr
			}
		}
		return result, nil
	})
	ccl.RegisterFunction("CASE", func(args ...any) (any, error) {
		if len(args) < 3 {
			return nil, fmt.Errorf("CASE requires at least 3 arguments")
		}

		// 參數數量必須是奇數 (條件1,結果1,條件2,結果2,...,預設值)
		if len(args)%2 != 1 {
			return nil, fmt.Errorf("CASE requires an odd number of arguments, with conditions and results in pairs, ending with a default value")
		}

		// 檢查條件和對應值
		for i := 0; i < len(args)-1; i += 2 {
			// 直接處理布林值
			if cond, ok := args[i].(bool); ok {
				if cond {
					return args[i+1], nil
				}
				continue // 跳過到下一個條件
			}

			// 安全地轉換其他類型為布林值
			var condition bool
			switch v := args[i].(type) {
			case int:
				condition = v != 0
			case int32:
				condition = v != 0
			case int64:
				condition = v != 0
			case float32:
				condition = v != 0
			case float64:
				condition = v != 0
			case string:
				// 簡化字串到布林值的轉換
				v = strings.TrimSpace(v)
				condition = v != "" && v != "0" && v != "false" && v != "False" && v != "FALSE"
			case nil:
				condition = false
			default:
				return nil, fmt.Errorf("condition at position %d of type %T cannot be evaluated as boolean", i, args[i])
			}

			if condition {
				return args[i+1], nil
			}
		}

		// 如果沒有條件符合，返回最後一個參數作為預設值
		return args[len(args)-1], nil
	})
	ccl.RegisterFunction("ISNA", func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("ISNA requires 1 argument")
		}
		val := args[0]
		switch v := val.(type) {
		case float64:
			return math.IsNaN(v), nil
		case float32:
			return math.IsNaN(float64(v)), nil
		case string:
			return v == "#N/A", nil
		}
		return false, nil
	})
	ccl.RegisterFunction("IFNA", func(args ...any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("IFNA requires 2 arguments")
		}
		val := args[0]
		isNA := false
		switch v := val.(type) {
		case float64:
			isNA = math.IsNaN(v)
		case float32:
			isNA = math.IsNaN(float64(v))
		case string:
			isNA = v == "#N/A"
		}

		if isNA {
			return args[1], nil
		}
		return val, nil
	})

	ccl.RegisterAggregateFunction("SUM", func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return 0.0, nil
		}
		var sum float64
		for _, col := range args {
			for _, val := range col {
				if f, ok := toFloat64(val); ok {
					sum += f
				}
			}
		}
		return sum, nil
	})
	ccl.RegisterAggregateFunction("AVG", func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return 0.0, nil
		}
		var sum float64
		var count int
		for _, col := range args {
			for _, val := range col {
				if f, ok := toFloat64(val); ok {
					sum += f
					count++
				}
			}
		}
		if count == 0 {
			return 0.0, nil
		}
		return sum / float64(count), nil
	})
	ccl.RegisterAggregateFunction("COUNT", func(args ...[]any) (any, error) {
		var count int
		for _, col := range args {
			count += len(col)
		}
		return float64(count), nil
	})
	ccl.RegisterAggregateFunction("MAX", func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return nil, nil
		}
		maxVal := -math.MaxFloat64
		found := false
		for _, col := range args {
			for _, val := range col {
				if f, ok := toFloat64(val); ok {
					if f > maxVal {
						maxVal = f
					}
					found = true
				}
			}
		}
		if !found {
			return nil, nil
		}
		return maxVal, nil
	})
	ccl.RegisterAggregateFunction("MIN", func(args ...[]any) (any, error) {
		if len(args) == 0 {
			return nil, nil
		}
		minVal := math.MaxFloat64
		found := false
		for _, col := range args {
			for _, val := range col {
				if f, ok := toFloat64(val); ok {
					if f < minVal {
						minVal = f
					}
					found = true
				}
			}
		}
		if !found {
			return nil, nil
		}
		return minVal, nil
	})
}

// toFloat64 is a helper to convert any to float64, copied from evaluator for use in functions
func toFloat64(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	case nil:
		return 0, true
	default:
		return 0, false
	}
}
