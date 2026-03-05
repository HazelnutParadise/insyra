package commands

import (
	"fmt"
	"strconv"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "sample",
		Usage:       "sample <var> <n> [as <var>]",
		Description: "Simple random sample from DataTable",
		Run:         runSampleCommand,
	})
}

func runSampleCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 2 {
		return fmt.Errorf("usage: sample <var> <n> [as <var>]")
	}
	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	n, err := strconv.Atoi(coreArgs[1])
	if err != nil {
		return fmt.Errorf("invalid sample size: %s", coreArgs[1])
	}
	ctx.Vars[alias] = table.SimpleRandomSample(n)
	_, _ = fmt.Fprintf(ctx.Output, "saved sample as %s\n", alias)
	return nil
}
