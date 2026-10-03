package commands

import (
	"errors"
	"fmt"
)

// ErrExit is wrapped by every error exit returns. The REPL and run stop when a
// command returns it; a program embedding the DSL can test for it with
// errors.Is to end its own loop.
var ErrExit = errors.New("exit")

func init() {
	_ = Register(&CommandHandler{
		Name:        "exit",
		Args:        MaxArgs(0),
		Aliases:     []string{"quit"},
		Usage:       "exit",
		Description: "End the REPL or the running script",
		Run: func(ctx *ExecContext, args []string) error {
			if ctx.InREPL || ctx.scriptDepth > 0 {
				return ErrExit
			}
			// One-shot `insyra exit` and a single Session.Execute("exit")
			// have nothing to end, so succeeding would say it did something.
			return fmt.Errorf("%w only ends the REPL or a script run by `run`; there is nothing to end here", ErrExit)
		},
	})
}
