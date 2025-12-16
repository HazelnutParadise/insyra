package insyra

import (
	"fmt"
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

// evaluateFormula evaluates a formula string on a single row of data.
// colNameMap is optional - pass nil if not using ['colName'] syntax.
func evaluateCCLFormula(row []any, formula string, colNameMap map[string]int) (any, error) {
	tokens, err := ccl.Tokenize(formula)
	if err != nil {
		return nil, err
	}
	ast, err := ccl.Parse(tokens)
	if err != nil {
		return nil, err
	}
	return ccl.Evaluate(ast, row, colNameMap)
}

// applyCCLOnDataTable evaluates the formula on each row of a DataTable.
func applyCCLOnDataTable(table *DataTable, formula string) ([]any, error) {
	var result []any
	var err error
	table.AtomicDo(func(table *DataTable) {
		numRow, numCol := table.getMaxColLength(), len(table.columns)
		result = make([]any, numRow)

		// 建立欄位名稱到索引的映射（支援 ['colName'] 語法）
		colNameMap := make(map[string]int)
		for j := range numCol {
			if table.columns[j].name != "" {
				colNameMap[table.columns[j].name] = j
			}
		}

		for i := range numRow {
			// 建構第 i 行的資料
			row := make([]any, numCol)
			for j := range numCol {
				if i < len(table.columns[j].data) {
					row[j] = table.columns[j].data[i]
				} else {
					row[j] = nil
				}
			}
			val, err2 := evaluateCCLFormula(row, formula, colNameMap)
			if err2 != nil {
				err = err2
				return
			}
			result[i] = val
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
}
