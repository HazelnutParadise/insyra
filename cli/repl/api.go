package repl

import (
	"bufio"
	"errors"
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

// NewDSLSession creates a DSL session bound to mgr's environment storage.
//
// mgr must be non-nil — pass env.Default() to use the standard
// <UserHomeDir>/.insyra root, or env.NewManager(path) for a custom one.
// envName "" defaults to "default". output nil silently discards.
func NewDSLSession(mgr *env.Manager, envName string, output io.Writer) (*DSLSession, error) {
	if mgr == nil {
		return nil, errors.New("dsl: env manager is required (pass env.Default() or env.NewManager(path))")
	}

	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(envName) == "" {
		envName = "default"
	}

	envPath, err := mgr.Open(envName)
	if err != nil {
		return nil, err
	}

	vars, err := mgr.RestoreVariables(envName)
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
			Env:     mgr,
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

	_ = session.ctx.Env.AppendHistory(session.ctx.EnvName, trimmed)
	return session.ctx.Env.SaveState(session.ctx.EnvName, session.ctx.Vars)
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
	defer func() {
		_ = file.Close()
	}()

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
