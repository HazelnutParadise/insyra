package commands

import (
	"fmt"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "help",
		Usage:       "help [command]",
		Description: "Show command help",
		Run:         runHelpCommand,
	})
}

func runHelpCommand(ctx *ExecContext, args []string) error {
	if len(args) == 1 {
		handler, ok := LookupCommand(args[0])
		if !ok {
			return fmt.Errorf("unknown command: %s", args[0])
		}
		_, _ = fmt.Fprintf(ctx.Output, "%s\nusage: %s\n", handler.Description, handler.Usage)
		if len(handler.Forms) > 0 {
			_, _ = fmt.Fprintln(ctx.Output, "\nForms:")
			for _, form := range handler.Forms {
				_, _ = fmt.Fprintf(ctx.Output, "  %s\n", form)
			}
		}
		if len(handler.Examples) > 0 {
			_, _ = fmt.Fprintln(ctx.Output, "\nExamples:")
			for _, ex := range handler.Examples {
				_, _ = fmt.Fprintf(ctx.Output, "  %s\n", ex)
			}
		}
		return nil
	}

	keys, handlers := SnapshotRegistry()

	// Size the name column to the longest name rather than a fixed 12, which
	// pushed the description out of line for knn_neighbors and anything else
	// past that width.
	width := 0
	for _, key := range keys {
		if len(key) > width {
			width = len(key)
		}
	}

	_, _ = fmt.Fprintln(ctx.Output, "available commands:")
	for _, key := range keys {
		_, _ = fmt.Fprintf(ctx.Output, "  %-*s  %s\n", width, key, handlers[key].Description)
	}
	return nil
}
