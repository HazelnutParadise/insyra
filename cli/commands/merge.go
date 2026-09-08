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
	if len(coreArgs) > 4 {
		if !strings.EqualFold(coreArgs[4], "on") {
			return fmt.Errorf("merge: unexpected argument %q (usage: merge <var1> <var2> <direction> <mode> [on <cols>] [as <var>])", coreArgs[4])
		}
		if len(coreArgs) == 5 {
			return fmt.Errorf("merge: `on` needs at least one column name")
		}
		onColumns = append(onColumns, coreArgs[5:]...)
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
