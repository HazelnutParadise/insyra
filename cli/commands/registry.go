package commands

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

type ExecContext struct {
	Vars     map[string]any
	EnvName  string
	EnvPath  string
	Output   io.Writer
	InREPL   bool
	OpenREPL func(ctx *ExecContext) error
}

type CommandHandler struct {
	Name               string
	Aliases            []string
	Usage              string
	Description        string
	DisableFlagParsing bool
	Run                func(ctx *ExecContext, args []string) error
}

var Registry = map[string]*CommandHandler{}

func Register(handler *CommandHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is nil")
	}
	if handler.Name == "" {
		return fmt.Errorf("handler name is required")
	}
	if handler.Run == nil {
		return fmt.Errorf("handler run function is required")
	}
	if _, exists := Registry[handler.Name]; exists {
		return fmt.Errorf("command already registered: %s", handler.Name)
	}
	Registry[handler.Name] = handler
	return nil
}

func Dispatch(ctx *ExecContext, name string, args []string) error {
	handler, ok := Registry[name]
	if !ok {
		return fmt.Errorf("unknown command: %s", name)
	}
	if ctx == nil {
		ctx = &ExecContext{}
	}
	if ctx.Vars == nil {
		ctx.Vars = map[string]any{}
	}
	if ctx.Output == nil {
		ctx.Output = os.Stdout
	}
	return handler.Run(ctx, args)
}

func BuildCobraCommands(ctx *ExecContext) []*cobra.Command {
	keys := make([]string, 0, len(Registry))
	for name := range Registry {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	commands := make([]*cobra.Command, 0, len(keys))
	for _, name := range keys {
		handler := Registry[name]
		localHandler := handler

		use := localHandler.Usage
		if use == "" {
			use = localHandler.Name
		}

		commands = append(commands, &cobra.Command{
			Use:                use,
			Aliases:            localHandler.Aliases,
			Short:              localHandler.Description,
			DisableFlagParsing: localHandler.DisableFlagParsing,
			RunE: func(cmd *cobra.Command, args []string) error {
				return Dispatch(ctx, localHandler.Name, args)
			},
		})
	}

	return commands
}
