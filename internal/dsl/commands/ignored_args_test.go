package commands

import (
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// An argument a command does not use must be an error, not dropped: dropped,
// a typo or a misremembered option looks like it worked. `mean x as m` printed
// the mean and stored nothing (it now stores it, #370), and `accel plan --precision float32` ran as if
// the flag meant something.
func TestExtraArgumentsAreRejected(t *testing.T) {
	for _, name := range []string{"sum", "mean", "median", "mode", "stdev", "var", "min", "max", "range"} {
		ctx := newTestExecContext(t)
		ctx.Vars["x"] = insyra.NewDataList(1.0, 2.0, 2.0, 4.0)
		if err := Dispatch(ctx, name, []string{"x"}); err != nil {
			t.Errorf("%s x: %v", name, err)
		}
		// `as` keeps the number, so later commands can take it by name (#370).
		if err := Dispatch(ctx, name, []string{"x", "as", "m"}); err != nil {
			t.Errorf("%s x as m: %v", name, err)
		}
		if _, stored := ctx.Vars["m"]; !stored {
			t.Errorf("%s x as m did not store m", name)
		}
		if err := Dispatch(ctx, name, []string{"x", "y"}); err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), `"y"`) {
			t.Errorf("%s x y: got %v, want an error naming %s and the argument", name, err, name)
		}
	}

	for _, args := range [][]string{
		{"devices", "junk"},
		{"plan", "--precision", "float32"},
	} {
		ctx := newTestExecContext(t)
		if err := runAccelCommand(ctx, args); err == nil {
			t.Errorf("accel %v was accepted", args)
		}
	}
}
