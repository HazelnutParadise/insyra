package cli

import (
	"strings"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/commands"
	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/HazelnutParadise/insyra/cli/repl"
	"github.com/spf13/cobra"
)

var (
	flagEnv      string
	flagNoColor  bool
	flagLogLevel string
)

func Execute() error {
	return NewRootCommand().Execute()
}

func NewRootCommand() *cobra.Command {
	execCtx := &commands.ExecContext{
		OpenREPL: repl.Start,
	}
	cmd := &cobra.Command{
		Use:          "insyra",
		Short:        "Insyra CLI and REPL for data analysis",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := env.EnsureDefaultEnvironment(); err != nil {
				return err
			}

			envPath, err := env.Open(flagEnv)
			if err != nil {
				return err
			}

			execCtx.EnvName = flagEnv
			execCtx.EnvPath = envPath

			vars, err := env.RestoreVariables(flagEnv)
			if err != nil {
				execCtx.Vars = map[string]any{}
			} else {
				execCtx.Vars = vars
			}

			applyRuntimeConfig(flagNoColor, flagLogLevel)
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if execCtx.EnvName == "" {
				return nil
			}
			if execCtx.InREPL {
				return nil
			}
			return env.SaveState(execCtx.EnvName, execCtx.Vars)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return repl.Start(execCtx)
		},
	}

	cmd.PersistentFlags().StringVar(&flagEnv, "env", "default", "Environment name")
	cmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "info", "Log level: debug|info|warning|fatal")

	for _, sub := range commands.BuildCobraCommands(execCtx) {
		cmd.AddCommand(sub)
	}

	return cmd
}

func applyRuntimeConfig(noColor bool, logLevel string) {
	insyra.Config.SetUseColoredOutput(!noColor)

	switch strings.ToLower(logLevel) {
	case "debug":
		insyra.Config.SetLogLevel(insyra.LogLevelDebug)
	case "warning", "warn":
		insyra.Config.SetLogLevel(insyra.LogLevelWarning)
	case "fatal":
		insyra.Config.SetLogLevel(insyra.LogLevelFatal)
	default:
		insyra.Config.SetLogLevel(insyra.LogLevelInfo)
	}
}
