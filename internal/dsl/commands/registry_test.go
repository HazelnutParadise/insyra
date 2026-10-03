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
