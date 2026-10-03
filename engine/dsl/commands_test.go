package dsl

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// A program adds its own command through engine/dsl alone, and every session
// runs it like a built-in one: it reads and writes the session's variables,
// prints to the session's output, and its result is saved.
func TestRegisterACommandAndRunItInASession(t *testing.T) {
	err := Register(&CommandHandler{
		Name:        "zzdouble",
		Usage:       "zzdouble <var> [as <var>]",
		Description: "Double every number in a list (test command)",
		Args:        MaxArgs(1).WithAlias(),
		Run: func(ctx *ExecContext, args []string) error {
			name := args[0]
			if _, ok := ctx.Vars[name]; !ok {
				return fmt.Errorf("variable not found: %s", name)
			}
			target := "doubled"
			if len(args) == 3 && args[1] == "as" {
				target = args[2]
			}
			ctx.Vars[target] = "doubled " + name
			_, _ = fmt.Fprintf(ctx.Output, "saved as %s\n", target)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	mgr := NewManager(t.TempDir(), "")
	var out bytes.Buffer
	session, err := NewSession(mgr, "default", &out)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Execute("newdl 1 2 3 as x"); err != nil {
		t.Fatal(err)
	}
	if err := session.Execute("zzdouble x as y"); err != nil {
		t.Fatalf("running the registered command: %v", err)
	}
	if !strings.Contains(out.String(), "saved as y") {
		t.Errorf("output %q does not hold what the command printed", out.String())
	}
	vars, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatal(err)
	}
	if vars["y"] != "doubled x" {
		t.Errorf("the command's result was not saved: y = %v", vars["y"])
	}

	// The declared argument count is enforced before Run, as for a built-in.
	if err := session.Execute("zzdouble x y z"); err == nil {
		t.Error("an argument past the declared count was accepted")
	}
}

func TestRegisterRefusesATakenName(t *testing.T) {
	run := func(ctx *ExecContext, args []string) error { return nil }
	if err := Register(&CommandHandler{Name: "newdl", Args: OpenArgs(), Run: run}); err == nil {
		t.Error("Register replaced the built-in newdl")
	}
	if err := Register(&CommandHandler{Name: "zzonce", Args: OpenArgs(), Run: run}); err != nil {
		t.Fatal(err)
	}
	if err := Register(&CommandHandler{Name: "zzonce", Args: OpenArgs(), Run: run}); err == nil {
		t.Error("Register accepted the same name twice")
	}
	if err := Register(&CommandHandler{Name: "zznorun", Args: OpenArgs()}); err == nil {
		t.Error("Register accepted a command without Run")
	}
}

// A command's error reaches the caller of Execute, and nothing is saved.
func TestARegisteredCommandsErrorReachesTheCaller(t *testing.T) {
	sentinel := errors.New("zzfail failed")
	if err := Register(&CommandHandler{
		Name: "zzfail",
		Args: FormArgs(map[string]int{"now": 1}),
		Run:  func(ctx *ExecContext, args []string) error { return sentinel },
	}); err != nil {
		t.Fatal(err)
	}
	session, err := NewSession(NewManager(t.TempDir(), ""), "default", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Execute("zzfail now"); !errors.Is(err, sentinel) {
		t.Errorf("Execute returned %v, want the command's error", err)
	}
	_ = FormArgsAt(1, map[string]int{"x": 2})
	var _ CommandFlag
	var _ ArgLimit
}

// A program reading lines from its user ends its loop on exit by testing the
// error against ErrExit, with nothing imported but engine/dsl.
func TestExecuteExitWrapsErrExit(t *testing.T) {
	session, err := NewSession(NewManager(t.TempDir(), ""), "default", nil)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	for _, word := range []string{"exit", "quit"} {
		if err := session.Execute(word); !errors.Is(err, ErrExit) {
			t.Errorf("Execute(%q) = %v, want an error wrapping ErrExit", word, err)
		}
	}
	if err := session.Execute("newdl 1 as x"); errors.Is(err, ErrExit) || err != nil {
		t.Errorf("Execute(newdl) = %v", err)
	}
	if !strings.Contains(fmt.Sprint(ErrExit), "exit") {
		t.Errorf("ErrExit reads %q", ErrExit)
	}
}
