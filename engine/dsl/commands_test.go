package dsl

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRegisterACommandAndRunItInASession(t *testing.T) {
	err := Register(&CommandHandler{
		Name: "zzdouble", Usage: "zzdouble <var> [as <var>]", Args: MaxArgs(1).WithAlias(),
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
		t.Fatalf("running registered command: %v", err)
	}
	if !strings.Contains(out.String(), "saved as y") {
		t.Errorf("output %q does not contain command output", out.String())
	}
	vars, err := mgr.RestoreVariables("default")
	if err != nil {
		t.Fatal(err)
	}
	if vars["y"] != "doubled x" {
		t.Errorf("command result was not saved: y = %v", vars["y"])
	}
	if err := session.Execute("zzdouble x y z"); err == nil {
		t.Error("argument beyond the declared count was accepted")
	}
}

func TestRegisterRefusesTakenNamesAndInvalidHandlers(t *testing.T) {
	run := func(*ExecContext, []string) error { return nil }
	if err := Register(&CommandHandler{Name: "newdl", Args: OpenArgs(), Run: run}); err == nil {
		t.Error("Register replaced built-in newdl")
	}
	if err := Register(&CommandHandler{Name: "zzonce", Args: OpenArgs(), Run: run}); err != nil {
		t.Fatal(err)
	}
	if err := Register(&CommandHandler{Name: "zzonce", Args: OpenArgs(), Run: run}); err == nil {
		t.Error("Register accepted a duplicate name")
	}
	if err := Register(&CommandHandler{Name: "zznorun", Args: OpenArgs()}); err == nil {
		t.Error("Register accepted a missing Run")
	}
}

func TestRegisteredCommandErrorsReachSessionCaller(t *testing.T) {
	sentinel := errors.New("zzfail failed")
	if err := Register(&CommandHandler{Name: "zzfail", Args: FormArgs(map[string]int{"now": 1}), Run: func(*ExecContext, []string) error { return sentinel }}); err != nil {
		t.Fatal(err)
	}
	session, err := NewSession(NewManager(t.TempDir(), ""), "default", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Execute("zzfail now"); !errors.Is(err, sentinel) {
		t.Errorf("Execute returned %v, want command error", err)
	}
	_ = FormArgsAt(1, map[string]int{"x": 2})
	var _ CommandFlag
	var _ ArgLimit
}
