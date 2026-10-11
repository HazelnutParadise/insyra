package commands

import (
	"bytes"
	"testing"
)

func TestRegisterAndDispatch(t *testing.T) {
	oldRegistry := Registry
	Registry = map[string]*CommandHandler{}
	defer func() { Registry = oldRegistry }()

	called := false
	err := Register(&CommandHandler{
		Name: "hello",
		Run: func(ctx *ExecContext, args []string) error {
			called = true
			ctx.Vars["ok"] = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	ctx := &ExecContext{Vars: map[string]any{}, Output: &bytes.Buffer{}}
	if err := Dispatch(ctx, "hello", nil); err != nil {
		t.Fatalf("dispatch failed: %v", err)
	}
	if !called {
		t.Fatalf("handler not called")
	}
	if ok, exists := ctx.Vars["ok"]; !exists || ok != true {
		t.Fatalf("context not updated by handler")
	}
}

func TestRegisterRejectsDuplicate(t *testing.T) {
	oldRegistry := Registry
	Registry = map[string]*CommandHandler{}
	defer func() { Registry = oldRegistry }()

	firstErr := Register(&CommandHandler{Name: "dup", Run: func(ctx *ExecContext, args []string) error { return nil }})
	if firstErr != nil {
		t.Fatalf("first register failed: %v", firstErr)
	}
	secondErr := Register(&CommandHandler{Name: "dup", Run: func(ctx *ExecContext, args []string) error { return nil }})
	if secondErr == nil {
		t.Fatalf("expected duplicate register to fail")
	}
}

// A registration refused as a duplicate changes nothing: retrying the same
// handler with a new argument limit does not wrap the registered command in
// that limit.
func TestDuplicateRegistrationDoesNotMutateOriginalHandler(t *testing.T) {
	oldRegistry := Registry
	Registry = map[string]*CommandHandler{}
	defer func() { Registry = oldRegistry }()
	handler := &CommandHandler{Name: "dup", Run: func(*ExecContext, []string) error { return nil }}
	if err := Register(handler); err != nil {
		t.Fatal(err)
	}
	handler.Args = MaxArgs(0)
	if err := Register(handler); err == nil {
		t.Fatal("expected duplicate error")
	}
	if err := Registry["dup"].Run(&ExecContext{}, []string{"a"}); err != nil {
		t.Fatalf("the registered command now refuses an argument: %v", err)
	}
	if err := handler.Run(&ExecContext{}, []string{"a"}); err != nil {
		t.Fatalf("the caller's handler was wrapped in the refused limit: %v", err)
	}
}

func TestRegisterRejectsNegativeArgumentLimits(t *testing.T) {
	for _, limit := range []ArgLimit{MaxArgs(-1), FormArgsAt(-1, map[string]int{"x": 1}), FormArgs(map[string]int{"x": -1})} {
		if err := Register(&CommandHandler{Name: "invalid", Args: limit, Run: func(*ExecContext, []string) error { return nil }}); err == nil {
			t.Errorf("Register(%+v) succeeded", limit)
		}
	}
}

func TestBuildCobraCommands(t *testing.T) {
	oldRegistry := Registry
	Registry = map[string]*CommandHandler{}
	defer func() { Registry = oldRegistry }()

	if err := Register(&CommandHandler{Name: "show", Usage: "show <var>", DisableFlagParsing: true, Run: func(ctx *ExecContext, args []string) error { return nil }}); err != nil {
		t.Fatalf("register show failed: %v", err)
	}
	if err := Register(&CommandHandler{Name: "help", Usage: "help", Run: func(ctx *ExecContext, args []string) error { return nil }}); err != nil {
		t.Fatalf("register help failed: %v", err)
	}

	commands := BuildCobraCommands(&ExecContext{Vars: map[string]any{}})
	if len(commands) != 2 {
		t.Fatalf("expected 2 cobra commands, got %d", len(commands))
	}

	if commands[0].Name() != "help" || commands[1].Name() != "show" {
		t.Fatalf("commands should be sorted by name, got %s then %s", commands[0].Name(), commands[1].Name())
	}
	if !commands[1].DisableFlagParsing {
		t.Fatalf("show command should keep DisableFlagParsing=true")
	}
}
