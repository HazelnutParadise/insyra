package commands

import (
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

// `ttest two` without the variance token runs Welch's test, as R's t.test
// does, so a caller who says nothing gets the unpooled standard error rather
// than Student's.
func TestTTestTwoDefaultsToWelch(t *testing.T) {
	run := func(t *testing.T, args ...string) string {
		t.Helper()
		ctx := newTestExecContext(t)
		ctx.Vars["a"] = insyra.NewDataList(55.1, 49.3, 58.2, 61.9, 47.3, 51.0, 53.8, 59.7)
		ctx.Vars["b"] = insyra.NewDataList(46.9, 41.2, 45.7, 49.8, 44.0, 47.6, 46.5, 43.9, 50.2)
		if err := Dispatch(ctx, "ttest", args); err != nil {
			t.Fatalf("ttest %v: %v", args, err)
		}
		return outputOf(ctx)
	}

	def := run(t, "two", "a", "b")
	if def == "" {
		t.Fatal("ttest two a b printed nothing")
	}
	if unequal := run(t, "two", "a", "b", "unequal"); def != unequal {
		t.Errorf("default output %q, want the unequal output %q", def, unequal)
	}
	if equal := run(t, "two", "a", "b", "equal"); def == equal {
		t.Errorf("default output %q equals the equal-variance output; want Welch's test by default", def)
	}
}
