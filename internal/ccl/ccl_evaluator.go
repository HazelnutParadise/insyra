package ccl

import (
	"fmt"
	"math"
	"strconv"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// 新增一個全域變數來追蹤遞迴深度
var evalDepth int = 0
var maxEvalDepth int = 100 // 設置合理的最大遞迴深度

// ResetEvalDepth 重置全域評估深度變數
func ResetEvalDepth() {
	evalDepth = 0
}

func Evaluate(n cclNode, row []any) (any, error) {
	// 檢查遞迴深度並添加除錯日誌
	evalDepth++

	// 每10層輸出一次當前深度
	if evalDepth%10 == 0 {
		fmt.Printf("CCL evaluation depth: %d\n", evalDepth)
	}

	if evalDepth > maxEvalDepth {
		evalDepth = 0
		return nil, fmt.Errorf("evaluate: maximum recursion depth exceeded (%d), possibly infinite recursion", maxEvalDepth)
	}

	// 使用 defer 確保退出前減少深度計數
	defer func() {
		evalDepth--
	}()

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
	case *funcCallNode:
		args := []any{}
		for _, arg := range t.args {
			val, err := Evaluate(arg, row)
			if err != nil {
				return nil, err
			}
			args = append(args, val)
		}
		return callFunction(t.name, args)
	case *cclBinaryOpNode:
		left, err := Evaluate(t.left, row)
		if err != nil {
			return nil, err
		}
		right, err := Evaluate(t.right, row)
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
			val, err := Evaluate(valNode, row)
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

	// 對於布林運算，特別處理
	if op == "==" || op == "!=" {
		// 如果是布林值，直接比較
		lb, lokBool := left.(bool)
		rb, rokBool := right.(bool)
		if lokBool && rokBool {
			if op == "==" {
				return lb == rb, nil
			} else {
				return lb != rb, nil
			}
		}

		// 如果是字串比較
		ls, lokStr := left.(string)
		rs, rokStr := right.(string)
		if lokStr && rokStr {
			if op == "==" {
				return ls == rs, nil
			} else {
				return ls != rs, nil
			}
		}

		// 不同數據類型間的比較，直接返回false或true
		if op == "==" {
			return false, nil
		} else { // op == "!="
			return true, nil
		}
	}

	// 其他情況處理數值比較
	lf, lok := toFloat64(left)
	rf, rok := toFloat64(right)
	if !lok || !rok {
		// 不同數據類型間的大小比較，直接返回false
		if op == ">" || op == "<" || op == ">=" || op == "<=" {
			return false, nil
		}
		return nil, fmt.Errorf("invalid operands for %s: %v, %v", op, left, right)
	}

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
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	case nil:
		return 0.0, true
	default:
		return 0, false
	}
}
