package dsl

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/dsl/commands"
	"github.com/HazelnutParadise/insyra/internal/dsl/env"
)

type Session struct {
	ctx *commands.ExecContext
}

// NewSession creates a DSL session on the environment envName of mgr,
// creating the environment when it does not exist.
//
// mgr must be non-nil. engine/dsl documents the choices callers have.
// envName "" defaults to "default". output nil silently discards.
func NewSession(mgr *env.Manager, envName string, output io.Writer) (*Session, error) {
	if mgr == nil {
		return nil, errors.New("dsl: env manager is required (pass dsl.DefaultManager() or dsl.NewManager(root, dir))")
	}

	if err := mgr.EnsureDefaultEnvironment(); err != nil {
		return nil, err
	}

	if strings.TrimSpace(envName) == "" {
		envName = "default"
	}

	// A missing environment is created. Create refuses one that already
	// exists, so when another session creates it in between, the second check
	// finds it and the session opens it instead of failing.
	if !mgr.Exists(envName) {
		if err := mgr.Create(envName); err != nil && !mgr.Exists(envName) {
			return nil, err
		}
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

	return &Session{
		ctx: &commands.ExecContext{
			EnvName: envName,
			EnvPath: envPath,
			Vars:    vars,
			Output:  output,
			Env:     mgr,
		},
	}, nil
}

func (session *Session) Execute(line string) error {
	if session == nil || session.ctx == nil {
		return fmt.Errorf("dsl session is nil")
	}

	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}

	tokens := Tokenize(trimmed)
	if len(tokens) == 0 {
		return nil
	}

	if err := commands.Dispatch(session.ctx, tokens[0], tokens[1:]); err != nil {
		return err
	}

	_ = session.ctx.Env.AppendHistory(session.ctx.EnvName, commands.SanitizeHistoryLine(trimmed))
	return commands.SaveEnvState(session.ctx)
}

func (session *Session) Context() *commands.ExecContext {
	if session == nil {
		return nil
	}
	return session.ctx
}

func (session *Session) ExecuteFile(path string) error {
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
