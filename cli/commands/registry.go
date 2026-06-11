package commands

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/spf13/cobra"
)

type ExecContext struct {
	Vars     map[string]any
	DBConns  map[string]*DBConn
	EnvName  string
	EnvPath  string
	Output   io.Writer
	InREPL   bool
	OpenREPL func(ctx *ExecContext) error
	// Env is the per-session environment Manager. Commands MUST go through
	// it (ctx.Env.SaveState etc.) so embedders that supply a per-workspace
	// Manager don't get their writes redirected to the default ~/.insyra.
	// Dispatch fills this with env.Default() if the caller left it nil.
	Env *env.Manager
}

type CommandHandler struct {
	Name               string
	Aliases            []string
	Usage              string
	Description        string
	// Forms lists the major sub-shapes of a command (one entry per shape).
	// Each entry is rendered as-is under a "Forms:" header by `help <cmd>`.
	// Use for commands like `ttest single|two|paired` where the bare Usage
	// can't enumerate every form.
	Forms []string
	// Examples lists ready-to-run invocations rendered under "Examples:" by
	// `help <cmd>`. Each entry should be a complete `insyra ...` line that
	// the user can copy into a shell.
	Examples           []string
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
	if ctx.Env == nil {
		ctx.Env = env.Default()
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
				runArgs := args
				if localHandler.Name == "env" && len(args) > 0 && strings.EqualFold(args[0], "clear") {
					keepHistory, flagErr := cmd.Flags().GetBool("keep-history")
					if flagErr == nil && keepHistory {
						runArgs = append(runArgs, "--keep-history")
					}
				}
				if localHandler.Name == "env" && len(args) > 0 && strings.EqualFold(args[0], "import") {
					force, flagErr := cmd.Flags().GetBool("force")
					if flagErr == nil && force {
						runArgs = append(runArgs, "--force")
					}
				}
				err := Dispatch(ctx, localHandler.Name, runArgs)
				envName := ctx.EnvName
				if envName == "" {
					envName = "default"
				}
				if localHandler.Name != "history" {
					line := localHandler.Name
					if len(runArgs) > 0 {
						line += " " + strings.Join(runArgs, " ")
					}
					mgr := ctx.Env
					if mgr == nil {
						mgr = env.Default()
					}
					_ = mgr.AppendHistory(envName, line)
				}
				return err
			},
		})

		created := commands[len(commands)-1]
		if localHandler.Name == "env" {
			created.Flags().Bool("keep-history", false, "With 'env clear', keep command history")
			created.Flags().Bool("force", false, "With 'env import', overwrite non-empty target environment")
		}
	}

	return commands
}
