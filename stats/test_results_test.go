package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// Every hypothesis test's result type satisfies HypothesisTestResult through
// the TestResult it embeds.
var (
	_ stats.HypothesisTestResult = (*stats.TTestResult)(nil)
	_ stats.HypothesisTestResult = (*stats.ZTestResult)(nil)
	_ stats.HypothesisTestResult = (*stats.FTestResult)(nil)
	_ stats.HypothesisTestResult = (*stats.ChiSquareTestResult)(nil)
	_ stats.HypothesisTestResult = (*stats.CorrelationResult)(nil)
	_ stats.HypothesisTestResult = (*stats.WilcoxonTestResult)(nil)
	_ stats.HypothesisTestResult = (*stats.MannWhitneyUResult)(nil)
	_ stats.HypothesisTestResult = (*stats.KruskalWallisResult)(nil)
	_ stats.HypothesisTestResult = (*stats.FriedmanTestResult)(nil)
)

func TestBaseReturnsTheEmbeddedResult(t *testing.T) {
	a := insyra.NewDataList(55.1, 49.3, 58.2, 61.9, 47.3, 51.0)
	b := insyra.NewDataList(46.9, 41.2, 45.7, 49.8, 44.0, 47.6)

	res, err := stats.TwoSampleTTest(a, b)
	if err != nil {
		t.Fatalf("TwoSampleTTest: %v", err)
	}
	if res.Base() != &res.TestResult {
		t.Errorf("Base() = %p, want &res.TestResult = %p", res.Base(), &res.TestResult)
	}
}

func TestOneFunctionOverAnyTest(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	tt, err := stats.SingleSampleTTest(x, 50)
	if err != nil {
		t.Fatalf("SingleSampleTTest: %v", err)
	}
	mwu, err := stats.MannWhitneyU(x, y)
	if err != nil {
		t.Fatalf("MannWhitneyU: %v", err)
	}
	chi, err := stats.ChiSquareIndependenceTest(
		insyra.NewDataList("a", "a", "b", "b", "a", "b", "a", "b"),
		insyra.NewDataList("x", "y", "x", "y", "x", "x", "y", "y"),
	)
	if err != nil {
		t.Fatalf("ChiSquareIndependenceTest: %v", err)
	}

	results := []stats.HypothesisTestResult{tt, mwu, chi}
	for i, r := range results {
		if got, want := r.Base().PValue, []float64{tt.PValue, mwu.PValue, chi.PValue}[i]; got != want {
			t.Errorf("result %d: Base().PValue = %v, want %v", i, got, want)
		}
		if p := r.Base().PValue; p < 0 || p > 1 {
			t.Errorf("result %d: Base().PValue = %v, want between 0 and 1", i, p)
		}
	}
}

// The paired test reports both group means and both sample sizes, so a
// caller never has to reach for a NaN or a 0 to learn that a field does not
// apply. The four means are the arithmetic of the two lists, not values
// transcribed from a reference implementation.
func TestPairedTTestReportsBothMeans(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	r, err := stats.PairedTTest(x, y)
	if err != nil {
		t.Fatalf("PairedTTest: %v", err)
	}

	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"Mean", r.Mean, 52.84285714285714},          // 369.9/7
		{"Mean2", *r.Mean2, 49.9},                    // 349.3/7
		{"MeanDiff", *r.MeanDiff, 2.942857142857141}, // 20.6/7
	} {
		if math.Abs(c.got-c.want) > 1e-12 {
			t.Errorf("%s = %.17g, want %.17g", c.name, c.got, c.want)
		}
	}
	if r.N != 7 {
		t.Errorf("N = %d, want 7", r.N)
	}
	if *r.N2 != 7 {
		t.Errorf("N2 = %d, want 7", *r.N2)
	}
}

// A field that only some calls fill is nil when it does not apply; a field
// every call fills is a plain value. Neither case may report itself as NaN
// or 0.
func TestTTestNilFieldsFollowTheRule(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	single, err := stats.SingleSampleTTest(x, 50)
	if err != nil {
		t.Fatalf("SingleSampleTTest: %v", err)
	}
	if single.Mean2 != nil {
		t.Errorf("single-sample Mean2 = %v, want nil", *single.Mean2)
	}
	if single.N2 != nil {
		t.Errorf("single-sample N2 = %v, want nil", *single.N2)
	}
	if single.MeanDiff != nil {
		t.Errorf("single-sample MeanDiff = %v, want nil", *single.MeanDiff)
	}

	two, err := stats.TwoSampleTTest(x, y)
	if err != nil {
		t.Fatalf("TwoSampleTTest: %v", err)
	}
	if two.MeanDiff != nil {
		t.Errorf("two-sample MeanDiff = %v, want nil", *two.MeanDiff)
	}
	if two.Mean2 == nil {
		t.Error("two-sample Mean2 = nil, want the second mean")
	}
	if two.N2 == nil {
		t.Error("two-sample N2 = nil, want the second sample size")
	}
}

// Z is the asymptotic standardization, so it is nil wherever no z was
// computed: the exact distribution and the "undefined" all-zero-differences
// return. A tie is what pushes an untied-but-small sample onto the
// asymptotic path.
func TestRankTestZIsNilWithoutAsymptotics(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	exact, err := stats.SingleSampleWilcoxon(x, 50)
	if err != nil {
		t.Fatalf("SingleSampleWilcoxon (untied): %v", err)
	}
	if exact.Method != "exact" {
		t.Errorf("untied Method = %q, want \"exact\"", exact.Method)
	}
	if exact.Z != nil {
		t.Errorf("untied Z = %v, want nil", *exact.Z)
	}

	tied, err := stats.SingleSampleWilcoxon(
		insyra.NewDataList(1.0, 2.0, 2.0, 3.0, 4.0, 4.0, 5.0), 0)
	if err != nil {
		t.Fatalf("SingleSampleWilcoxon (tied): %v", err)
	}
	if tied.Method != "asymptotic" {
		t.Errorf("tied Method = %q, want \"asymptotic\"", tied.Method)
	}
	if tied.Z == nil {
		t.Fatal("tied Z = nil, want the asymptotic z")
	}
	if math.IsNaN(*tied.Z) || math.IsInf(*tied.Z, 0) {
		t.Errorf("tied Z = %v, want a finite value", *tied.Z)
	}

	mwu, err := stats.MannWhitneyU(x, y)
	if err != nil {
		t.Fatalf("MannWhitneyU: %v", err)
	}
	if mwu.Method != "exact" {
		t.Errorf("untied Method = %q, want \"exact\"", mwu.Method)
	}
	if mwu.Z != nil {
		t.Errorf("untied Z = %v, want nil", *mwu.Z)
	}

	undef, err := stats.SingleSampleWilcoxon(insyra.NewDataList(50.0, 50.0, 50.0), 50)
	if err != nil {
		t.Fatalf("SingleSampleWilcoxon (all-zero differences): %v", err)
	}
	if undef.Method != "undefined" {
		t.Errorf("all-zero Method = %q, want \"undefined\"", undef.Method)
	}
	if undef.Z != nil {
		t.Errorf("all-zero Z = %v, want nil", *undef.Z)
	}
}

// Bartlett's statistic is a chi-square with one degree of freedom, so a
// second one is not a value the test failed to compute — there is none. A
// Levene F, which really does have two, reports it.
func TestBartlettHasNoSecondDF(t *testing.T) {
	a := insyra.NewDataList(1.0, 2.0, 3.5, 4.0)
	b := insyra.NewDataList(2.0, 4.5, 6.0, 9.0)
	c := insyra.NewDataList(1.0, 1.5, 2.2, 2.4)
	groups := []insyra.IDataList{a, b, c}

	bart, err := stats.BartlettTest(groups)
	if err != nil {
		t.Fatalf("BartlettTest: %v", err)
	}
	if bart.DF == nil || *bart.DF != 2 {
		t.Errorf("BartlettTest DF = %v, want 2", bart.DF)
	}
	if bart.DF2 != nil {
		t.Errorf("BartlettTest DF2 = %v, want nil", *bart.DF2)
	}

	levene, err := stats.LeveneTest(groups)
	if err != nil {
		t.Fatalf("LeveneTest: %v", err)
	}
	if levene.DF2 == nil {
		t.Fatal("LeveneTest DF2 = nil, want 9")
	}
	if *levene.DF2 != 9 {
		t.Errorf("LeveneTest DF2 = %v, want 9", *levene.DF2)
	}
}
