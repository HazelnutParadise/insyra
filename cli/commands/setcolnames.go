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
	names := args[1:]
	if len(names) != table.NumCols() {
		return fmt.Errorf("setcolnames: got %d names for %d columns; pass exactly one name per column", len(names), table.NumCols())
	}
	table.SetColNames(names)
	if err := checkTableErr("setcolnames", table); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "column names updated")
	return nil
}
