package insyra

import (
	"fmt"
)

// EvaluateFormula evaluates a formula string on a single row of data.
func EvaluateCCLFormula(row []any, formula string) (any, error) {
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

// ApplyCCLOnDataTable evaluates the formula on each row of a DataTable.
func ApplyCCLOnDataTable(table *DataTable, formula string) ([]any, error) {
	numRow, numCol := table.Size()
	result := make([]any, numRow)
	for i := 0; i < numRow; i++ {
		// 建構第 i 行的資料
		row := make([]any, numCol)
		for j := 0; j < numCol; j++ {
			if i < len(table.columns[j].data) {
				row[j] = table.columns[j].data[i]
			} else {
				row[j] = nil
			}
		}
		val, err := EvaluateCCLFormula(row, formula)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}

// InitCCLFunctions registers default functions for use with CCL.
func InitCCLFunctions() {
	RegisterFunction("IF", func(args ...any) (any, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("IF requires 3 arguments")
		}
		cond, ok := args[0].(bool)
		if !ok {
			return nil, fmt.Errorf("first argument to IF must be boolean")
		}
		if cond {
			return args[1], nil
		}
		return args[2], nil
	})
}
