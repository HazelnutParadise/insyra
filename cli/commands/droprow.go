package commands

import (
	"fmt"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "droprow",
		Args:        OpenArgs(),
		Usage:       "droprow <var> <row...>",
		Description: "Drop rows by index or name",
		Run:         runDropRowCommand,
	})
}

func runDropRowCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: droprow <var> <row...>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	// Every token is resolved before anything is dropped, so a bad one leaves
	// the table untouched and the positions all refer to the same table.
	positions := make([]int, 0, len(args)-1)
	for _, token := range args[1:] {
		pos, err := resolveRowToken("droprow", table, token)
		if err != nil {
			return err
		}
		positions = append(positions, pos)
	}
	table.DropRowsByIndex(positions...)
	if err := checkTableErr("droprow", table); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "rows dropped")
	return nil
}
