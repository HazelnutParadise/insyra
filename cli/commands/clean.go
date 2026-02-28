package commands

import (
	"fmt"
	"strconv"
	"strings"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "clean",
		Usage:       "clean <var> nan|nil|strings|outliers [<stddev>]",
		Description: "Clean values from DataTable/DataList",
		Run:         runCleanCommand,
	})
}

func runCleanCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: clean <var> nan|nil|strings|outliers [<stddev>]")
	}
	name := args[0]
	mode := strings.ToLower(args[1])
	stddev := 2.0
	if len(args) >= 3 {
		if parsed, err := strconv.ParseFloat(args[2], 64); err == nil {
			stddev = parsed
		}
	}

	if table, err := getDataTableVar(ctx, name); err == nil {
		switch mode {
		case "nan":
			table.DropRowsContainNaN()
		case "nil":
			table.DropRowsContainNil()
		case "strings":
			table.DropRowsContainString()
		default:
			return fmt.Errorf("unsupported clean mode for datatable: %s", mode)
		}
		_, _ = fmt.Fprintln(ctx.Output, "cleaned")
		return nil
	}

	if list, err := getDataListVar(ctx, name); err == nil {
		switch mode {
		case "nan":
			list.ClearNaNs()
		case "nil":
			list.ClearNils()
		case "strings":
			list.ClearStrings()
		case "outliers":
			list.ClearOutliers(stddev)
		default:
			return fmt.Errorf("unsupported clean mode for datalist: %s", mode)
		}
		_, _ = fmt.Fprintln(ctx.Output, "cleaned")
		return nil
	}

	return fmt.Errorf("variable not found: %s", name)
}
