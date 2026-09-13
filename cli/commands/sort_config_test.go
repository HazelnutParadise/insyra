package commands

import (
	"bytes"
	"log"
	"reflect"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// sortTableCtx returns a context holding a three-column table `dt`.
func sortTableCtx(t *testing.T) *ExecContext {
	t.Helper()
	ctx := newTestExecContext(t)
	ctx.Vars["dt"] = insyra.NewDataTable(
		insyra.NewDataList(1, 2, 3).SetName("price"),
		insyra.NewDataList(4, 5, 6).SetName("qty"),
		insyra.NewDataList("a", "b", "c").SetName("tag"),
	)
	return ctx
}

// captureSortWarnings pins the log level at Warning, so a warning reaches the
// log and a no-warning check cannot pass vacuously, and returns the log output.
func captureSortWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	insyra.Config.SetLogLevel(insyra.LogLevelWarning)
	t.Cleanup(func() { insyra.Config.SetLogLevel(level) })
	var logged bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &logged
}

// The sort command resolves a column argument to a name or a position and
// passes only that field. It used to set ColumnNumber: -1 next to every name,
// which SortBy now reads as a second selector and warns about.
func TestSortCommandPassesOnlyTheFieldItResolved(t *testing.T) {
	logged := captureSortWarnings(t)

	for _, c := range []struct {
		label string
		argv  []string
		col   string
		want  []any
	}{
		{"the first column by position", []string{"sort", "dt", "0", "desc"}, "price", []any{3, 2, 1}},
		{"a later column by position", []string{"sort", "dt", "1", "desc"}, "qty", []any{6, 5, 4}},
		{"a column by name", []string{"sort", "dt", "tag", "desc"}, "tag", []any{"c", "b", "a"}},
	} {
		logged.Reset()
		ctx := sortTableCtx(t)
		if err := Dispatch(ctx, c.argv[0], c.argv[1:]); err != nil {
			t.Errorf("%s: %v", c.label, err)
			continue
		}
		dt := ctx.Vars["dt"].(*insyra.DataTable)
		if got := dt.GetColByName(c.col).Data(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %s is %v, want %v", c.label, c.col, got, c.want)
		}
		if out := ctx.Output.(*bytes.Buffer).String(); out != "sorted\n" {
			t.Errorf("%s: printed %q, want %q", c.label, out, "sorted\n")
		}
		if strings.Contains(logged.String(), "SortBy") {
			t.Errorf("%s: sorting logged a warning: %q", c.label, logged.String())
		}
	}
}

// A column that is not there is still reported, now by SortBy, and the table
// does not move. The command's own output and exit status are unchanged.
func TestSortCommandReportsAMissingColumn(t *testing.T) {
	logged := captureSortWarnings(t)

	for _, selector := range []string{"nonexistent", "99", "-1"} {
		logged.Reset()
		ctx := sortTableCtx(t)
		if err := Dispatch(ctx, "sort", []string{"dt", selector}); err != nil {
			t.Errorf("sort dt %s: %v", selector, err)
			continue
		}
		dt := ctx.Vars["dt"].(*insyra.DataTable)
		if got := dt.GetColByName("price").Data(); !reflect.DeepEqual(got, []any{1, 2, 3}) {
			t.Errorf("sort dt %s: the table moved: %v", selector, got)
		}
		if !strings.Contains(logged.String(), "SortBy") {
			t.Errorf("sort dt %s: nothing reported the missing column: %q", selector, logged.String())
		}
	}
}
