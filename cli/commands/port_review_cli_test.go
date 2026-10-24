package commands

import (
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// A negative mu reaches wilcoxon in one-shot mode instead of being read as a
// flag.
func TestOneShot_WilcoxonAcceptsANegativeMu(t *testing.T) {
	root, ctx, out := newOneShotRoot(t)
	ctx.Vars["x"] = insyra.NewDataList(-3.0, -2.5, -1.0, 0.5, -4.0, -2.0)
	root.SetArgs([]string{"wilcoxon", "single", "x", "-1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("wilcoxon single x -1 failed: %v", err)
	}
	if !strings.Contains(out.String(), "W=") {
		t.Errorf("output = %q, want a W statistic", out.String())
	}
}

// The same holds for the one-sample t and z tests, whose mu is the argument a
// leading minus sign used to be taken from.
func TestOneShot_TAndZTestsAcceptANegativeMu(t *testing.T) {
	for _, args := range [][]string{
		{"ttest", "single", "x", "-1"},
		{"ztest", "single", "x", "-1", "1"},
	} {
		root, ctx, out := newOneShotRoot(t)
		ctx.Vars["x"] = insyra.NewDataList(-3.0, -2.5, -1.0, 0.5, -4.0, -2.0)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v failed: %v", args, err)
		}
		if out.Len() == 0 {
			t.Errorf("%v printed nothing", args)
		}
	}
}

// A trailing argument the rank tests do not read is an error, not ignored.
func TestRankTestsRefuseExtraArguments(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{
		"a": insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0),
		"b": insyra.NewDataList(2.0, 3.0, 4.0, 5.0, 7.0),
	})
	cases := []struct {
		run  func(*ExecContext, []string) error
		args []string
	}{
		{runWilcoxonCommand, []string{"single", "a", "0", "two-sided", "less"}},
		{runWilcoxonCommand, []string{"paired", "a", "b", "two-sided", "less"}},
		{runMannWhitneyCommand, []string{"a", "b", "two-sided", "greater"}},
	}
	for _, c := range cases {
		if err := c.run(ctx, c.args); err == nil {
			t.Errorf("%v was accepted", c.args)
		}
	}
}

// The list form of anova twoway takes its level counts first; a table that
// happens to be named like a number does not take the call over.
func TestAnovaTwowayListFormSurvivesATableNamedLikeANumber(t *testing.T) {
	ctx := newTimeSeriesContext(t, map[string]any{
		"2":   insyra.NewDataTable(insyra.NewDataList(1.0)),
		"c11": insyra.NewDataList(1.0, 2.0, 3.0),
		"c12": insyra.NewDataList(2.0, 3.0, 4.0),
		"c21": insyra.NewDataList(5.0, 6.0, 7.0),
		"c22": insyra.NewDataList(6.0, 8.0, 9.0),
	})
	if err := runAnovaCommand(ctx, []string{"twoway", "2", "2", "c11", "c12", "c21", "c22"}); err != nil {
		t.Fatalf("anova twoway 2 2 ... failed: %v", err)
	}
}
