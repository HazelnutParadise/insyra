package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "drop",
		Usage:       "drop <var>",
		Description: "Delete variable",
		Run:         runDropCommand,
	})
}

func runDropCommand(ctx *ExecContext, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: drop <var>")
	}
	if _, exists := ctx.Vars[args[0]]; !exists {
		return fmt.Errorf("variable not found: %s", args[0])
	}
	delete(ctx.Vars, args[0])
	_, _ = fmt.Fprintf(ctx.Output, "dropped: %s\n", args[0])
	return nil
}
