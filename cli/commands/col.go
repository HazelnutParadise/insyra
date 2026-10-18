package commands

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "col",
		Usage:       "col <var> <name|number|letter> [as <var>]",
		Description: "Extract DataTable column as DataList. A selector that is a whole number is a 0-based position; otherwise an exact column name wins, then an Excel-style letter (A, B, ..., AA)",
		Examples: []string{
			"insyra col t price as x",
			"insyra col t 0 as x",
			"insyra col t A as x",
		},
		Run: runColCommand,
	})
}

func runColCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: col <var> <name|number|letter> [as <var>]")
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
		dl = table.GetColByNumber(index)
	} else if slices.Contains(table.ColNames(), selector) {
		dl = table.GetColByName(selector)
	} else if index, ok := insyra.ParseColIndex(selector); ok {
		// Only a letter that lands inside the table is a column; one past the
		// end falls through to "not found" instead of a library warning.
		if _, cols := table.Size(); index < cols {
			dl = table.GetColByNumber(index)
		}
	}
	if dl == nil {
		return fmt.Errorf("col: column not found: %s", selector)
	}
	ctx.Vars[alias] = dl
	_, _ = fmt.Fprintf(ctx.Output, "saved column to %s\n", alias)
	return nil
}
