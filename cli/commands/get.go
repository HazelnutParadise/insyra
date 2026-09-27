package commands

import (
	"fmt"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "get",
		Args:        MaxArgs(3),
		Usage:       "get <var> <row> <col>",
		Description: "Get single element from DataTable",
		Run:         runGetCommand,
	})
}

func runGetCommand(ctx *ExecContext, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: get <var> <row> <col>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	row, err := resolveRowToken("get", table, args[1])
	if err != nil {
		return err
	}
	col, err := resolveColumnToken("get", table, args[2])
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", table.GetElementByNumberIndex(row, col))
	return nil
}
