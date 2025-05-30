package insyra

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func evaluate(n cclNode, row []any) (any, error) {
	switch t := n.(type) {
	case *cclNumberNode:
		return t.value, nil
	case *cclStringNode:
		return t.value, nil
	case *cclIdentifierNode:
		idx := colNameToIndex(t.name)
		if idx >= len(row) {
			return nil, fmt.Errorf("column %s out of range", t.name)
		}
		return row[idx], nil
	case *funcCallNode:
		args := []any{}
		for _, arg := range t.args {
			val, _ := evaluate(arg, row)
			args = append(args, val)
		}
		return callFunction(t.name, args)
	case *cclBinaryOpNode:
		left, err := evaluate(t.left, row)
		if err != nil {
			return nil, err
		}
		right, err := evaluate(t.right, row)
		if err != nil {
			return nil, err
		}
		return applyOperator(t.op, left, right)
	}

	return nil, fmt.Errorf("invalid node")
}

func applyOperator(op string, left, right any) (any, error) {
	lf, lok := toFloat64(left)
	rf, rok := toFloat64(right)
	if !lok || !rok {
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
	case string:
		f, err := strconv.ParseFloat(v, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func colNameToIndex(name string) int {
	name = strings.ToUpper(name)
	sum := 0
	for i := range len(name) {
		sum = sum*26 + int(name[i]-'A'+1)
	}
	return sum - 1
}
