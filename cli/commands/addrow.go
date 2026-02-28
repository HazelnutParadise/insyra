package commands

import (
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "addrow",
		Usage:       "addrow <var> <values...>",
		Description: "Add one row to DataTable",
		Run:         runAddRowCommand,
	})
}

func runAddRowCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: addrow <var> <values...>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	values := make([]any, 0, len(args)-1)
	for _, raw := range args[1:] {
		values = append(values, parseLiteral(raw))
	}
	table.AppendRowsFromDataList(insyra.NewDataList(values...))
	_, _ = fmt.Fprintln(ctx.Output, "row added")
	return nil
}
