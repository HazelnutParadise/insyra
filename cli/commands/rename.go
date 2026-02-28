package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "rename",
		Usage:       "rename <var> <new>",
		Description: "Rename variable",
		Run:         runRenameCommand,
	})
}

func runRenameCommand(ctx *ExecContext, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: rename <var> <new>")
	}
	oldName := args[0]
	newName := args[1]
	value, exists := ctx.Vars[oldName]
	if !exists {
		return fmt.Errorf("variable not found: %s", oldName)
	}
	if _, conflict := ctx.Vars[newName]; conflict {
		return fmt.Errorf("target variable already exists: %s", newName)
	}
	ctx.Vars[newName] = value
	delete(ctx.Vars, oldName)
	_, _ = fmt.Fprintf(ctx.Output, "renamed: %s -> %s\n", oldName, newName)
	return nil
}
