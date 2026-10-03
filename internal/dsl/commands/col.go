package commands

import (
	"fmt"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "col",
		Args:        MaxArgs(2).WithAlias(),
		Usage:       "col <var> <col> [as <var>]",
		Description: "Extract DataTable column as DataList",
		Run:         runColCommand,
	})
}

func runColCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: col <var> <col> [as <var>]")
	}
	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	pos, err := resolveColumnToken("col", table, coreArgs[1])
	if err != nil {
		return err
	}
	// Keep the concrete type: a nil *DataList stored in an `any` is not == nil,
	// and SaveState would later dereference it.
	dl := table.GetColByNumber(pos)
	if dl == nil {
		return fmt.Errorf("col: column not found: %s", coreArgs[1])
	}
	ctx.Vars[alias] = dl
	_, _ = fmt.Fprintf(ctx.Output, "saved column to %s\n", alias)
	return nil
}
