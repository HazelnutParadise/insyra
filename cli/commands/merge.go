package commands

import (
	"fmt"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{Name: "merge", Usage: "merge <var1> <var2> <direction> <mode> [on <cols>] [as <var>]", Description: "Merge two DataTables", Run: runMergeCommand})
}

func runMergeCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 4 {
		return fmt.Errorf("usage: merge <var1> <var2> <direction> <mode> [on <cols>] [as <var>]")
	}
	left, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	right, err := getDataTableVar(ctx, coreArgs[1])
	if err != nil {
		return err
	}
	direction, err := parseMergeDirection(coreArgs[2])
	if err != nil {
		return err
	}
	mode, err := parseMergeMode(coreArgs[3])
	if err != nil {
		return err
	}
	onColumns := []string{}
	for index := 4; index < len(coreArgs); index++ {
		if coreArgs[index] == "on" {
			onColumns = append(onColumns, coreArgs[index+1:]...)
			break
		}
	}
	result, err := left.Merge(right, direction, mode, onColumns...)
	if err != nil {
		return err
	}
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "merged into %s\n", alias)
	return nil
}

func parseMergeDirection(raw string) (insyra.MergeDirection, error) {
	switch strings.ToLower(raw) {
	case "horizontal":
		return insyra.MergeDirectionHorizontal, nil
	case "vertical":
		return insyra.MergeDirectionVertical, nil
	default:
		return 0, fmt.Errorf("invalid merge direction: %s", raw)
	}
}

func parseMergeMode(raw string) (insyra.MergeMode, error) {
	switch strings.ToLower(raw) {
	case "inner":
		return insyra.MergeModeInner, nil
	case "outer":
		return insyra.MergeModeOuter, nil
	case "left":
		return insyra.MergeModeLeft, nil
	case "right":
		return insyra.MergeModeRight, nil
	default:
		return 0, fmt.Errorf("invalid merge mode: %s", raw)
	}
}
