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
	t.Cleanup(func() { env.SetBasePath("") })
	return home
}

func TestNewDSLSessionRequiresManager(t *testing.T) {
	if _, err := NewDSLSession(nil, "default", nil); err == nil {
		t.Fatalf("expected error when manager is nil")
	}
}

func TestNewDSLSessionWithCustomManager(t *testing.T) {
	setupTempHome(t)

	workspace := filepath.Join(t.TempDir(), "workspace", ".idensyra")
	mgr := env.NewManager(workspace, "")

	session, err := NewDSLSession(mgr, "default", nil)
	if err != nil {
		t.Fatalf("NewDSLSession with custom manager failed: %v", err)
	}

	expected := filepath.Join(workspace, "envs", "default")
	if session.Context().EnvPath != expected {
		t.Fatalf("EnvPath = %q, want %q", session.Context().EnvPath, expected)
	}

	if err := session.Execute("newdl 1 2 3 as x"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	statePath := filepath.Join(expected, "state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state.json under workspace missing: %v", err)
	}
}

func TestNewDSLSessionWithCustomEnvsDirName(t *testing.T) {
	setupTempHome(t)

	workspace := filepath.Join(t.TempDir(), "workspace", ".idensyra")
	mgr := env.NewManager(workspace, "insights")

	session, err := NewDSLSession(mgr, "default", nil)
	if err != nil {
		t.Fatalf("NewDSLSession with custom envs dir failed: %v", err)
	}

	expected := filepath.Join(workspace, "insights", "default")
	if session.Context().EnvPath != expected {
		t.Fatalf("EnvPath = %q, want %q", session.Context().EnvPath, expected)
	}

	if err := session.Execute("newdl 1 2 3 as x"); err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(expected, "state.json")); err != nil {
		t.Fatalf("state.json under insights/ missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "envs")); !os.IsNotExist(err) {
		t.Fatalf("legacy envs/ should not be created when custom name is used")
	}
}

func TestTwoSessionsWithDifferentManagersDoNotInterfere(t *testing.T) {
	setupTempHome(t)

	wsA := filepath.Join(t.TempDir(), "wsA", ".idensyra")
	wsB := filepath.Join(t.TempDir(), "wsB", ".idensyra")

	mgrA := env.NewManager(wsA, "")
	mgrB := env.NewManager(wsB, "")

	sessionA, err := NewDSLSession(mgrA, "default", nil)
	if err != nil {
		t.Fatalf("create sessionA: %v", err)
	}
	sessionB, err := NewDSLSession(mgrB, "default", nil)
	if err != nil {
		t.Fatalf("create sessionB: %v", err)
	}

	// Interleave commands across sessions. Each session is bound to its own
	// Manager, so writes go strictly to the matching root.
	if err := sessionA.Execute("newdl 1 2 3 as a_var"); err != nil {
		t.Fatalf("sessionA execute: %v", err)
	}
	if err := sessionB.Execute("newdl 9 9 as b_var"); err != nil {
		t.Fatalf("sessionB execute: %v", err)
	}
	if err := sessionA.Execute("mean a_var"); err != nil {
		t.Fatalf("sessionA mean: %v", err)
	}

	pathA := filepath.Join(wsA, "envs", "default")
	pathB := filepath.Join(wsB, "envs", "default")

	stateAraw, err := os.ReadFile(filepath.Join(pathA, "state.json"))
	if err != nil {
		t.Fatalf("read wsA state: %v", err)
	}
	stateBraw, err := os.ReadFile(filepath.Join(pathB, "state.json"))
	if err != nil {
		t.Fatalf("read wsB state: %v", err)
	}

	if !strings.Contains(string(stateAraw), "a_var") || strings.Contains(string(stateAraw), "b_var") {
		t.Fatalf("wsA state should contain a_var only, got: %s", string(stateAraw))
	}
	if !strings.Contains(string(stateBraw), "b_var") || strings.Contains(string(stateBraw), "a_var") {
		t.Fatalf("wsB state should contain b_var only, got: %s", string(stateBraw))
	}

	historyA, err := os.ReadFile(filepath.Join(pathA, "history.txt"))
	if err != nil {
		t.Fatalf("read wsA history: %v", err)
	}
	historyB, err := os.ReadFile(filepath.Join(pathB, "history.txt"))
	if err != nil {
		t.Fatalf("read wsB history: %v", err)
	}
	if !strings.Contains(string(historyA), "a_var") || strings.Contains(string(historyA), "b_var") {
		t.Fatalf("wsA history should only contain a_var commands, got: %s", string(historyA))
	}
	if !strings.Contains(string(historyB), "b_var") || strings.Contains(string(historyB), "a_var") {
		t.Fatalf("wsB history should only contain b_var commands, got: %s", string(historyB))
	}
}

func TestNewDSLSessionWithDefaultManager(t *testing.T) {
	home := setupTempHome(t)

	session, err := NewDSLSession(env.Default(), "default", nil)
	if err != nil {
		t.Fatalf("NewDSLSession failed: %v", err)
	}

	expected := filepath.Join(home, ".insyra", "envs", "default")
	if session.Context().EnvPath != expected {
		t.Fatalf("EnvPath = %q, want %q", session.Context().EnvPath, expected)
	}
}

func TestNewDSLSessionDefaultEnvironment(t *testing.T) {
	home := setupTempHome(t)

	session, err := NewDSLSession(env.Default(), "", nil)
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

	session, err := NewDSLSession(env.Default(), "default", nil)
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

	session, err := NewDSLSession(env.Default(), "default", nil)
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

	session, err := NewDSLSession(env.Default(), "default", nil)
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

	session, err := NewDSLSession(env.Default(), "default", nil)
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
