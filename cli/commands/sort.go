package commands

import (
	"fmt"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "sort",
		Usage:       "sort <var> <col> [asc|desc]",
		Description: "Sort DataTable by one column",
		Run:         runSortCommand,
	})
}

func runSortCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: sort <var> <col> [asc|desc]")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	desc := false
	if len(args) >= 3 {
		desc, err = parseSortDirection(args[2])
		if err != nil {
			return fmt.Errorf("sort: %v", err)
		}
	}
	if len(args) > 3 {
		return fmt.Errorf("sort: unexpected argument %q (usage: sort <var> <col> [asc|desc])", args[3])
	}

	name, number, err := resolveColumn("sort", table, args[1])
	if err != nil {
		return err
	}
	config := insyra.DataTableSortConfig{ColumnNumber: number, ColumnName: name, Descending: desc}
	table.SortBy(config)
	if err := checkTableErr("sort", table); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "sorted")
	return nil
}
