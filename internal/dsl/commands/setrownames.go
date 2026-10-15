package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "setrownames",
		Args:        OpenArgs(),
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
	names := args[1:]
	if len(names) != table.NumRows() {
		return fmt.Errorf("setrownames: got %d names for %d rows; pass exactly one name per row", len(names), table.NumRows())
	}
	table.SetRowNames(names)
	if err := checkTableErr("setrownames", table); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "row names updated")
	return nil
}
