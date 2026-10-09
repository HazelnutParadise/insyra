package commands

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func init() {
	_ = Register(&CommandHandler{
		Name:               "env",
		Args:               FormArgs(map[string]int{"create": 2, "list": 1, "open": 2, "info": 2, "delete": 3, "rename": 3, "export": 3, "clear": 3, "import": 4}),
		Usage:              "env <create|list|open|clear|export|import|delete|rename|info> [args]",
		Description:        "Environment management",
		DisableFlagParsing: false,
		Flags: []CommandFlag{
			{Name: "keep-history", Usage: "With 'env clear', keep command history", Form: "clear"},
			{Name: "force", Usage: "With 'env import', replace a non-empty target environment; with 'env delete', allow deleting default", Form: "import|delete"},
		},
		Forms: []string{
			"env create <name>                      make a new environment",
			"env list                               list every environment",
			"env open <name>                        switch to one; replaces the session's variables",
			"env info [name]                        show where it lives and what is in it",
			"env clear [name] [--keep-history]      drop its variables, and its history unless --keep-history",
			"env rename <old> <new>                 rename one",
			"env delete <name> [--force]            remove one with its history, without asking",
			"                                       (not the current one; default needs --force)",
			"",
			"env export [name] <file>               write it to a file; replaces <file> if it exists",
			"env import <file> [name] [--force]     read one back; --force replaces a target that is not empty",
		},
		Examples: []string{
			"insyra env create analysis",
			"insyra env open analysis",
			"insyra env export analysis backup.zip",
			"insyra env import backup.zip restored",
			"insyra env clear analysis --keep-history",
		},
		Run: runEnvCommand,
	})
}

func runEnvCommand(ctx *ExecContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: env <create|list|open|clear|export|import|delete|rename|info> [args]")
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: env create <name>")
		}
		if err := ctx.Env.Create(args[1]); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(ctx.Output, "created environment: %s\n", args[1])
		return nil
	case "list":
		items, err := ctx.Env.List()
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
		envPath, err := ctx.Env.Open(args[1])
		if err != nil {
			return err
		}
		ctx.EnvName = args[1]
		ctx.EnvPath = envPath
		vars, err := ctx.Env.RestoreVariables(args[1])
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
		if err := ctx.Env.Clear(name, keepHistory); err != nil {
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
		if err := ctx.Env.Export(name, out); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(ctx.Output, "exported environment %s -> %s\n", name, out)
		return nil
	case "import":
		in, target, force, err := parseEnvImportArgs(args[1:])
		if err != nil {
			return err
		}
		name, err := ctx.Env.Import(in, target, force)
		if err != nil {
			return err
		}
		if name == ctx.EnvName {
			vars, restoreErr := ctx.Env.RestoreVariables(name)
			if restoreErr == nil {
				ctx.Vars = vars
			}
		}
		_, _ = fmt.Fprintf(ctx.Output, "imported environment from %s -> %s\n", in, name)
		return nil
	case "delete":
		name, force, err := parseEnvDeleteArgs(args[1:])
		if err != nil {
			return err
		}
		if sameEnvironment(ctx, name, ctx.EnvName) {
			return fmt.Errorf("cannot delete current environment: %s", name)
		}
		// default is the environment every command opens when --env is not
		// given, so deleting it by mistake loses the most.
		if !force && sameEnvironment(ctx, name, "default") {
			return fmt.Errorf("env delete: default is the environment insyra opens when --env is not given; deleting it loses its variables and history, and it comes back empty on the next command. Add --force to delete it")
		}
		if err := ctx.Env.Delete(name); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(ctx.Output, "deleted environment: %s\n", name)
		return nil
	case "rename":
		if len(args) < 3 {
			return fmt.Errorf("usage: env rename <old> <new>")
		}
		if err := ctx.Env.Rename(args[1], args[2]); err != nil {
			return err
		}
		if ctx.EnvName == args[1] {
			ctx.EnvName = args[2]
			if envPath, err := ctx.Env.ResolveEnvPath(args[2]); err == nil {
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
		item, err := ctx.Env.Info(name)
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

// sameEnvironment reports whether a and b name one environment directory. On a
// file system that ignores case, Default and default are the same directory,
// so comparing the names alone would let a refusal be spelled around.
func sameEnvironment(ctx *ExecContext, a, b string) bool {
	if a == b {
		return true
	}
	pathA, errA := ctx.Env.ResolveEnvPath(a)
	pathB, errB := ctx.Env.ResolveEnvPath(b)
	if errA != nil || errB != nil {
		return false
	}
	infoA, errA := os.Stat(pathA)
	infoB, errB := os.Stat(pathB)
	return errA == nil && errB == nil && os.SameFile(infoA, infoB)
}

func parseEnvDeleteArgs(args []string) (string, bool, error) {
	name := ""
	force := false
	for _, arg := range args {
		switch arg {
		case "--force":
			force = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", false, fmt.Errorf("unknown flag for env delete: %s", arg)
			}
			if name != "" {
				return "", false, fmt.Errorf("usage: env delete <name> [--force]")
			}
			name = arg
		}
	}
	if name == "" {
		return "", false, fmt.Errorf("usage: env delete <name> [--force]")
	}
	return name, force, nil
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

func parseEnvImportArgs(args []string) (string, string, bool, error) {
	if len(args) == 0 {
		return "", "", false, fmt.Errorf("usage: env import <file> [name] [--force]")
	}

	input := args[0]
	target := ""
	force := false
	remain := args[1:]

	for i := 0; i < len(remain); i++ {
		arg := remain[i]
		switch arg {
		case "--force":
			force = true
		default:
			if strings.HasPrefix(arg, "--") {
				return "", "", false, fmt.Errorf("unknown flag for env import: %s", arg)
			}
			if target != "" {
				return "", "", false, fmt.Errorf("usage: env import <file> [name] [--force]")
			}
			target = arg
		}
	}

	return input, target, force, nil
}
