package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "dropcol",
		Usage:       "dropcol <var> <name|index...>",
		Description: "Drop columns by name or index",
		Run:         runDropColCommand,
	})
}

func runDropColCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: dropcol <var> <name|index...>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	names := []string{}
	indices := []int{}
	for _, token := range args[1:] {
		if index, convErr := strconv.Atoi(token); convErr == nil {
			indices = append(indices, index)
		} else {
			names = append(names, token)
		}
	}
	for _, index := range indices {
		if index < 0 || index >= table.NumCols() {
			return fmt.Errorf("dropcol: column %d is out of range (the table has %d columns)", index, table.NumCols())
		}
	}
	for _, name := range names {
		if err := requireColumnName("dropcol", table, name); err != nil {
			return err
		}
	}
	if len(indices) > 0 {
		table.DropColsByNumber(indices...)
	}
	if len(names) > 0 {
		table.DropColsByName(names...)
	}
	if err := checkTableErr("dropcol", table); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "columns dropped")
	return nil
}
