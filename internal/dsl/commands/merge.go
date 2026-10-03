package commands

import (
	"fmt"
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{Name: "merge", Args: FormArgsAt(2, map[string]int{"horizontal": 7, "vertical": 4}).WithAlias(), Usage: "merge <var1> <var2> <direction> <mode> [on <cols>] [as <var>]", Description: "Merge two DataTables", Run: runMergeCommand})
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
		// Merge joins on names, so each token is resolved against its own
		// table (the first against left, the second, or the same one, against
		// right) and handed over as that column's name.
		tables := []*insyra.DataTable{left, right}
		for i, token := range coreArgs[5:] {
			table := tables[min(i, 1)]
			pos, err := resolveColumnToken("merge", table, token)
			if err != nil {
				return err
			}
			name := table.ColNames()[pos]
			if name == "" {
				return fmt.Errorf("merge: %s is an unnamed column; merge joins on a column name", token)
			}
			onColumns = append(onColumns, name)
		}
		if len(onColumns) == 1 {
			// One token names the key in both tables.
			pos, err := resolveColumnToken("merge", right, coreArgs[5])
			if err != nil {
				return err
			}
			if right.ColNames()[pos] != onColumns[0] {
				return fmt.Errorf("merge: %s is %q in the first table but %q in the second; give both: on %s %s", coreArgs[5], onColumns[0], right.ColNames()[pos], onColumns[0], right.ColNames()[pos])
			}
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
