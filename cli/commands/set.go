package commands

import (
	"fmt"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "set",
		Args:        MaxArgs(4),
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
	row, err := resolveRowToken("set", table, args[1])
	if err != nil {
		return err
	}
	col, err := resolveColumnToken("set", table, args[2])
	if err != nil {
		return err
	}
	value := parseLiteral(args[3])
	table.UpdateElement(row, col, value)
	// Read the write back, so a write the table refused cannot be reported as
	// done. (value comes from parseLiteral, so it is a comparable scalar.)
	if got := table.GetElementByNumberIndex(row, col); got != value {
		return fmt.Errorf("set: update did not take effect for row %s, col %s", args[1], args[2])
	}
	_, _ = fmt.Fprintln(ctx.Output, "updated")
	return nil
}
