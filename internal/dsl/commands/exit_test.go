package commands

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeScript(t *testing.T, dir, name string, lines ...string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// exit and quit were no-ops in a script, so every line after them ran (#328).
func TestExitEndsTheScriptAtThatLine(t *testing.T) {
	for _, word := range []string{"exit", "quit"} {
		t.Run(word, func(t *testing.T) {
			ctx := newTestExecContext(t)
			script := writeScript(t, t.TempDir(), "s.isr", "newdl 1 as a", word, "newdl 2 as b")
			if err := Dispatch(ctx, "run", []string{script}); err != nil {
				t.Fatalf("run: %v", err)
			}
			if _, ok := ctx.Vars["a"]; !ok {
				t.Error("the line before exit did not run")
			}
			if _, ok := ctx.Vars["b"]; ok {
				t.Error("the line after exit ran")
			}
			out := ctx.Output.(*bytes.Buffer).String()
			if !strings.Contains(out, "script ended by exit at line 2") {
				t.Errorf("output does not say where the script ended:\n%s", out)
			}
			if strings.Contains(out, "script complete") || strings.Contains(out, "error") {
				t.Errorf("output reports completion or an error:\n%s", out)
			}
		})
	}
}

func TestExitInANestedScriptStopsEveryScript(t *testing.T) {
	dir := t.TempDir()
	inner := writeScript(t, dir, "inner.isr", "newdl 1 as a", "exit", "newdl 2 as b")
	outer := writeScript(t, dir, "outer.isr", "run "+inner, "newdl 3 as c")
	ctx := newTestExecContext(t)
	if err := Dispatch(ctx, "run", []string{outer}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, ok := ctx.Vars["a"]; !ok {
		t.Error("the inner script's first line did not run")
	}
	for _, name := range []string{"b", "c"} {
		if _, ok := ctx.Vars[name]; ok {
			t.Errorf("%s was created after exit", name)
		}
	}
	out := ctx.Output.(*bytes.Buffer).String()
	if strings.Count(out, "script ended by exit") != 1 || strings.Contains(out, "script complete") || strings.Contains(out, "error") {
		t.Errorf("unexpected output:\n%s", out)
	}
	if ctx.scriptDepth != 0 {
		t.Errorf("script depth left at %d", ctx.scriptDepth)
	}
}

// A script run from the REPL ends at exit, and the REPL goes on.
func TestExitInAScriptRunFromTheREPLEndsOnlyTheScript(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.InREPL = true
	script := writeScript(t, t.TempDir(), "s.isr", "exit", "newdl 2 as b")
	if err := Dispatch(ctx, "run", []string{script}); err != nil {
		t.Fatalf("run returned %v, which would end the REPL", err)
	}
	if _, ok := ctx.Vars["b"]; ok {
		t.Error("the line after exit ran")
	}
}

func TestExitInTheREPLReturnsErrExit(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.InREPL = true
	for _, word := range []string{"exit", "quit"} {
		if err := Dispatch(ctx, word, nil); !errors.Is(err, ErrExit) || err.Error() != ErrExit.Error() {
			t.Errorf("%s in the REPL returned %v, want ErrExit", word, err)
		}
	}
}

// One-shot `insyra exit` succeeded although there was nothing to end.
func TestExitOutsideTheREPLOrAScriptSaysSo(t *testing.T) {
	for _, word := range []string{"exit", "quit"} {
		ctx := newTestExecContext(t)
		err := Dispatch(ctx, word, nil)
		if err == nil {
			t.Fatalf("%s outside the REPL or a script succeeded", word)
		}
		if !errors.Is(err, ErrExit) {
			t.Errorf("%s: error %v does not wrap ErrExit", word, err)
		}
		if !strings.Contains(err.Error(), "only ends the REPL or a script") {
			t.Errorf("%s: error %q does not say where exit applies", word, err)
		}
	}
}

func TestDispatchFindsACommandByItsAlias(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.InREPL = true
	if err := Dispatch(ctx, "quit", nil); !errors.Is(err, ErrExit) {
		t.Errorf("Dispatch(quit) = %v, want ErrExit", err)
	}
	if err := Dispatch(ctx, "no-such-command", nil); err == nil || !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Dispatch(no-such-command) = %v", err)
	}
}
