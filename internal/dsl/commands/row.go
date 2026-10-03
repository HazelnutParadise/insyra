package commands

import (
	"fmt"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "row",
		Args:        MaxArgs(2).WithAlias(),
		Usage:       "row <var> <row> [as <var>]",
		Description: "Extract DataTable row as DataList",
		Run:         runRowCommand,
	})
}

func runRowCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: row <var> <row> [as <var>]")
	}
	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	pos, err := resolveRowToken("row", table, coreArgs[1])
	if err != nil {
		return err
	}
	// Keep the concrete type: a nil *DataList stored in an `any` is not == nil,
	// and SaveState would later dereference it.
	dl := table.GetRow(pos)
	if dl == nil {
		return fmt.Errorf("row: row not found: %s", coreArgs[1])
	}
	ctx.Vars[alias] = dl
	_, _ = fmt.Fprintf(ctx.Output, "saved row to %s\n", alias)
	return nil
}
