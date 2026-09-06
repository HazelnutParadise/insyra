package cli

import (
	"fmt"
	"strconv"
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
		Env:      env.Default(),
	}
	cmd := &cobra.Command{
		Use:          "insyra",
		Short:        "Insyra CLI and REPL for data analysis",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return openEnvironment(execCtx)
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			if execCtx.EnvName == "" {
				return nil
			}
			if execCtx.InREPL {
				return nil
			}
			return execCtx.Env.SaveState(execCtx.EnvName, execCtx.Vars)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return repl.Start(execCtx)
		},
	}

	cmd.PersistentFlags().StringVar(&flagEnv, "env", "default", "Environment name")
	cmd.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	cmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "info", "Log level: debug|info|warning|fatal")

	for _, sub := range commands.BuildCobraCommands(execCtx) {
		if sub.DisableFlagParsing {
			wrapRawArgCommand(sub, execCtx)
		}
		cmd.AddCommand(sub)
	}

	return cmd
}

// openEnvironment applies the root flags: opens (creating if needed) the
// selected environment, restores its variables and sets colour/log level.
func openEnvironment(execCtx *commands.ExecContext) error {
	if err := execCtx.Env.EnsureDefaultEnvironment(); err != nil {
		return err
	}

	envPath, err := execCtx.Env.Open(flagEnv)
	if err != nil {
		return err
	}

	execCtx.EnvName = flagEnv
	execCtx.EnvPath = envPath

	vars, err := execCtx.Env.RestoreVariables(flagEnv)
	if err != nil {
		execCtx.Vars = map[string]any{}
	} else {
		execCtx.Vars = vars
	}

	applyRuntimeConfig(flagNoColor, flagLogLevel)
	return nil
}

// wrapRawArgCommand handles root flags for a subcommand that turns Cobra's
// flag parsing off (newdl, addcol, addrow, show take raw values such as -1).
// Cobra then hands `--env e2 newdl 1 2` to the command with the root flags
// still in front, so they are peeled off here and applied before the
// command runs; otherwise they would be stored as data in the default
// environment.
func wrapRawArgCommand(sub *cobra.Command, execCtx *commands.ExecContext) {
	original := sub.RunE
	sub.RunE = func(cmd *cobra.Command, args []string) error {
		rest, changed, err := consumeRootFlags(args)
		if err != nil {
			return err
		}
		if changed {
			if err := openEnvironment(execCtx); err != nil {
				return err
			}
		}
		return original(cmd, rest)
	}
}

// consumeRootFlags strips leading --env/--no-color/--log-level flags from
// args, updates the flag variables, and reports whether anything changed.
func consumeRootFlags(args []string) ([]string, bool, error) {
	changed := false
	for len(args) > 0 {
		arg := args[0]
		name, value, hasValue := strings.Cut(arg, "=")
		takeValue := func() (string, error) {
			if hasValue {
				return value, nil
			}
			if len(args) < 2 {
				return "", fmt.Errorf("flag %s requires a value", name)
			}
			v := args[1]
			args = args[1:]
			return v, nil
		}
		switch name {
		case "--env":
			v, err := takeValue()
			if err != nil {
				return nil, false, err
			}
			flagEnv = v
		case "--log-level":
			v, err := takeValue()
			if err != nil {
				return nil, false, err
			}
			flagLogLevel = v
		case "--no-color":
			flagNoColor = true
			if hasValue {
				b, err := strconv.ParseBool(value)
				if err != nil {
					return nil, false, fmt.Errorf("invalid value for --no-color: %q", value)
				}
				flagNoColor = b
			}
		default:
			return args, changed, nil
		}
		changed = true
		args = args[1:]
	}
	return args, changed, nil
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
