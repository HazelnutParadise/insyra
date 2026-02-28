package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "row",
		Usage:       "row <var> <index|name> [as <var>]",
		Description: "Extract DataTable row as DataList",
		Run:         runRowCommand,
	})
}

func runRowCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: row <var> <index|name> [as <var>]")
	}
	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	selector := coreArgs[1]
	var dl any
	if index, convErr := strconv.Atoi(selector); convErr == nil {
		dl = table.GetRow(index)
	} else {
		dl = table.GetRowByName(selector)
	}
	if dl == nil {
		return fmt.Errorf("row not found: %s", selector)
	}
	ctx.Vars[alias] = dl
	_, _ = fmt.Fprintf(ctx.Output, "saved row to %s\n", alias)
	return nil
}
