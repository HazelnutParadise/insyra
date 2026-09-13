package commands

import (
	"bytes"
	"log"
	"reflect"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// The sort command resolves a column argument to a name or a position and
// passes only that field. It used to set ColumnNumber: -1 next to every name,
// which SortBy now reads as a second selector and warns about.
func TestSortCommandPassesOnlyTheFieldItResolved(t *testing.T) {
	// A warning only reaches the log at Warning level or below, so pin it; a
	// quieter level would make the no-warning check pass vacuously.
	level := insyra.Config.GetLogLevel()
	insyra.Config.SetLogLevel(insyra.LogLevelWarning)
	t.Cleanup(func() { insyra.Config.SetLogLevel(level) })
	var logged bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(prev) })

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
		ctx := tableCtx(t)
		if err := Dispatch(ctx, c.argv[0], c.argv[1:]); err != nil {
			t.Errorf("%s: %v", c.label, err)
			continue
		}
		dt := ctx.Vars["dt"].(*insyra.DataTable)
		if got := dt.GetColByName(c.col).Data(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %s is %v, want %v", c.label, c.col, got, c.want)
		}
		if strings.Contains(logged.String(), "SortBy") {
			t.Errorf("%s: sorting logged a warning: %q", c.label, logged.String())
		}
	}
}
