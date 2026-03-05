package commands

import (
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "addcol",
		Usage:       "addcol <var> <values...>",
		Description: "Add one column to DataTable",
		Run:         runAddColCommand,
	})
}

func runAddColCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: addcol <var> <values...>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	values := make([]any, 0, len(args)-1)
	for _, raw := range args[1:] {
		values = append(values, parseLiteral(raw))
	}
	table.AppendCols(insyra.NewDataList(values...))
	_, _ = fmt.Fprintln(ctx.Output, "column added")
	return nil
}
