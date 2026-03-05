package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "col",
		Usage:       "col <var> <name|index> [as <var>]",
		Description: "Extract DataTable column as DataList",
		Run:         runColCommand,
	})
}

func runColCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: col <var> <name|index> [as <var>]")
	}
	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	selector := coreArgs[1]
	var dl any
	if index, convErr := strconv.Atoi(selector); convErr == nil {
		dl = table.GetColByNumber(index)
	} else {
		dl = table.GetColByName(selector)
	}
	if dl == nil {
		return fmt.Errorf("column not found: %s", selector)
	}
	ctx.Vars[alias] = dl
	_, _ = fmt.Fprintf(ctx.Output, "saved column to %s\n", alias)
	return nil
}
