package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "swap",
		Usage:       "swap <var> col|row <a> <b>",
		Description: "Swap DataTable columns or rows",
		Run:         runSwapCommand,
	})
}

func runSwapCommand(ctx *ExecContext, args []string) error {
	if len(args) < 4 {
		return fmt.Errorf("usage: swap <var> col|row <a> <b>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	dimension := args[1]
	a := args[2]
	b := args[3]

	switch dimension {
	case "col":
		if aIndex, errA := strconv.Atoi(a); errA == nil {
			if bIndex, errB := strconv.Atoi(b); errB == nil {
				table.SwapColsByNumber(aIndex, bIndex)
			} else {
				return fmt.Errorf("both col selectors must be numeric or both names")
			}
		} else {
			table.SwapColsByName(a, b)
		}
	case "row":
		if aIndex, errA := strconv.Atoi(a); errA == nil {
			if bIndex, errB := strconv.Atoi(b); errB == nil {
				table.SwapRowsByIndex(aIndex, bIndex)
			} else {
				return fmt.Errorf("both row selectors must be numeric or both names")
			}
		} else {
			table.SwapRowsByName(a, b)
		}
	default:
		return fmt.Errorf("dimension must be col or row")
	}
	_, _ = fmt.Fprintln(ctx.Output, "swap complete")
	return nil
}
