package commands

import "fmt"

func init() {
	_ = Register(&CommandHandler{
		Name:        "clear",
		Args:        MaxArgs(0),
		Usage:       "clear",
		Description: "Clear terminal screen",
		Run: func(ctx *ExecContext, args []string) error {
			_, _ = fmt.Fprint(ctx.Output, "\033[2J\033[H")
			return nil
		},
	})
}
