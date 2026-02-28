package commands

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/cli/env"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "history",
		Usage:       "history",
		Description: "Show command history",
		Run:         runHistoryCommand,
	})
}

func runHistoryCommand(ctx *ExecContext, args []string) error {
	if ctx.EnvName == "" {
		ctx.EnvName = "default"
	}
	entries, err := env.ReadHistory(ctx.EnvName)
	if err != nil {
		return err
	}
	for index, entry := range entries {
		_, _ = fmt.Fprintf(ctx.Output, "%4d  %s\n", index+1, entry)
	}
	return nil
}
