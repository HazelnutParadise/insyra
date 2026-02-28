package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "transpose",
		Usage:       "transpose <var> [as <var>]",
		Description: "Transpose DataTable",
		Run:         runTransposeCommand,
	})
}

func runTransposeCommand(ctx *ExecContext, args []string) error {
	coreArgs, alias := parseAlias(args)
	if len(coreArgs) < 1 {
		return fmt.Errorf("usage: transpose <var> [as <var>]")
	}
	table, err := getDataTableVar(ctx, coreArgs[0])
	if err != nil {
		return err
	}
	result := table.Clone().Transpose()
	ctx.Vars[alias] = result
	_, _ = fmt.Fprintf(ctx.Output, "saved transposed table as %s\n", alias)
	return nil
}
