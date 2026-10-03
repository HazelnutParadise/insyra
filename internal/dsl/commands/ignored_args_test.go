package commands

import (
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// An argument a command does not use must be an error, not dropped: dropped,
// a typo or a misremembered option looks like it worked. `mean x as m` printed
// the mean and stored nothing, and `accel plan --precision float32` ran as if
// the flag meant something.
func TestExtraArgumentsAreRejected(t *testing.T) {
	for _, name := range []string{"sum", "mean", "median", "mode", "stdev", "var", "min", "max", "range"} {
		ctx := newTestExecContext(t)
		ctx.Vars["x"] = insyra.NewDataList(1.0, 2.0, 2.0, 4.0)
		if err := Dispatch(ctx, name, []string{"x"}); err != nil {
			t.Errorf("%s x: %v", name, err)
		}
		err := Dispatch(ctx, name, []string{"x", "as", "m"})
		if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), `"as"`) {
			t.Errorf("%s x as m: got %v, want an error naming %s and the argument", name, err, name)
		}
		if _, stored := ctx.Vars["m"]; stored {
			t.Errorf("%s x as m stored m", name)
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
