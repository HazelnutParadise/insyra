package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// The types command must write its rendered output to ctx.Output (which the
// REPL and `run` capture), not os.Stdout. Before the writer-aware
// ShowTypesTo / ShowTypesRangeTo change, this output escaped to stdout and the
// captured buffer stayed empty — so this test would have failed.
func TestTypesOutputReachesCtxOutput(t *testing.T) {
	out := func(ctx *ExecContext) string {
		return ctx.Output.(*bytes.Buffer).String()
	}

	t.Run("types DataList", func(t *testing.T) {
		ctx := newTestExecContext(t)
		ctx.Vars["x"] = insyra.NewDataList(1, "two", 3.0)
		if err := runTypesCommand(ctx, []string{"x"}); err != nil {
			t.Fatalf("types: %v", err)
		}
		s := out(ctx)
		if !strings.Contains(s, "DataList Type Info") {
			t.Fatalf("types output did not reach ctx.Output; buffer=%q", s)
		}
		if !strings.Contains(s, "string") {
			t.Fatalf("expected element type names in output; buffer=%q", s)
		}
	})

	t.Run("types DataTable", func(t *testing.T) {
		ctx := newTestExecContext(t)
		ctx.Vars["t"] = insyra.NewDataTable(insyra.NewDataList(1, 2).SetName("n"))
		if err := runTypesCommand(ctx, []string{"t"}); err != nil {
			t.Fatalf("types: %v", err)
		}
		if !strings.Contains(out(ctx), "DataTable Type Info") {
			t.Fatalf("types output did not reach ctx.Output; buffer=%q", out(ctx))
		}
	})

	t.Run("unknown variable errors", func(t *testing.T) {
		ctx := newTestExecContext(t)
		if err := runTypesCommand(ctx, []string{"nope"}); err == nil {
			t.Fatal("expected error for unknown variable")
		}
	})
}
