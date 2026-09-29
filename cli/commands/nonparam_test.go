package commands

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func setupNonparamVars(ctx *ExecContext) {
	ctx.Vars["before"] = insyra.NewDataList(3.0, 4, 2, 5, 3, 4, 2, 3)
	ctx.Vars["after"] = insyra.NewDataList(4.0, 5, 4, 5, 4, 5, 3, 4)
	ctx.Vars["a"] = insyra.NewDataList(15.0, 18, 22, 11, 30, 14, 26, 25)
	ctx.Vars["b"] = insyra.NewDataList(10.0, 9, 13, 17, 7, 12, 19, 8, 20)
	ctx.Vars["g1"] = insyra.NewDataList(1.0, 2, 3, 4)
	ctx.Vars["g2"] = insyra.NewDataList(3.0, 5, 6, 7)
	ctx.Vars["g3"] = insyra.NewDataList(8.0, 9, 10, 12)
}

func runWilcoxon(ctx *ExecContext, args ...string) (string, error) {
	err := Dispatch(ctx, "wilcoxon", args)
	return outputOf(ctx), err
}

func runMannWhitney(ctx *ExecContext, args ...string) (string, error) {
	err := Dispatch(ctx, "mannwhitney", args)
	return outputOf(ctx), err
}

func runKruskal(ctx *ExecContext, args ...string) (string, error) {
	err := Dispatch(ctx, "kruskal", args)
	return outputOf(ctx), err
}

func libraryWilcoxonSingle(dl *insyra.DataList, mu float64, alt stats.AlternativeHypothesis) string {
	r, _ := stats.SingleSampleWilcoxon(dl, mu, stats.WilcoxonOptions{Alternative: alt})
	return fmt.Sprintf("W=%v p=%v\n", r.Statistic, r.PValue)
}

func libraryWilcoxonPaired(dl1, dl2 *insyra.DataList, alt stats.AlternativeHypothesis) string {
	r, _ := stats.PairedWilcoxon(dl1, dl2, stats.WilcoxonOptions{Alternative: alt})
	return fmt.Sprintf("W=%v p=%v\n", r.Statistic, r.PValue)
}

func libraryMannWhitney(a, b *insyra.DataList, alt stats.AlternativeHypothesis) string {
	r, _ := stats.MannWhitneyU(a, b, stats.MannWhitneyUOptions{Alternative: alt})
	return fmt.Sprintf("U=%v p=%v\n", r.Statistic, r.PValue)
}

func libraryKruskal(groups []insyra.IDataList) string {
	r, _ := stats.KruskalWallis(groups)
	df := math.NaN()
	if r.DF != nil {
		df = *r.DF
	}
	return fmt.Sprintf("H=%v df=%v p=%v\n", r.Statistic, df, r.PValue)
}

func TestWilcoxonSingleMatchesLibrary(t *testing.T) {
	ctx := newTestExecContext(t)
	setupNonparamVars(ctx)

	// wilcoxon single before 3 (two-sided default)
	out, err := runWilcoxon(ctx, "single", "before", "3")
	if err != nil {
		t.Fatalf("wilcoxon single before 3 failed: %v", err)
	}
	expected := libraryWilcoxonSingle(ctx.Vars["before"].(*insyra.DataList), 3, stats.TwoSided)
	if out != expected {
		t.Errorf("output %q != expected %q", out, expected)
	}

	// wilcoxon single before 3 greater
	ctx2 := newTestExecContext(t)
	setupNonparamVars(ctx2)
	out2, err := runWilcoxon(ctx2, "single", "before", "3", "greater")
	if err != nil {
		t.Fatalf("wilcoxon single before 3 greater failed: %v", err)
	}
	expected2 := libraryWilcoxonSingle(ctx2.Vars["before"].(*insyra.DataList), 3, stats.Greater)
	if out2 != expected2 {
		t.Errorf("output %q != expected %q", out2, expected2)
	}
}

func TestWilcoxonPairedMatchesLibrary(t *testing.T) {
	ctx := newTestExecContext(t)
	setupNonparamVars(ctx)

	// wilcoxon paired before after less
	out, err := runWilcoxon(ctx, "paired", "before", "after", "less")
	if err != nil {
		t.Fatalf("wilcoxon paired before after less failed: %v", err)
	}
	expected := libraryWilcoxonPaired(ctx.Vars["before"].(*insyra.DataList), ctx.Vars["after"].(*insyra.DataList), stats.Less)
	if out != expected {
		t.Errorf("output %q != expected %q", out, expected)
	}

	// wilcoxon paired before after (two-sided default)
	ctx2 := newTestExecContext(t)
	setupNonparamVars(ctx2)
	out2, err := runWilcoxon(ctx2, "paired", "before", "after")
	if err != nil {
		t.Fatalf("wilcoxon paired before after failed: %v", err)
	}
	expected2 := libraryWilcoxonPaired(ctx2.Vars["before"].(*insyra.DataList), ctx2.Vars["after"].(*insyra.DataList), stats.TwoSided)
	if out2 != expected2 {
		t.Errorf("output %q != expected %q", out2, expected2)
	}
}

func TestMannWhitneyMatchesLibrary(t *testing.T) {
	ctx := newTestExecContext(t)
	setupNonparamVars(ctx)

	// mannwhitney a b (two-sided default)
	out, err := runMannWhitney(ctx, "a", "b")
	if err != nil {
		t.Fatalf("mannwhitney a b failed: %v", err)
	}
	expected := libraryMannWhitney(ctx.Vars["a"].(*insyra.DataList), ctx.Vars["b"].(*insyra.DataList), stats.TwoSided)
	if out != expected {
		t.Errorf("output %q != expected %q", out, expected)
	}

	// mannwhitney a b greater
	ctx2 := newTestExecContext(t)
	setupNonparamVars(ctx2)
	out2, err := runMannWhitney(ctx2, "a", "b", "greater")
	if err != nil {
		t.Fatalf("mannwhitney a b greater failed: %v", err)
	}
	expected2 := libraryMannWhitney(ctx2.Vars["a"].(*insyra.DataList), ctx2.Vars["b"].(*insyra.DataList), stats.Greater)
	if out2 != expected2 {
		t.Errorf("output %q != expected %q", out2, expected2)
	}
}

func TestKruskalMatchesLibrary(t *testing.T) {
	ctx := newTestExecContext(t)
	setupNonparamVars(ctx)

	// kruskal g1 g2 g3
	groups := []insyra.IDataList{
		ctx.Vars["g1"].(*insyra.DataList),
		ctx.Vars["g2"].(*insyra.DataList),
		ctx.Vars["g3"].(*insyra.DataList),
	}
	out, err := runKruskal(ctx, "g1", "g2", "g3")
	if err != nil {
		t.Fatalf("kruskal g1 g2 g3 failed: %v", err)
	}
	expected := libraryKruskal(groups)
	if out != expected {
		t.Errorf("output %q != expected %q", out, expected)
	}
}

func TestNonparamErrors(t *testing.T) {
	// misspelled alternative
	ctx := newTestExecContext(t)
	setupNonparamVars(ctx)
	_, err := runMannWhitney(ctx, "a", "b", "grater")
	if err == nil {
		t.Error("mannwhitney a b grater should have failed")
	} else if !strings.Contains(err.Error(), "invalid alternative") {
		t.Errorf("expected invalid alternative error, got %q", err.Error())
	}

	// wilcoxon single missing mu
	ctx2 := newTestExecContext(t)
	setupNonparamVars(ctx2)
	_, err = runWilcoxon(ctx2, "single", "before")
	if err == nil {
		t.Error("wilcoxon single before should have failed")
	} else if err.Error() != "usage: wilcoxon single <var> <mu> [two-sided|greater|less]" {
		t.Errorf("wrong error: got %q", err.Error())
	}

	// wilcoxon single bad mu
	ctx3 := newTestExecContext(t)
	setupNonparamVars(ctx3)
	_, err = runWilcoxon(ctx3, "single", "before", "abc")
	if err == nil {
		t.Error("wilcoxon single before abc should have failed")
	} else if !strings.Contains(err.Error(), "invalid mu") {
		t.Errorf("expected invalid mu error, got %q", err.Error())
	}

	// wilcoxon unknown mode
	ctx4 := newTestExecContext(t)
	setupNonparamVars(ctx4)
	_, err = runWilcoxon(ctx4, "triple", "before")
	if err == nil {
		t.Error("wilcoxon triple before should have failed")
	} else if err.Error() != "unsupported wilcoxon mode: triple" {
		t.Errorf("wrong error: got %q", err.Error())
	}

	// kruskal one group
	ctx5 := newTestExecContext(t)
	setupNonparamVars(ctx5)
	_, err = runKruskal(ctx5, "g1")
	if err == nil {
		t.Error("kruskal g1 should have failed")
	} else if err.Error() != "usage: kruskal <group1> <group2> [groupN]" {
		t.Errorf("wrong error: got %q", err.Error())
	}

	// mannwhitney unknown variable
	ctx6 := newTestExecContext(t)
	setupNonparamVars(ctx6)
	_, err = runMannWhitney(ctx6, "a", "missing")
	if err == nil {
		t.Error("mannwhitney a missing should have failed")
	} else if !strings.Contains(err.Error(), "variable not found") {
		t.Errorf("expected variable not found error, got %q", err.Error())
	}
}

func TestWilcoxonHelpShowsForms(t *testing.T) {
	ctx := newTestExecContext(t)
	if err := runHelpCommand(ctx, []string{"wilcoxon"}); err != nil {
		t.Fatalf("help wilcoxon failed: %v", err)
	}
	got := ctx.Output.(*bytes.Buffer).String()
	for _, want := range []string{
		"wilcoxon paired <var1> <var2>",
		"wilcoxon single <var> <mu>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in help output, got:\n%s", want, got)
		}
	}
}

func TestMannWhitneyHelp(t *testing.T) {
	ctx := newTestExecContext(t)
	if err := runHelpCommand(ctx, []string{"mannwhitney"}); err != nil {
		t.Fatalf("help mannwhitney failed: %v", err)
	}
	got := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(got, "mannwhitney <var1> <var2>") {
		t.Errorf("expected usage in help output, got:\n%s", got)
	}
}

func TestKruskalHelp(t *testing.T) {
	ctx := newTestExecContext(t)
	if err := runHelpCommand(ctx, []string{"kruskal"}); err != nil {
		t.Fatalf("help kruskal failed: %v", err)
	}
	got := ctx.Output.(*bytes.Buffer).String()
	if !strings.Contains(got, "kruskal <group1> <group2>") {
		t.Errorf("expected usage in help output, got:\n%s", got)
	}
}
