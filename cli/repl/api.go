package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/HazelnutParadise/insyra/cli/commands"
	"github.com/HazelnutParadise/insyra/cli/env"
)

type DSLSession struct {
	ctx *commands.ExecContext
}

func NewDSLSession(envName string, output io.Writer) (*DSLSession, error) {
	if err := env.EnsureDefaultEnvironment(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(envName) == "" {
		envName = "default"
	}

	envPath, err := env.Open(envName)
	if err != nil {
		return nil, err
	}

	vars, err := env.RestoreVariables(envName)
	if err != nil {
		vars = map[string]any{}
	}

	if output == nil {
		output = io.Discard
	}

	return &DSLSession{
		ctx: &commands.ExecContext{
			EnvName: envName,
			EnvPath: envPath,
			Vars:    vars,
			Output:  output,
		},
	}, nil
}

func (session *DSLSession) Execute(line string) error {
	if session == nil || session.ctx == nil {
		return fmt.Errorf("dsl session is nil")
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}

	tokens := tokenize(trimmed)
	if len(tokens) == 0 {
		return nil
	}

	if err := commands.Dispatch(session.ctx, tokens[0], tokens[1:]); err != nil {
		return err
	}

	_ = env.AppendHistory(session.ctx.EnvName, trimmed)
	return env.SaveState(session.ctx.EnvName, session.ctx.Vars)
}

func (session *DSLSession) Context() *commands.ExecContext {
	if session == nil {
		return nil
	}
	return session.ctx
}

func (session *DSLSession) ExecuteFile(path string) error {
	if session == nil || session.ctx == nil {
		return fmt.Errorf("dsl session is nil")
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if err := session.Execute(line); err != nil {
			return fmt.Errorf("line %d: %w", lineNumber, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
