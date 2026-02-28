package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "rows",
		Usage:       "rows <var>",
		Description: "List DataTable row names",
		Run:         runRowsCommand,
	})
}

func runRowsCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: rows <var>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	for _, rowName := range table.RowNames() {
		_, _ = fmt.Fprintln(ctx.Output, rowName)
	}
	return nil
}
