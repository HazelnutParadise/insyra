package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "set",
		Usage:       "set <var> <row> <col> <value>",
		Description: "Set single element in DataTable",
		Run:         runSetCommand,
	})
}

func runSetCommand(ctx *ExecContext, args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("usage: set <var> <row> <col> <value>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	row, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid row index: %s", args[1])
	}
	col := args[2]
	value := parseLiteral(args[3])
	table.UpdateElement(row, col, value)
	// Verify the write took effect; UpdateElement is a silent no-op when the
	// row/column does not resolve, so read back and surface a real error instead
	// of always reporting success. (value comes from parseLiteral, so it is a
	// comparable scalar.)
	if got := table.GetElement(row, col); got != value {
		return fmt.Errorf("set: update did not take effect for row %d, col %q (does it exist?)", row, col)
	}
	_, _ = fmt.Fprintln(ctx.Output, "updated")
	return nil
}
