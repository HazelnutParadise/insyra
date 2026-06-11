package stats_test

import (
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// Verifies the exact code from Docs/tutorials/nonparametric-tests-when-normality-fails.md
// and Docs/stats.md compiles, runs, and returns sane values.
func TestDocTutorialNonparametricProgram(t *testing.T) {
	before := insyra.NewDataList(3, 4, 2, 5, 3, 4, 2, 3, 4, 2)
	after := insyra.NewDataList(4, 5, 4, 5, 4, 5, 3, 4, 5, 3)
	variantA := insyra.NewDataList(4, 5, 3, 4, 5, 4, 3, 5)
	variantB := insyra.NewDataList(2, 3, 3, 2, 4, 3, 2, 3, 2)

	w, err := stats.PairedWilcoxon(before, after, stats.Less)
	if err != nil {
		t.Fatalf("PairedWilcoxon: %v", err)
	}
	if w.CI == nil || len(w.EffectSizes) == 0 {
		t.Fatalf("Wilcoxon result incomplete: CI=%v ES=%v", w.CI, w.EffectSizes)
	}
	if w.EffectSizes[0].Type != "rank_biserial" {
		t.Errorf("want rank_biserial, got %q", w.EffectSizes[0].Type)
	}
	if w.PValue < 0 || w.PValue > 1 {
		t.Errorf("Wilcoxon p out of range: %v", w.PValue)
	}

	u, err := stats.MannWhitneyU(variantA, variantB, stats.TwoSided)
	if err != nil {
		t.Fatalf("MannWhitneyU: %v", err)
	}
	if len(u.EffectSizes) < 2 || u.EffectSizes[1].Type != "cles_a12" {
		t.Fatalf("MWU effect sizes wrong: %#v", u.EffectSizes)
	}
	if u.CI == nil {
		t.Fatal("MWU CI nil")
	}
	if u.U1+u.U2 != float64(8*9) {
		t.Errorf("U1+U2 should equal n1*n2=72, got %v", u.U1+u.U2)
	}

	kw, err := stats.KruskalWallis(
		insyra.NewDataList(3, 4, 2, 3, 4, 3),
		insyra.NewDataList(4, 5, 4, 5, 4, 5),
		insyra.NewDataList(2, 3, 2, 3, 2, 3),
	)
	if err != nil {
		t.Fatalf("KruskalWallis: %v", err)
	}
	if kw.DF == nil || *kw.DF != 2 {
		t.Errorf("KW df want 2, got %v", kw.DF)
	}
	if kw.EffectSizes[0].Type != "epsilon_squared" {
		t.Errorf("want epsilon_squared, got %q", kw.EffectSizes[0].Type)
	}
	if len(kw.GroupRankSum) != 3 || kw.NTotal != 18 {
		t.Errorf("KW group/total wrong: rs=%v total=%d", kw.GroupRankSum, kw.NTotal)
	}

	fr, err := stats.FriedmanTest(
		insyra.NewDataList(3, 4, 2),
		insyra.NewDataList(4, 5, 3),
		insyra.NewDataList(2, 4, 2),
		insyra.NewDataList(3, 5, 3),
		insyra.NewDataList(4, 5, 2),
	)
	if err != nil {
		t.Fatalf("FriedmanTest: %v", err)
	}
	if fr.DF == nil || *fr.DF != 2 {
		t.Errorf("Friedman df want 2, got %v", fr.DF)
	}
	if fr.NSubjects != 5 || fr.KConditions != 3 {
		t.Errorf("Friedman n/k wrong: n=%d k=%d", fr.NSubjects, fr.KConditions)
	}
	if fr.EffectSizes[0].Type != "kendalls_w" {
		t.Errorf("want kendalls_w, got %q", fr.EffectSizes[0].Type)
	}

	// Single-sample Wilcoxon from the stats.md example block.
	ss, err := stats.SingleSampleWilcoxon(
		insyra.NewDataList(3.0, 4, 2, 5, 3, 4, 2, 3), 2.5, stats.Greater)
	if err != nil {
		t.Fatalf("SingleSampleWilcoxon: %v", err)
	}
	if ss.NEffective <= 0 {
		t.Errorf("NEffective should be > 0, got %d", ss.NEffective)
	}
}
