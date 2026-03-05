package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/cli/env"
)

func setupTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	return home
}

func TestNewDSLSessionDefaultEnvironment(t *testing.T) {
	home := setupTempHome(t)

	session, err := NewDSLSession("", nil)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if session == nil || session.Context() == nil {
		t.Fatalf("session/context should not be nil")
	}
	if session.Context().EnvName != "default" {
		t.Fatalf("unexpected env name: %s", session.Context().EnvName)
	}

	defaultPath := filepath.Join(home, ".insyra", "envs", "default")
	if _, err := os.Stat(defaultPath); err != nil {
		t.Fatalf("default env path missing: %v", err)
	}
}

func TestDSLSessionExecutePersistsStateAndHistory(t *testing.T) {
	setupTempHome(t)

	session, err := NewDSLSession("default", nil)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := session.Execute("newdl 1 2 3 as x"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if _, ok := session.Context().Vars["x"]; !ok {
		t.Fatalf("expected variable x in session vars")
	}

	state, err := env.LoadState("default")
	if err != nil {
		t.Fatalf("failed to load saved state: %v", err)
	}
	if _, ok := state.Variables["x"]; !ok {
		t.Fatalf("expected variable x in persisted state")
	}

	history, err := env.ReadHistory("default")
	if err != nil {
		t.Fatalf("failed to read history: %v", err)
	}
	if len(history) == 0 || history[len(history)-1] != "newdl 1 2 3 as x" {
		t.Fatalf("unexpected history contents: %v", history)
	}
}

func TestDSLSessionExecuteCommentAndUnknownCommand(t *testing.T) {
	setupTempHome(t)

	session, err := NewDSLSession("default", nil)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if err := session.Execute("# comment"); err != nil {
		t.Fatalf("comment line should not fail: %v", err)
	}

	if err := session.Execute("unknowncmd"); err == nil {
		t.Fatalf("expected error for unknown command")
	}
}

func TestDSLSessionExecuteFile(t *testing.T) {
	setupTempHome(t)

	session, err := NewDSLSession("default", nil)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	scriptPath := filepath.Join(t.TempDir(), "pipeline.isr")
	content := "# build sample\nnewdl 2 4 6 as x\nmean x\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create script file: %v", err)
	}

	if err := session.ExecuteFile(scriptPath); err != nil {
		t.Fatalf("execute file failed: %v", err)
	}

	if _, ok := session.Context().Vars["x"]; !ok {
		t.Fatalf("expected variable x after execute file")
	}
}

func TestDSLSessionExecuteFileIncludesLineNumber(t *testing.T) {
	setupTempHome(t)

	session, err := NewDSLSession("default", nil)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	scriptPath := filepath.Join(t.TempDir(), "bad.isr")
	content := "newdl 1 as x\nunknowncmd\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create bad script file: %v", err)
	}

	err = session.ExecuteFile(scriptPath)
	if err == nil {
		t.Fatalf("expected error for bad script command")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("expected line number in error, got: %v", err)
	}
}
