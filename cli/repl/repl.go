package repl

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/HazelnutParadise/insyra/cli/commands"
	"github.com/HazelnutParadise/insyra/cli/env"
	"github.com/ergochat/readline"
)

func Start(ctx *commands.ExecContext) error {
	if ctx == nil {
		ctx = &commands.ExecContext{}
	}
	ctx.InREPL = true
	defer func() {
		ctx.InREPL = false
	}()
	if ctx.EnvName == "" {
		ctx.EnvName = "default"
	}
	if ctx.EnvPath == "" {
		envPath, err := env.Open(ctx.EnvName)
		if err != nil {
			return err
		}
		ctx.EnvPath = envPath
	}
	if ctx.Vars == nil {
		vars, err := env.RestoreVariables(ctx.EnvName)
		if err != nil {
			ctx.Vars = map[string]any{}
		} else {
			ctx.Vars = vars
		}
	}

	historyFile := filepath.Join(ctx.EnvPath, "history.txt")
	instance, err := readline.NewFromConfig(&readline.Config{
		Prompt:          prompt(ctx.EnvName),
		HistoryFile:     historyFile,
		AutoComplete:    NewAutoCompleter(ctx),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return err
	}
	defer instance.Close()
	defer func() {
		_ = env.SaveState(ctx.EnvName, ctx.Vars)
	}()

	for {
		line, err := instance.ReadLine()
		if errors.Is(err, readline.ErrInterrupt) {
			continue
		}
		if err != nil {
			return nil
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if trimmed == "exit" || trimmed == "quit" {
			return nil
		}
		tokens := tokenize(trimmed)
		if len(tokens) == 0 {
			continue
		}

		if err := commands.Dispatch(ctx, tokens[0], tokens[1:]); err != nil {
			_, _ = fmt.Fprintf(instance.Stderr(), "error: %v\n", err)
		}

		_ = env.AppendHistory(ctx.EnvName, trimmed)
		_ = env.SaveState(ctx.EnvName, ctx.Vars)

		if ctx.EnvName != "" {
			instance.SetPrompt(prompt(ctx.EnvName))
		}
	}
}

func prompt(envName string) string {
	return fmt.Sprintf("insyra [%s] > ", envName)
}

func tokenize(input string) []string {
	result := []string{}
	var builder strings.Builder
	quote := rune(0)
	escaped := false

	flush := func() {
		if builder.Len() == 0 {
			return
		}
		result = append(result, builder.String())
		builder.Reset()
	}

	for _, ch := range input {
		if escaped {
			builder.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
				continue
			}
			builder.WriteRune(ch)
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == ' ' || ch == '\t' {
			flush()
			continue
		}
		builder.WriteRune(ch)
	}
	flush()
	return result
}
