package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "read",
		Usage:       "read <file> [headers true|false] [rownames true|false] [encoding <enc>] [sheet <name>]",
		Description: "Quick preview a file without saving variable",
		Run:         runReadCommand,
	})
}

func runReadCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: read <file> [headers true|false] [rownames true|false] [encoding <enc>] [sheet <name>]")
	}
	fakeArgs := append([]string(nil), args...)
	fakeArgs = append(fakeArgs, "as", "$preview")
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
