package commands

import (
	"fmt"
	"strconv"
	"strings"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "filter",
		Usage:       "filter <var> <expr> [as <var>]",
		Description: "Filter DataTable by expression: <col> <op> <value>",
		Run:         runFilterCommand,
	})
}

func runFilterCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: filter <var> <expr> [as <var>]")
	}
	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}

	expr := strings.TrimSpace(strings.Join(coreArgs[1:], " "))
	col, op, rawValue, err := parseFilterExpression(expr)
	if err != nil {
		return err
	}
	target := parseLiteral(rawValue)

	clone := table.Clone()
	dropRows := []int{}
	for row := 0; row < clone.NumRows(); row++ {
		value := clone.GetElement(row, col)
		if !compareValue(value, op, target) {
			dropRows = append(dropRows, row)
		}
	}
	if len(dropRows) > 0 {
		clone.DropRowsByIndex(dropRows...)
	}
	ctx.Vars[alias] = clone
	_, _ = fmt.Fprintf(ctx.Output, "filtered rows: %d -> %d (%s)\n", table.NumRows(), clone.NumRows(), alias)
	return nil
}

func parseFilterExpression(expr string) (string, string, string, error) {
	ops := []string{"<=", ">=", "==", "!=", "<", ">"}
	for _, op := range ops {
		idx := strings.Index(expr, op)
		if idx <= 0 {
			continue
		}
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+len(op):])
		if left == "" || right == "" {
			return "", "", "", fmt.Errorf("invalid filter expression: %s", expr)
		}
		return left, op, right, nil
	}
	return "", "", "", fmt.Errorf("unsupported filter expression: %s", expr)
}

func compareValue(value any, op string, target any) bool {
	vf, vOK := toFloat(value)
	tf, tOK := toFloat(target)
	if vOK && tOK {
		switch op {
		case "<":
			return vf < tf
		case "<=":
			return vf <= tf
		case ">":
			return vf > tf
		case ">=":
			return vf >= tf
		case "==":
			return vf == tf
		case "!=":
			return vf != tf
		}
	}
	vs := fmt.Sprintf("%v", value)
	ts := fmt.Sprintf("%v", target)
	switch op {
	case "==":
		return vs == ts
	case "!=":
		return vs != ts
	case "<":
		return vs < ts
	case "<=":
		return vs <= ts
	case ">":
		return vs > ts
	case ">=":
		return vs >= ts
	default:
		return false
	}
}

func toFloat(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case string:
		number, err := strconv.ParseFloat(typed, 64)
		if err == nil {
			return number, true
		}
	}
	return 0, false
}
