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
	if _, convErr := strconv.Atoi(col); convErr == nil {
		table.UpdateElement(row, col, value)
	} else {
		table.UpdateElement(row, col, value)
	}
	_, _ = fmt.Fprintln(ctx.Output, "updated")
	return nil
}
