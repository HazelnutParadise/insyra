package commands

import (
	"strings"

	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/HazelnutParadise/insyra/internal/dsl/commands"
	"github.com/spf13/cobra"
)

// BuildCobraCommands makes one shell command for every registered command, in
// name order. Each runs its command against ctx, hands it the flags it
// declares in Flags, and records the line in the environment's history, except
// for history itself.
func BuildCobraCommands(ctx *ExecContext) []*cobra.Command {
	names, handlers := commands.SnapshotRegistry()
	return buildCobraCommands(ctx, names, handlers, commands.Dispatch)
}

// buildCobraCommands is BuildCobraCommands over the given handlers, running
// each through dispatch.
func buildCobraCommands(ctx *ExecContext, names []string, handlers map[string]*CommandHandler, dispatch func(*ExecContext, string, []string) error) []*cobra.Command {
	built := make([]*cobra.Command, 0, len(names))
	for _, name := range names {
		handler := handlers[name]

		use := handler.Usage
		if use == "" {
			use = handler.Name
		}

		cmd := &cobra.Command{
			Use:                use,
			Aliases:            handler.Aliases,
			Short:              handler.Description,
			DisableFlagParsing: handler.DisableFlagParsing,
			RunE: func(cmd *cobra.Command, args []string) error {
				runArgs := args
				for _, flag := range handler.Flags {
					runArgs = appendFlag(cmd, flag, args, runArgs)
				}
				err := dispatch(ctx, handler.Name, runArgs)
				envName := ctx.EnvName
				if envName == "" {
					envName = "default"
				}
				if handler.Name != "history" {
					line := handler.Name
					if len(runArgs) > 0 {
						line += " " + strings.Join(runArgs, " ")
					}
					mgr := ctx.Env
					if mgr == nil {
						mgr = env.Default()
					}
					_ = mgr.AppendHistory(envName, SanitizeHistoryLine(line))
				}
				return err
			},
		}
		for _, flag := range handler.Flags {
			defineFlag(cmd, flag)
		}
		built = append(built, cmd)
	}
	return built
}

// formListed reports whether form is one of the words in a flag's Form, which
// separates several with |, as in a Usage line.
func formListed(forms, form string) bool {
	for _, word := range strings.Split(forms, "|") {
		if strings.EqualFold(form, word) {
			return true
		}
	}
	return false
}

// defineFlag adds flag to cmd.
func defineFlag(cmd *cobra.Command, flag CommandFlag) {
	if flag.TakesValue {
		cmd.Flags().String(flag.Name, "", flag.Usage)
		return
	}
	cmd.Flags().Bool(flag.Name, false, flag.Usage)
}

// appendFlag returns runArgs with flag appended when it is set and applies to
// the form args name.
func appendFlag(cmd *cobra.Command, flag CommandFlag, args, runArgs []string) []string {
	if flag.Form != "" && (len(args) == 0 || !formListed(flag.Form, args[0])) {
		return runArgs
	}
	if flag.TakesValue {
		value, err := cmd.Flags().GetString(flag.Name)
		if err == nil && strings.TrimSpace(value) != "" {
			runArgs = append(runArgs, "--"+flag.Name, value)
		}
		return runArgs
	}
	if set, err := cmd.Flags().GetBool(flag.Name); err == nil && set {
		runArgs = append(runArgs, "--"+flag.Name)
	}
	return runArgs
}
