package repl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/cli/commands"
	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/HazelnutParadise/insyra/cli/style"
	enginedsl "github.com/HazelnutParadise/insyra/engine/dsl"
	"github.com/HazelnutParadise/insyra/internal/dsl"
	"github.com/ergochat/readline"
)

func Start(ctx *enginedsl.ExecContext) error {
	if ctx == nil {
		ctx = &enginedsl.ExecContext{}
	}
	if ctx.Env == nil {
		ctx.Env = env.Default()
	}
	ctx.InREPL = true
	defer func() {
		ctx.InREPL = false
	}()
	// The library itself prints nothing on import; the interactive REPL is the
	// one place a banner belongs.
	if ctx.Output != nil {
		fmt.Fprintf(ctx.Output, "Welcome to Insyra %s (v%s)!\nOfficial website: https://insyra.hazelnut-paradise.com\n\n", insyra.VersionName, insyra.Version)
	}
	if ctx.EnvName == "" {
		ctx.EnvName = "default"
	}
	if ctx.EnvPath == "" {
		envPath, err := ctx.Env.Open(ctx.EnvName)
		if err != nil {
			return err
		}
		ctx.EnvPath = envPath
	}
	if ctx.Vars == nil {
		vars, err := ctx.Env.RestoreVariables(ctx.EnvName)
		if err != nil {
			ctx.Vars = map[string]any{}
		} else {
			ctx.Vars = vars
		}
	}

	historyFile := filepath.Join(ctx.EnvPath, "history.txt")
	instance, err := readline.NewFromConfig(&readline.Config{
		Prompt:       prompt(ctx.EnvName),
		HistoryFile:  historyFile,
		AutoComplete: NewAutoCompleter(ctx),
		// History is saved by hand below so a `db connect` line is masked
		// before it reaches disk.
		DisableAutoSaveHistory: true,
		InterruptPrompt:        "^C",
		EOFPrompt:              "exit",
	})
	if err != nil {
		return err
	}
	// readline creates the file with the process umask; history can hold
	// data paths and connection strings, so keep it private to the user.
	_ = os.Chmod(historyFile, 0o600)
	defer func() {
		_ = instance.Close()
	}()
	defer func() {
		_ = commands.SaveEnvState(ctx)
	}()
	defer commands.CloseAllDBConns(ctx)

	for {
		line, err := instance.ReadLine()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if err != nil {
			return nil
		}
		trimmed := strings.TrimSpace(line)
		if entry, ok := historyEntry(line); ok {
			_ = instance.SaveToHistory(entry)
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		tokens := dsl.Tokenize(trimmed)
		if len(tokens) == 0 {
			continue
		}

		if err := commands.Dispatch(ctx, tokens[0], tokens[1:]); err != nil {
			if errors.Is(err, enginedsl.ErrExit) {
				return nil
			}
			_, _ = fmt.Fprintln(instance.Stderr(), style.ErrorText(err.Error()))
		}

		// Do NOT AppendHistory here: readline already persists each entered line
		// to HistoryFile (the same env history.txt), so an explicit append would
		// write every command twice. The one-shot CLI path keeps its own
		// AppendHistory since it does not go through readline.
		_ = commands.SaveEnvState(ctx)

		if ctx.EnvName != "" {
			instance.SetPrompt(prompt(ctx.EnvName))
		}
	}
}

// historyEntry returns what an entered line adds to history.txt, with any
// database password masked. Every non-empty line is saved, comments, exit and
// lines without tokens included, as readline's automatic saving did before
// history was saved by hand; the caller saves it before deciding what the line
// does.
func historyEntry(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return "", false
	}
	return commands.SanitizeHistoryLine(trimmed), true
}

func prompt(envName string) string {
	return fmt.Sprintf("insyra [%s] > ", envName)
}
