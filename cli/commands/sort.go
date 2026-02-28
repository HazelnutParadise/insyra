package commands

import (
	"fmt"
	"strconv"
	"strings"

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
	selector := args[1]
	desc := len(args) >= 3 && strings.EqualFold(args[2], "desc")

	config := insyra.DataTableSortConfig{ColumnNumber: -1, Descending: desc}
	if number, convErr := strconv.Atoi(selector); convErr == nil {
		config.ColumnNumber = number
	} else {
		config.ColumnName = selector
	}
	table.SortBy(config)
	_, _ = fmt.Fprintln(ctx.Output, "sorted")
	return nil
}
