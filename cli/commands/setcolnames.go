package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "setcolnames",
		Usage:       "setcolnames <var> <names...>",
		Description: "Set DataTable column names",
		Run:         runSetColNamesCommand,
	})
}

func runSetColNamesCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: setcolnames <var> <names...>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	table.SetColNames(args[1:])
	_, _ = fmt.Fprintln(ctx.Output, "column names updated")
	return nil
}
