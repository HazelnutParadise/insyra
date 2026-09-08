package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "droprow",
		Usage:       "droprow <var> <index|name...>",
		Description: "Drop rows by index or name",
		Run:         runDropRowCommand,
	})
}

func runDropRowCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: droprow <var> <index|name...>")
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
		if err := requireRowIndex("droprow", table, index); err != nil {
			return err
		}
	}
	for _, name := range names {
		if err := requireRowName("droprow", table, name); err != nil {
			return err
		}
	}
	if len(indices) > 0 {
		table.DropRowsByIndex(indices...)
	}
	if len(names) > 0 {
		table.DropRowsByName(names...)
	}
	if err := checkTableErr("droprow", table); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "rows dropped")
	return nil
}
