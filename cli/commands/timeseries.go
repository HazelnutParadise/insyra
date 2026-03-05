package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{Name: "movavg", Usage: "movavg <var> <window> [as <var>]", Description: "Moving average", Run: runMovAvgCommand})
	_ = Register(&CommandHandler{Name: "expsmooth", Usage: "expsmooth <var> <alpha> [as <var>]", Description: "Exponential smoothing", Run: runExpSmoothCommand})
	_ = Register(&CommandHandler{Name: "diff", Usage: "diff <var> [as <var>]", Description: "Difference", Run: runDiffCommand})
	_ = Register(&CommandHandler{Name: "fillnan", Usage: "fillnan <var> mean", Description: "Fill NaN with mean", Run: runFillNaNCommand})
}

func runMovAvgCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: movavg <var> <window> [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	window, err := strconv.Atoi(coreArgs[1])
	if err != nil {
		return err
	}
	ctx.Vars[alias] = dl.Clone().MovingAverage(window)
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runExpSmoothCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: expsmooth <var> <alpha> [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	alpha, err := strconv.ParseFloat(coreArgs[1], 64)
	if err != nil {
		return err
	}
	ctx.Vars[alias] = dl.Clone().ExponentialSmoothing(alpha)
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runDiffCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: diff <var> [as <var>]")
	}
	dl, err := getDataListVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	ctx.Vars[alias] = dl.Clone().Difference()
	_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", alias)
	return nil
}

func runFillNaNCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: fillnan <var> mean")
	}
	if args[1] != "mean" {
		return fmt.Errorf("only mean is supported")
	}
	dl, err := getDataListVar(ctx, args[0])
	if err != nil {
		return err
	}
	dl.FillNaNWithMean()
	_, _ = fmt.Fprintln(ctx.Output, "filled NaN with mean")
	return nil
}
