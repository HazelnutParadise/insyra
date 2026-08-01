package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// M20: the show / summary / describe commands must write their rendered output
// to ctx.Output (which the REPL and `run` capture), not os.Stdout. Before the
// writer-aware ShowRangeTo / ShowTo / SummaryTo change, this output escaped to
// stdout and the captured buffer stayed empty — so this test would have failed.
func TestM20_DisplayOutputReachesCtxOutput(t *testing.T) {
	out := func(ctx *ExecContext) string {
		return ctx.Output.(*bytes.Buffer).String()
	}

	t.Run("summary DataList", func(t *testing.T) {
		ctx := newTestExecContext(t)
		ctx.Vars["x"] = insyra.NewDataList(1, 2, 3, 4, 5)
		if err := runSummaryCommand(ctx, []string{"x"}); err != nil {
			t.Fatalf("summary: %v", err)
		}
		if !strings.Contains(out(ctx), "Statistical Summary") {
			t.Fatalf("summary output did not reach ctx.Output; buffer=%q", out(ctx))
		}
	})

	t.Run("show DataList", func(t *testing.T) {
		ctx := newTestExecContext(t)
		ctx.Vars["x"] = insyra.NewDataList(10, 20, 30)
		if err := runShowCommand(ctx, []string{"x"}); err != nil {
			t.Fatalf("show: %v", err)
		}
		if strings.TrimSpace(out(ctx)) == "" {
			t.Fatal("show output did not reach ctx.Output; buffer empty")
		}
	})

	t.Run("describe DataTable rendered table", func(t *testing.T) {
		ctx := newTestExecContext(t)
		ctx.Vars["x"] = insyra.NewDataList(1, 2, 3, 4, 5)
		if err := runDescribeCommand(ctx, []string{"x"}); err != nil {
			t.Fatalf("describe: %v", err)
		}
		s := out(ctx)
		if !strings.Contains(s, "saved description as") {
			t.Fatalf("missing save message; buffer=%q", s)
		}
		// The rendered describe table (result.Show() -> ShowTo(ctx.Output)) must be
		// present too, not just the one-line save message.
		if len(strings.TrimSpace(s)) <= len("saved description as $result") {
			t.Fatalf("describe rendered table did not reach ctx.Output; buffer=%q", s)
		}
	})
}
