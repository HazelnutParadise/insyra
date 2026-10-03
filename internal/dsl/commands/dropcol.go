package commands

import (
	"fmt"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "dropcol",
		Args:        OpenArgs(),
		Usage:       "dropcol <var> <col...>",
		Description: "Drop columns by name or index",
		Run:         runDropColCommand,
	})
}

func runDropColCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: dropcol <var> <col...>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	// Every token is resolved before anything is dropped, so a bad one leaves
	// the table untouched and the positions all refer to the same table.
	positions, err := resolveColumnTokens("dropcol", table, args[1:])
	if err != nil {
		return err
	}
	table.DropColsByNumber(positions...)
	if err := checkTableErr("dropcol", table); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "columns dropped")
	return nil
}
