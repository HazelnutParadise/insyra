package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "setrownames",
		Usage:       "setrownames <var> <names...>",
		Description: "Set DataTable row names",
		Run:         runSetRowNamesCommand,
	})
}

func runSetRowNamesCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: setrownames <var> <names...>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	table.SetRowNames(args[1:])
	_, _ = fmt.Fprintln(ctx.Output, "row names updated")
	return nil
}
