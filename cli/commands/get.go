package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "get",
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
	row, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid row index: %s", args[1])
	}
	col := args[2]
	var value any
	if colNum, convErr := strconv.Atoi(col); convErr == nil {
		value = table.GetElementByNumberIndex(row, colNum)
	} else {
		value = table.GetElement(row, col)
	}
	_, _ = fmt.Fprintf(ctx.Output, "%v\n", value)
	return nil
}
