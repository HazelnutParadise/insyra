package commands

import (
	"fmt"
	"strings"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "replace",
		Usage:       "replace <var> <old|nan|nil> <new>",
		Description: "Replace values in DataTable/DataList",
		Run:         runReplaceCommand,
	})
}

func runReplaceCommand(ctx *ExecContext, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: replace <var> <old|nan|nil> <new>")
	}
	name := args[0]
	oldToken := strings.ToLower(args[1])
	newValue := parseLiteral(args[2])

	if table, err := getDataTableVar(ctx, name); err == nil {
		switch oldToken {
		case "nan":
			table.ReplaceNaNsWith(newValue)
		case "nil":
			table.ReplaceNilsWith(newValue)
		default:
			table.Replace(parseLiteral(args[1]), newValue)
		}
		_, _ = fmt.Fprintln(ctx.Output, "replaced")
		return nil
	}
	if list, err := getDataListVar(ctx, name); err == nil {
		switch oldToken {
		case "nan":
			list.ReplaceNaNsWith(newValue)
		case "nil":
			list.ReplaceNilsWith(newValue)
		default:
			list.ReplaceAll(parseLiteral(args[1]), newValue)
		}
		_, _ = fmt.Fprintln(ctx.Output, "replaced")
		return nil
	}
	return fmt.Errorf("variable not found: %s", name)
}
