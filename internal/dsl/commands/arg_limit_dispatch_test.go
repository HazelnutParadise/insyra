package commands

import (
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// Through Dispatch, the path every real invocation takes, an argument a
// command would have ignored is refused and named; a valid call still runs.
func TestDispatchRefusesIgnoredArguments(t *testing.T) {
	ctx := newTestExecContext(t)
	ctx.Vars["x"] = insyra.NewDataList(1.0, 2.0, 3.0, 4.0)
	ctx.Vars["t"] = insyra.NewDataTable(insyra.NewDataList(1.0, 2.0).SetName("a"), insyra.NewDataList(3.0, 4.0).SetName("b"))

	refused := []struct {
		call []string
		bad  string
	}{
		{[]string{"iqr", "x", "junk"}, "junk"},
		{[]string{"capitalize", "x", "junk", "as", "y"}, "junk"},
		{[]string{"ttest", "single", "x", "0", "junk"}, "junk"},
		{[]string{"clean", "x", "nan", "3"}, "3"},
		{[]string{"merge", "t", "t", "vertical", "inner", "on", "a"}, "on"},
		{[]string{"env", "list", "junk"}, "junk"},
		{[]string{"version", "junk"}, "junk"},
		{[]string{"find", "t", "1", "as", "found"}, "as"},
	}
	for _, tc := range refused {
		err := Dispatch(ctx, tc.call[0], tc.call[1:])
		if err == nil || !strings.Contains(err.Error(), `"`+tc.bad+`"`) {
			t.Errorf("%v: got %v, want an error naming %q", tc.call, err, tc.bad)
		}
	}

	accepted := [][]string{
		{"iqr", "x"},
		{"capitalize", "x", "as", "y"},
		{"ttest", "single", "x", "0"},
		{"sort", "t", "a", "desc"},
		{"version"},
	}
	for _, call := range accepted {
		if err := Dispatch(ctx, call[0], call[1:]); err != nil {
			t.Errorf("%v: refused with %v", call, err)
		}
	}
}
