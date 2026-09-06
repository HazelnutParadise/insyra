package commands

import (
	"fmt"
	"strconv"

	"github.com/HazelnutParadise/insyra"
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
	// Keep the concrete type: a nil *DataList stored in an `any` is not == nil,
	// and SaveState would later dereference it.
	var dl *insyra.DataList
	if index, convErr := strconv.Atoi(selector); convErr == nil {
		dl = table.GetRow(index)
	} else {
		dl = table.GetRowByName(selector)
	}
	if dl == nil {
		return fmt.Errorf("row: row not found: %s", selector)
	}
	ctx.Vars[alias] = dl
	_, _ = fmt.Fprintf(ctx.Output, "saved row to %s\n", alias)
	return nil
}
