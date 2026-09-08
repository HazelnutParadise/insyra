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
		parsed, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return fmt.Errorf("clean: invalid standard deviation %q: it must be a number", args[2])
		}
		if parsed <= 0 {
			return fmt.Errorf("clean: standard deviation must be greater than 0, got %v", parsed)
		}
		stddev = parsed
	}
	if len(args) > 3 {
		return fmt.Errorf("clean: unexpected argument %q (usage: clean <var> nan|nil|strings|outliers [<stddev>])", args[3])
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
