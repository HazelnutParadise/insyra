package commands

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "filter",
		Usage:       "filter <var> <expr> [as <var>]",
		Description: "Filter DataTable by CCL expression",
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
	clone := table.Clone()
	tempColName := fmt.Sprintf("__filter_%d", time.Now().UnixNano())
	clone.AddColUsingCCL(tempColName, expr)
	condCol := clone.GetColByName(tempColName)
	if condCol == nil {
		return fmt.Errorf("failed to evaluate CCL expression: %s", expr)
	}

	dropRows := []int{}
	values := condCol.Data()
	for row := 0; row < len(values); row++ {
		if !toBool(values[row]) {
			dropRows = append(dropRows, row)
		}
	}
	if len(dropRows) > 0 {
		clone.DropRowsByIndex(dropRows...)
	}
	clone.DropColsByName(tempColName)
	ctx.Vars[alias] = clone
	_, _ = fmt.Fprintf(ctx.Output, "filtered rows: %d -> %d (%s)\n", table.NumRows(), clone.NumRows(), alias)
	return nil
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

func toBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case nil:
		return false
	case string:
		lower := strings.ToLower(strings.TrimSpace(typed))
		switch lower {
		case "true", "t", "yes", "y", "1":
			return true
		case "false", "f", "no", "n", "0", "", "nil", "null":
			return false
		}
		if parsed, ok := toFloat(typed); ok {
			return parsed != 0
		}
		return true
	default:
		if parsed, ok := toFloat(typed); ok {
			return parsed != 0
		}
	}
	return false
}
