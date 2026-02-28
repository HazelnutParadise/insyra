package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/HazelnutParadise/insyra/cli/env"
)

func init() {
	_ = Register(&CommandHandler{
		Name:               "env",
		Usage:              "env <create|list|open|clear|export|delete|rename|info> [args]",
		Description:        "Environment management",
		DisableFlagParsing: false,
		Run:                runEnvCommand,
	})
}

func runEnvCommand(ctx *ExecContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: env <create|list|open|clear|export|delete|rename|info> [args]")
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: env create <name>")
		}
		if err := env.Create(args[1]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(ctx.Output, "created environment: %s\n", args[1])
		return nil
	case "list":
		items, err := env.List()
		if err != nil {
			return err
		}
		if len(items) == 0 {
			_, _ = fmt.Fprintln(ctx.Output, "no environments found")
			return nil
		}
		for _, item := range items {
			lastAccess := "-"
			if !item.LastAccess.IsZero() {
				lastAccess = item.LastAccess.Format(time.RFC3339)
			}
			marker := " "
			if item.Name == ctx.EnvName {
				marker = "*"
			}
			_, _ = fmt.Fprintf(ctx.Output, "%s %s (vars=%d, lastAccess=%s)\n", marker, item.Name, item.VariableCount, lastAccess)
		}
		return nil
	case "open":
		if len(args) < 2 {
			return fmt.Errorf("usage: env open <name>")
		}
		envPath, err := env.Open(args[1])
		if err != nil {
			return err
		}
		ctx.EnvName = args[1]
		ctx.EnvPath = envPath
		vars, err := env.RestoreVariables(args[1])
		if err == nil {
			ctx.Vars = vars
		}
		_, _ = fmt.Fprintf(ctx.Output, "opened environment: %s\n", args[1])
		if !ctx.InREPL && ctx.OpenREPL != nil {
			return ctx.OpenREPL(ctx)
		}
		return nil
	case "clear":
		name, keepHistory, err := parseEnvClearArgs(ctx, args[1:])
		if err != nil {
			return err
		}
		if err := env.Clear(name, keepHistory); err != nil {
			return err
		}
		if name == ctx.EnvName {
			ctx.Vars = map[string]any{}
		}
		if keepHistory {
			_, _ = fmt.Fprintf(ctx.Output, "cleared environment variables: %s (history kept)\n", name)
		} else {
			_, _ = fmt.Fprintf(ctx.Output, "cleared environment: %s\n", name)
		}
		return nil
	case "export":
		name, out, err := parseEnvExportArgs(ctx, args[1:])
		if err != nil {
			return err
		}
		if err := env.Export(name, out); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(ctx.Output, "exported environment %s -> %s\n", name, out)
		return nil
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: env delete <name>")
		}
		if args[1] == ctx.EnvName {
			return fmt.Errorf("cannot delete current environment: %s", args[1])
		}
		if err := env.Delete(args[1]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(ctx.Output, "deleted environment: %s\n", args[1])
		return nil
	case "rename":
		if len(args) < 3 {
			return fmt.Errorf("usage: env rename <old> <new>")
		}
		if err := env.Rename(args[1], args[2]); err != nil {
			return err
		}
		if ctx.EnvName == args[1] {
			ctx.EnvName = args[2]
			if envPath, err := env.ResolveEnvPath(args[2]); err == nil {
				ctx.EnvPath = envPath
			}
		}
		_, _ = fmt.Fprintf(ctx.Output, "renamed environment: %s -> %s\n", args[1], args[2])
		return nil
	case "info":
		name := ctx.EnvName
		if len(args) > 1 {
			name = args[1]
		}
		if name == "" {
			name = "default"
		}
		item, err := env.Info(name)
		if err != nil {
			return err
		}
		lastAccess := "-"
		if !item.LastAccess.IsZero() {
			lastAccess = item.LastAccess.Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(ctx.Output, "name: %s\npath: %s\nvars: %d\nlastAccess: %s\n", item.Name, item.Path, item.VariableCount, lastAccess)
		return nil
	default:
		return fmt.Errorf("unknown env subcommand: %s", sub)
	}
}

func parseEnvClearArgs(ctx *ExecContext, args []string) (string, bool, error) {
	name := ctx.EnvName
	if name == "" {
		name = "default"
	}
	keepHistory := false
	nameProvided := false

	for _, arg := range args {
		switch arg {
		case "--keep-history":
			keepHistory = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", false, fmt.Errorf("unknown flag for env clear: %s", arg)
			}
			if nameProvided {
				return "", false, fmt.Errorf("usage: env clear [name] [--keep-history]")
			}
			name = arg
			nameProvided = true
		}
	}

	return name, keepHistory, nil
}

func parseEnvExportArgs(ctx *ExecContext, args []string) (string, string, error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("usage: env export [name] <file>")
	}

	if len(args) == 1 {
		name := ctx.EnvName
		if name == "" {
			name = "default"
		}
		return name, args[0], nil
	}

	if len(args) == 2 {
		return args[0], args[1], nil
	}

	return "", "", fmt.Errorf("usage: env export [name] <file>")
}
