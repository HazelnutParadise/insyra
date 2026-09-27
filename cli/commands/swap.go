package commands

import (
	"fmt"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "swap",
		Args:        MaxArgs(4),
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
		aPos, err := resolveColumnToken("swap", table, a)
		if err != nil {
			return err
		}
		bPos, err := resolveColumnToken("swap", table, b)
		if err != nil {
			return err
		}
		table.SwapColsByNumber(aPos, bPos)
	case "row":
		aPos, err := resolveRowToken("swap", table, a)
		if err != nil {
			return err
		}
		bPos, err := resolveRowToken("swap", table, b)
		if err != nil {
			return err
		}
		table.SwapRowsByIndex(aPos, bPos)
	default:
		return fmt.Errorf("dimension must be col or row")
	}
	if err := checkTableErr("swap", table); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(ctx.Output, "swap complete")
	return nil
}
