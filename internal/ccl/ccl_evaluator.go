package ccl

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// 新增一個全域變數來追蹤遞迴深度
var evalDepth int = 0
var maxEvalDepth int = 100 // 設置合理的最大遞迴深度

// ResetEvalDepth 重置全域評估深度變數
func ResetEvalDepth() {
	evalDepth = 0
}

// Evaluate evaluates a CCL node with the given row data.
// colNameMap is optional - pass nil if not using ['colName'] syntax.
// colNameMap maps column names to their indices (0-based).
func Evaluate(n cclNode, row []any, colNameMap ...map[string]int) (any, error) {
	var nameMap map[string]int
	if len(colNameMap) > 0 && colNameMap[0] != nil {
		nameMap = colNameMap[0]
	}
	return evaluateWithContext(n, row, nameMap)
}

func evaluateWithContext(n cclNode, row []any, colNameMap map[string]int) (any, error) {
	// 檢查遞迴深度（移除 debug 輸出以提升效能）
	evalDepth++
	if evalDepth > maxEvalDepth {
		evalDepth = 0
		return nil, fmt.Errorf("evaluate: maximum recursion depth exceeded (%d), possibly infinite recursion", maxEvalDepth)
	}
	defer func() { evalDepth-- }()

	switch t := n.(type) {
	case *cclNumberNode:
		return t.value, nil
	case *cclStringNode:
		return t.value, nil
	case *cclBooleanNode:
		return t.value, nil
	case *cclIdentifierNode:
		idx := utils.ParseColIndex(t.name)
		if idx >= len(row) {
			return nil, fmt.Errorf("column %s out of range", t.name)
		}
		return row[idx], nil
	case *cclColIndexNode:
		// [A] 形式的欄位索引引用
		idx := utils.ParseColIndex(t.index)
		if idx >= len(row) {
			return nil, fmt.Errorf("column [%s] out of range", t.index)
		}
		return row[idx], nil
	case *cclColNameNode:
		// ['colName'] 形式的欄位名稱引用
		if colNameMap == nil {
			return nil, fmt.Errorf("column name reference ['%s'] used but no column name mapping provided", t.name)
		}
		idx, ok := colNameMap[t.name]
		if !ok {
			return nil, fmt.Errorf("column name '%s' not found", t.name)
		}
		if idx >= len(row) {
			return nil, fmt.Errorf("column ['%s'] (index %d) out of range", t.name, idx)
		}
		return row[idx], nil
	case *funcCallNode:
		args := []any{}
		for _, arg := range t.args {
			val, err := evaluateWithContext(arg, row, colNameMap)
			if err != nil {
				return nil, err
			}
			args = append(args, val)
		}
		return callFunction(t.name, args)
	case *cclBinaryOpNode:
		left, err := evaluateWithContext(t.left, row, colNameMap)
		if err != nil {
			return nil, err
		}
		right, err := evaluateWithContext(t.right, row, colNameMap)
		if err != nil {
			return nil, err
		}
		return applyOperator(t.op, left, right)
	case *cclChainedComparisonNode:
		// 處理連續比較運算
		if len(t.values) != len(t.ops)+1 {
			return nil, fmt.Errorf("invalid chained comparison: number of values (%d) should be one more than number of operators (%d)", len(t.values), len(t.ops))
		}

		// 評估所有值
		values := make([]any, len(t.values))
		for i, valNode := range t.values {
			val, err := evaluateWithContext(valNode, row, colNameMap)
			if err != nil {
				return nil, err
			}
			values[i] = val
		}

		// 逐對比較所有值
		for i := 0; i < len(t.ops); i++ {
			result, err := applyOperator(t.ops[i], values[i], values[i+1])
			if err != nil {
				return nil, err
			}

			// 如果任何一個比較結果為假，整個連續比較的結果就為假
			boolResult, ok := result.(bool)
			if !ok {
				return nil, fmt.Errorf("comparison did not result in a boolean: %v", result)
			}
			if !boolResult {
				return false, nil
			}
		}

		// 所有比較都為真，整個結果為真
		return true, nil
	}

	return nil, fmt.Errorf("invalid node")
}

func applyOperator(op string, left, right any) (any, error) {
	// 特殊情況處理：其中一方為nil
	if left == nil || right == nil {
		switch op {
		case "==":
			// 只有當兩者都是nil時才相等
			return left == nil && right == nil, nil
		case "!=":
			// 只要有一方不是nil就不相等
			return left != nil || right != nil, nil
		case ">", "<", ">=", "<=":
			// nil與任何值進行大小比較都返回false
			return false, nil
		case "+", "-", "*", "/": // Added "^" to this line in the original thought process, but it's handled by toFloat64
			// 將 nil 視為 0 進行算術運算
			lf, lok := toFloat64(left)
			rf, rok := toFloat64(right)
			if !lok || !rok { // Should not happen if toFloat64 handles nil
				return nil, fmt.Errorf("internal error: failed to convert nil to float64 for operator %s", op)
			}
			switch op {
			case "+":
				return lf + rf, nil
			case "-":
				return lf - rf, nil
			case "*":
				return lf * rf, nil
			case "/":
				if rf == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				return lf / rf, nil
			}
		}
	}

	// 對於比較運算，嘗試將值轉換為數字進行比較
	lf, lok := toFloat64(left)
	rf, rok := toFloat64(right)
	if lok && rok {
		// 兩者都能轉換為數字，使用數字比較
		switch op {
		case "+":
			return lf + rf, nil
		case "-":
			return lf - rf, nil
		case "*":
			return lf * rf, nil
		case "/":
			return lf / rf, nil
		case "^":
			return math.Pow(lf, rf), nil
		case ">":
			return lf > rf, nil
		case "<":
			return lf < rf, nil
		case ">=":
			return lf >= rf, nil
		case "<=":
			return lf <= rf, nil
		case "==":
			return lf == rf, nil
		case "!=":
			return lf != rf, nil
		default:
			return nil, fmt.Errorf("unsupported operator: %s", op)
		}
	}

	// 如果不能都轉換為數字，對於==和!=，檢查是否都是字串
	if op == "==" || op == "!=" {
		ls, lokStr := left.(string)
		rs, rokStr := right.(string)
		if lokStr && rokStr {
			if op == "==" {
				return ls == rs, nil
			} else {
				return ls != rs, nil
			}
		}

		// 如果是布林值比較
		lb, lokBool := left.(bool)
		rb, rokBool := right.(bool)
		if lokBool && rokBool {
			if op == "==" {
				return lb == rb, nil
			} else {
				return lb != rb, nil
			}
		}

		// 不同數據類型間的比較，直接返回false或true
		if op == "==" {
			return false, nil
		} else { // op == "!="
			return true, nil
		}
	}

	// 對於大小比較，如果不能轉換為數字，返回false
	if op == ">" || op == "<" || op == ">=" || op == "<=" {
		return false, nil
	}

	return nil, fmt.Errorf("invalid operands for %s: %v, %v", op, left, right)
}

func toFloat64(val any) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case bool:
		if v {
			return 1.0, true
		}
		return 0.0, true
	case string:
		trimmed := strings.TrimSpace(v)
		f, err := strconv.ParseFloat(trimmed, 64)
		return f, err == nil
	case nil:
		return 0.0, true
	default:
		return 0, false
	}
}
