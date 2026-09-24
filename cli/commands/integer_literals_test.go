package commands

import (
	"bytes"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// A CSV load stores integers as int64, and in one-shot mode every variable is
// restored as int64, while the CLI reads a typed 2 as a Go int. The library
// compared the two with ==, so on data that plainly held 2, count printed 0,
// find printed [] and replace changed nothing.
func TestIntegerLiteralsMatchInt64Data(t *testing.T) {
	run := func(args ...string) (*ExecContext, string) {
		t.Helper()
		ctx := newTestExecContext(t)
		ctx.Vars["t"] = insyra.NewDataTable(insyra.NewDataList(int64(1), int64(2), int64(3), int64(2)).SetName("v"))
		if err := Dispatch(ctx, args[0], args[1:]); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		return ctx, strings.TrimSpace(ctx.Output.(*bytes.Buffer).String())
	}

	if _, out := run("count", "t", "2"); out != "2" {
		t.Errorf("count t 2 printed %q, want 2", out)
	}
	if _, out := run("find", "t", "2"); out != "[1 3]" {
		t.Errorf("find t 2 printed %q, want [1 3]", out)
	}
	ctx, _ := run("replace", "t", "2", "0")
	if n := ctx.Vars["t"].(*insyra.DataTable).Count(int64(2)); n != 0 {
		t.Errorf("replace t 2 0 left %d cells holding 2", n)
	}
}
