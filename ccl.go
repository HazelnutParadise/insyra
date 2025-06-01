package insyra

import (
	"fmt"

	"github.com/HazelnutParadise/Go-Utils/conv"
)

// evaluateFormula evaluates a formula string on a single row of data.
func evaluateCCLFormula(row []any, formula string) (any, error) {
	tokens, err := tokenize(formula)
	if err != nil {
		return nil, err
	}
	ast, err := parse(tokens)
	if err != nil {
		return nil, err
	}
	return evaluate(ast, row)
}

// applyCCLOnDataTable evaluates the formula on each row of a DataTable.
func applyCCLOnDataTable(table *DataTable, formula string) ([]any, error) {
	table.mu.Lock()
	defer table.mu.Unlock()
	numRow, numCol := table.getMaxColLength(), len(table.columns)
	result := make([]any, numRow)
	for i := 0; i < numRow; i++ {
		// 建構第 i 行的資料
		row := make([]any, numCol)
		for j := range numCol {
			if i < len(table.columns[j].data) {
				row[j] = table.columns[j].data[i]
			} else {
				row[j] = nil
			}
		}
		val, err := evaluateCCLFormula(row, formula)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}

// InitCCLFunctions registers default functions for use with CCL.
func initCCLFunctions() {
	cclRegisterFunction("IF", func(args ...any) (any, error) {
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
	cclRegisterFunction("AND", func(args ...any) (any, error) {
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
	cclRegisterFunction("OR", func(args ...any) (any, error) {
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
	cclRegisterFunction("CONCAT", func(args ...any) (any, error) {
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
}
