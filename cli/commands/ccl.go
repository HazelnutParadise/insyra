package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{Name: "ccl", Usage: "ccl <var> <expression>", Description: "Execute CCL statements on DataTable", Run: runCCLCommand})
	_ = Register(&CommandHandler{Name: "addcolccl", Usage: "addcolccl <var> <name> <expr>", Description: "Add DataTable column using CCL", Run: runAddColCCLCommand})
}

func runCCLCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: ccl <var> <expression>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	table.ExecuteCCL(args[1])
	_, _ = fmt.Fprintln(ctx.Output, "ccl executed")
	return nil
}

func runAddColCCLCommand(ctx *ExecContext, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: addcolccl <var> <name> <expr>")
	}
	table, err := getDataTableVar(ctx, args[0])
	if err != nil {
		return err
	}
	table.AddColUsingCCL(args[1], args[2])
	_, _ = fmt.Fprintln(ctx.Output, "column added by ccl")
	return nil
}
