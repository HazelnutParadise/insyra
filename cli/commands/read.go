package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "read",
		Usage:       "read <file>",
		Description: "Quick preview a file without saving variable",
		Run:         runReadCommand,
	})
}

func runReadCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: read <file>")
	}
	fakeArgs := append([]string{args[0]}, "as", "$preview")
	if err := runLoadCommand(ctx, fakeArgs); err != nil {
		return err
	}
	table, err := getDataTableVar(ctx, "$preview")
	if err != nil {
		return err
	}
	table.ShowRange(10)
	delete(ctx.Vars, "$preview")
	return nil
}
