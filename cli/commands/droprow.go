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
	if len(indices) > 0 {
		table.DropRowsByIndex(indices...)
	}
	if len(names) > 0 {
		table.DropRowsByName(names...)
	}
	_, _ = fmt.Fprintln(ctx.Output, "rows dropped")
	return nil
}
