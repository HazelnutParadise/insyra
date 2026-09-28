package stats_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// The z-tests take their alternative hypothesis and confidence level in an
// optional ZTestOptions. The zero value has to mean exactly what spelling both
// settings out does, or every existing caller changes its numbers.
func TestZTestOptionsZeroValueMatchesTwoSidedAt95(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)

	got, err := stats.SingleSampleZTest(x, 50, 10)
	if err != nil {
		t.Fatalf("SingleSampleZTest without options: %v", err)
	}
	want, err := stats.SingleSampleZTest(x, 50, 10, stats.ZTestOptions{
		Alternative:     stats.TwoSided,
		ConfidenceLevel: 0.95,
	})
	if err != nil {
		t.Fatalf("SingleSampleZTest with explicit options: %v", err)
	}

	if got.Statistic != want.Statistic {
		t.Errorf("Statistic: got %v, want %v", got.Statistic, want.Statistic)
	}
	if got.PValue != want.PValue {
		t.Errorf("PValue: got %v, want %v", got.PValue, want.PValue)
	}
	if got.CI[0] != want.CI[0] {
		t.Errorf("CI[0]: got %v, want %v", got.CI[0], want.CI[0])
	}
	if got.CI[1] != want.CI[1] {
		t.Errorf("CI[1]: got %v, want %v", got.CI[1], want.CI[1])
	}
	if got.Mean != want.Mean {
		t.Errorf("Mean: got %v, want %v", got.Mean, want.Mean)
	}
	if got.N != want.N {
		t.Errorf("N: got %v, want %v", got.N, want.N)
	}
	if len(got.EffectSizes) != 1 || len(want.EffectSizes) != 1 {
		t.Fatalf("expected one effect size each, got %+v and %+v", got.EffectSizes, want.EffectSizes)
	}
	if got.EffectSizes[0].Value != want.EffectSizes[0].Value {
		t.Errorf("effect size: got %v, want %v", got.EffectSizes[0].Value, want.EffectSizes[0].Value)
	}
}

func TestZTestOptionsConfidenceLevelErrors(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)

	cases := []struct {
		name string
		cl   float64
		want string
	}{
		{name: "NaN", cl: math.NaN(), want: "confidence level must be strictly between 0 and 1, got NaN"},
		{name: "one", cl: 1, want: "confidence level must be strictly between 0 and 1, got 1"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := stats.SingleSampleZTest(x, 50, 10, stats.ZTestOptions{ConfidenceLevel: c.cl})
			if got != nil {
				t.Errorf("result must be nil on error, got %+v", got)
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			if err.Error() != c.want {
				t.Errorf("error: got %q, want %q", err.Error(), c.want)
			}
		})
	}
}

func TestZTestOptionsRejectsUnknownAlternative(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)

	got, err := stats.SingleSampleZTest(x, 50, 10, stats.ZTestOptions{Alternative: "sideways"})
	if got != nil {
		t.Errorf("result must be nil on error, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := `alternative must be two-sided, greater or less, got "sideways"`; err.Error() != want {
		t.Errorf("error: got %q, want %q", err.Error(), want)
	}
}

func TestZTestRejectsMoreThanOneOptionsValue(t *testing.T) {
	a := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	b := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	got, err := stats.TwoSampleZTest(a, b, 10, 10,
		stats.ZTestOptions{Alternative: stats.Greater},
		stats.ZTestOptions{Alternative: stats.Less},
	)
	if got != nil {
		t.Errorf("result must be nil on error, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := "at most one stats.ZTestOptions may be given, got 2"; err.Error() != want {
		t.Errorf("error: got %q, want %q", err.Error(), want)
	}
}

// The Wilcoxon tests take their alternative hypothesis and confidence level in
// an optional WilcoxonOptions. The zero value has to mean exactly what spelling
// both settings out does, or every existing caller changes its numbers.
func TestPairedWilcoxonZeroValueMatchesTwoSidedAt95(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	got, err := stats.PairedWilcoxon(x, y)
	if err != nil {
		t.Fatalf("PairedWilcoxon without options: %v", err)
	}
	want, err := stats.PairedWilcoxon(x, y, stats.WilcoxonOptions{
		Alternative:     stats.TwoSided,
		ConfidenceLevel: 0.95,
	})
	if err != nil {
		t.Fatalf("PairedWilcoxon with explicit options: %v", err)
	}

	if got.Statistic != want.Statistic {
		t.Errorf("Statistic: got %v, want %v", got.Statistic, want.Statistic)
	}
	if got.PValue != want.PValue {
		t.Errorf("PValue: got %v, want %v", got.PValue, want.PValue)
	}
	if got.CI[0] != want.CI[0] {
		t.Errorf("CI[0]: got %v, want %v", got.CI[0], want.CI[0])
	}
	if got.CI[1] != want.CI[1] {
		t.Errorf("CI[1]: got %v, want %v", got.CI[1], want.CI[1])
	}
}

func TestSingleSampleWilcoxonOptionsConfidenceLevelErrors(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)

	cases := []struct {
		name string
		cl   float64
		want string
	}{
		{name: "NaN", cl: math.NaN(), want: "confidence level must be strictly between 0 and 1, got NaN"},
		{name: "one", cl: 1, want: "confidence level must be strictly between 0 and 1, got 1"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := stats.SingleSampleWilcoxon(x, 50, stats.WilcoxonOptions{ConfidenceLevel: c.cl})
			if got != nil {
				t.Errorf("result must be nil on error, got %+v", got)
			}
			if err == nil {
				t.Fatal("expected an error")
			}
			if err.Error() != c.want {
				t.Errorf("error: got %q, want %q", err.Error(), c.want)
			}
		})
	}
}

func TestPairedWilcoxonRejectsUnknownAlternative(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	got, err := stats.PairedWilcoxon(x, y, stats.WilcoxonOptions{Alternative: "up"})
	if got != nil {
		t.Errorf("result must be nil on error, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := `alternative must be two-sided, greater or less, got "up"`; err.Error() != want {
		t.Errorf("error: got %q, want %q", err.Error(), want)
	}
}

// MannWhitneyU takes its settings in an optional MannWhitneyUOptions, and its
// zero value has to mean exactly what spelling both out does.
func TestMannWhitneyUOptionsZeroValueMatchesTwoSidedAt95(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	got, err := stats.MannWhitneyU(x, y, stats.MannWhitneyUOptions{})
	if err != nil {
		t.Fatalf("MannWhitneyU with the zero value: %v", err)
	}
	want, err := stats.MannWhitneyU(x, y, stats.MannWhitneyUOptions{
		Alternative:     stats.TwoSided,
		ConfidenceLevel: 0.95,
	})
	if err != nil {
		t.Fatalf("MannWhitneyU with explicit options: %v", err)
	}

	if got.Statistic != want.Statistic {
		t.Errorf("Statistic: got %v, want %v", got.Statistic, want.Statistic)
	}
	if got.PValue != want.PValue {
		t.Errorf("PValue: got %v, want %v", got.PValue, want.PValue)
	}
	if got.CI[0] != want.CI[0] {
		t.Errorf("CI[0]: got %v, want %v", got.CI[0], want.CI[0])
	}
	if got.CI[1] != want.CI[1] {
		t.Errorf("CI[1]: got %v, want %v", got.CI[1], want.CI[1])
	}
	if got.U1 != want.U1 {
		t.Errorf("U1: got %v, want %v", got.U1, want.U1)
	}
	if got.U2 != want.U2 {
		t.Errorf("U2: got %v, want %v", got.U2, want.U2)
	}
}

func TestMannWhitneyURejectsMoreThanOneOptionsValue(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	got, err := stats.MannWhitneyU(x, y, stats.MannWhitneyUOptions{}, stats.MannWhitneyUOptions{})
	if got != nil {
		t.Errorf("result must be nil on error, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := "at most one stats.MannWhitneyUOptions may be given, got 2"; err.Error() != want {
		t.Errorf("error: got %q, want %q", err.Error(), want)
	}
}

// A one-sided test reports a half-open interval, which is existing behaviour
// the options struct must not change.
func TestZTestOptionsGreaterKeepsInfiniteUpperEnd(t *testing.T) {
	x := insyra.NewDataList(105, 110, 102, 108, 107, 112, 100, 109, 106, 111)

	got, err := stats.SingleSampleZTest(x, 100, 5, stats.ZTestOptions{Alternative: stats.Greater})
	if err != nil {
		t.Fatalf("SingleSampleZTest: %v", err)
	}
	if !math.IsInf(got.CI[1], 1) {
		t.Errorf("CI upper end: got %v, want +Inf", got.CI[1])
	}
}

// The t-tests take their alternative hypothesis and confidence level in an
// optional TTestOptions. The numbers below come from R 4.6.1 t.test.
func TestSingleSampleTTestOneSidedMatchesR(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)

	got, err := stats.SingleSampleTTest(x, 50, stats.TTestOptions{Alternative: stats.Greater})
	if err != nil {
		t.Fatalf("SingleSampleTTest: %v", err)
	}

	if math.Abs(got.Statistic-1.141068052) > 1e-8 {
		t.Errorf("Statistic: got %.17g, want 1.141068052", got.Statistic)
	}
	if math.Abs(got.PValue-0.1486647578) > 1e-8 {
		t.Errorf("PValue: got %.17g, want 0.1486647578", got.PValue)
	}
	if math.Abs(got.CI[0]-48.00161745) > 1e-8 {
		t.Errorf("CI[0]: got %.17g, want 48.00161745", got.CI[0])
	}
	if !math.IsInf(got.CI[1], 1) {
		t.Errorf("CI[1]: got %v, want +Inf", got.CI[1])
	}
	if got.DF == nil || *got.DF != 6 {
		t.Errorf("DF: got %v, want 6", got.DF)
	}
}

func TestPairedTTestOneSidedMatchesR(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	got, err := stats.PairedTTest(x, y, stats.TTestOptions{
		Alternative:     stats.Less,
		ConfidenceLevel: 0.9,
	})
	if err != nil {
		t.Fatalf("PairedTTest: %v", err)
	}

	if math.Abs(got.Statistic-6.0869715254) > 1e-8 {
		t.Errorf("Statistic: got %.17g, want 6.0869715254", got.Statistic)
	}
	if math.Abs(got.PValue-0.9995528678) > 1e-8 {
		t.Errorf("PValue: got %.17g, want 0.9995528678", got.PValue)
	}
	if !math.IsInf(got.CI[0], -1) {
		t.Errorf("CI[0]: got %v, want -Inf", got.CI[0])
	}
	if math.Abs(got.CI[1]-3.6389332568) > 1e-8 {
		t.Errorf("CI[1]: got %.17g, want 3.6389332568", got.CI[1])
	}
}

func TestTTestOptionsZeroValueMatchesTwoSidedAt95(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)

	got, err := stats.SingleSampleTTest(x, 50)
	if err != nil {
		t.Fatalf("SingleSampleTTest without options: %v", err)
	}
	if math.Abs(got.PValue-0.2973295155) > 1e-8 {
		t.Errorf("PValue: got %.17g, want 0.2973295155", got.PValue)
	}

	want, err := stats.SingleSampleTTest(x, 50, stats.TTestOptions{
		Alternative:     stats.TwoSided,
		ConfidenceLevel: 0.95,
	})
	if err != nil {
		t.Fatalf("SingleSampleTTest with explicit options: %v", err)
	}

	if got.Statistic != want.Statistic {
		t.Errorf("Statistic: got %v, want %v", got.Statistic, want.Statistic)
	}
	if got.PValue != want.PValue {
		t.Errorf("PValue: got %v, want %v", got.PValue, want.PValue)
	}
	if *got.DF != *want.DF {
		t.Errorf("DF: got %v, want %v", *got.DF, *want.DF)
	}
	if got.CI[0] != want.CI[0] {
		t.Errorf("CI[0]: got %v, want %v", got.CI[0], want.CI[0])
	}
	if got.CI[1] != want.CI[1] {
		t.Errorf("CI[1]: got %v, want %v", got.CI[1], want.CI[1])
	}
	if got.Mean != want.Mean {
		t.Errorf("Mean: got %v, want %v", got.Mean, want.Mean)
	}
	if len(got.EffectSizes) != 1 || len(want.EffectSizes) != 1 {
		t.Fatalf("expected one effect size each, got %+v and %+v", got.EffectSizes, want.EffectSizes)
	}
	if got.EffectSizes[0].Value != want.EffectSizes[0].Value {
		t.Errorf("effect size: got %v, want %v", got.EffectSizes[0].Value, want.EffectSizes[0].Value)
	}
}

// Constant data has no variance, so the statistic is ±Inf and the p-value
// follows the alternative instead of being hard-coded to 0.
func TestSingleSampleTTestConstantDataPValueFollowsAlternative(t *testing.T) {
	dl := insyra.NewDataList(5.0, 5.0, 5.0)

	less, err := stats.SingleSampleTTest(dl, 4, stats.TTestOptions{Alternative: stats.Less})
	if err != nil {
		t.Fatalf("SingleSampleTTest with Less: %v", err)
	}
	if !math.IsInf(less.Statistic, 1) {
		t.Errorf("Statistic with Less: got %v, want +Inf", less.Statistic)
	}
	if less.PValue != 1 {
		t.Errorf("PValue with Less: got %.17g, want exactly 1", less.PValue)
	}

	greater, err := stats.SingleSampleTTest(dl, 4, stats.TTestOptions{Alternative: stats.Greater})
	if err != nil {
		t.Fatalf("SingleSampleTTest with Greater: %v", err)
	}
	if !math.IsInf(greater.Statistic, 1) {
		t.Errorf("Statistic with Greater: got %v, want +Inf", greater.Statistic)
	}
	if greater.PValue != 0 {
		t.Errorf("PValue with Greater: got %.17g, want exactly 0", greater.PValue)
	}
}

func TestTTestOptionsConfidenceLevelErrors(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	got, err := stats.TwoSampleTTest(x, y, false, stats.TTestOptions{ConfidenceLevel: math.NaN()})
	if got != nil {
		t.Errorf("result must be nil on error, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := "confidence level must be strictly between 0 and 1, got NaN"; err.Error() != want {
		t.Errorf("error: got %q, want %q", err.Error(), want)
	}
}

func TestTTestRejectsUnknownAlternative(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)
	y := insyra.NewDataList(48.0, 55.0, 53.2, 50.1, 57.3, 40.2, 45.5)

	got, err := stats.PairedTTest(x, y, stats.TTestOptions{Alternative: "sideways"})
	if got != nil {
		t.Errorf("result must be nil on error, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := `alternative must be two-sided, greater or less, got "sideways"`; err.Error() != want {
		t.Errorf("error: got %q, want %q", err.Error(), want)
	}
}

func TestTTestRejectsMoreThanOneOptionsValue(t *testing.T) {
	x := insyra.NewDataList(52.1, 58.3, 57.4, 51.3, 61.2, 42.8, 46.8)

	got, err := stats.SingleSampleTTest(x, 50, stats.TTestOptions{}, stats.TTestOptions{})
	if got != nil {
		t.Errorf("result must be nil on error, got %+v", got)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := "at most one stats.TTestOptions may be given, got 2"; err.Error() != want {
		t.Errorf("error: got %q, want %q", err.Error(), want)
	}
}

// The k-sample tests take their lists as one []insyra.IDataList, the same shape
// LeveneTest and BartlettTest already take. A nil slice has to be refused the
// way an empty call was, not reach the arithmetic.
func TestKSampleTestsTakeASlice(t *testing.T) {
	t.Run("nil slice is refused", func(t *testing.T) {
		if got, err := stats.OneWayANOVA(nil); err == nil || got != nil {
			t.Errorf("OneWayANOVA(nil): got result %+v and error %v, want nil result and an error", got, err)
		}
		if got, err := stats.TwoWayANOVA(2, 2, nil); err == nil || got != nil {
			t.Errorf("TwoWayANOVA(2, 2, nil): got result %+v and error %v, want nil result and an error", got, err)
		}
		if got, err := stats.RepeatedMeasuresANOVA(nil); err == nil || got != nil {
			t.Errorf("RepeatedMeasuresANOVA(nil): got result %+v and error %v, want nil result and an error", got, err)
		}
		if got, err := stats.KruskalWallis(nil); err == nil || got != nil {
			t.Errorf("KruskalWallis(nil): got result %+v and error %v, want nil result and an error", got, err)
		}
		if got, err := stats.FriedmanTest(nil); err == nil || got != nil {
			t.Errorf("FriedmanTest(nil): got result %+v and error %v, want nil result and an error", got, err)
		}
	})

	t.Run("one-way over three groups of three", func(t *testing.T) {
		a := insyra.NewDataList(1.0, 2.0, 3.0)
		b := insyra.NewDataList(2.0, 3.0, 4.5)
		c := insyra.NewDataList(5.0, 6.0, 7.5)

		got, err := stats.OneWayANOVA([]insyra.IDataList{a, b, c})
		if err != nil {
			t.Fatalf("OneWayANOVA: %v", err)
		}
		if got.Factor.DF != 2 {
			t.Errorf("Factor.DF: got %d, want 2", got.Factor.DF)
		}
		if got.Within.DF != 6 {
			t.Errorf("Within.DF: got %d, want 6", got.Within.DF)
		}
	})
}

// FactorAnalysis takes its settings in an optional FactorAnalysisOptions. The
// zero value has to mean exactly what DefaultFactorAnalysisOptions spells out,
// and a second options value is an error rather than a silent first-wins.
func TestFactorAnalysisRunsWithoutOptions(t *testing.T) {
	dt := dataTableFromRows(factorAnalysisRows())

	got, err := stats.FactorAnalysis(dt)
	if err != nil {
		t.Fatalf("FactorAnalysis without options: %v", err)
	}
	want, err := stats.FactorAnalysis(dt, stats.DefaultFactorAnalysisOptions())
	if err != nil {
		t.Fatalf("FactorAnalysis with the default options: %v", err)
	}

	if got.CountUsed != want.CountUsed {
		t.Errorf("CountUsed: got %d, want %d", got.CountUsed, want.CountUsed)
	}
	gotLoadings := got.Loadings.(*insyra.DataTable).To2DSlice()
	wantLoadings := want.Loadings.(*insyra.DataTable).To2DSlice()
	if !reflect.DeepEqual(gotLoadings, wantLoadings) {
		t.Errorf("Loadings: got %v, want %v", gotLoadings, wantLoadings)
	}

	two, err := stats.FactorAnalysis(dt, stats.FactorAnalysisOptions{}, stats.FactorAnalysisOptions{})
	if two != nil {
		t.Errorf("model must be nil on error, got %+v", two)
	}
	if err == nil {
		t.Fatal("expected an error")
	}
	if want := "at most one stats.FactorAnalysisOptions may be given, got 2"; err.Error() != want {
		t.Errorf("error: got %q, want %q", err.Error(), want)
	}
}
