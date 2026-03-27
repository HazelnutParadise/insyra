package commands

import (
	"fmt"
	"strconv"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:               "show",
		Usage:              "show <var> [N] [M]",
		Description:        "Display data with optional range (supports negative and _) ",
		DisableFlagParsing: true,
		Run:                runShowCommand,
	})
}

func runShowCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: show <var> [N] [M]")
	}
	switch args[0] {
	case "accel.devices":
		return runAccelCommand(ctx, []string{"devices"})
	case "accel.cache":
		return runAccelCommand(ctx, []string{"cache"})
	}
	value, exists := ctx.Vars[args[0]]
	if !exists {
		return fmt.Errorf("variable not found: %s", args[0])
	}

	rangeArgs, err := parseRangeArgs(args[1:])
	if err != nil {
		return err
	}

	switch typed := value.(type) {
	case *insyra.DataTable:
		typed.ShowRange(rangeArgs...)
	case *insyra.DataList:
		typed.ShowRange(rangeArgs...)
	default:
		return fmt.Errorf("show is only supported for DataTable/DataList")
	}

	return nil
}

func parseRangeArgs(args []string) ([]any, error) {
	if len(args) == 0 {
		return []any{}, nil
	}
	if len(args) > 2 {
		return nil, fmt.Errorf("show accepts at most two range args")
	}
	result := make([]any, 0, len(args))
	for _, token := range args {
		if token == "_" {
			result = append(result, nil)
			continue
		}
		number, err := strconv.Atoi(token)
		if err != nil {
			return nil, fmt.Errorf("invalid range value: %s", token)
		}
		result = append(result, number)
	}
	return result, nil
}
