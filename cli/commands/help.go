package commands

import (
	"fmt"
	"sort"
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
		handler, ok := Registry[args[0]]
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

	keys := make([]string, 0, len(Registry))
	for key := range Registry {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	_, _ = fmt.Fprintln(ctx.Output, "available commands:")
	for _, key := range keys {
		handler := Registry[key]
		_, _ = fmt.Fprintf(ctx.Output, "  %-12s %s\n", key, handler.Description)
	}
	return nil
}
