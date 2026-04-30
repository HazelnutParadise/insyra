package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

const (
	tolCorr   = 1e-12 // r/rho/tau
	tolCorrP  = 1e-12 // p-value
	tolCorrCI = 1e-10 // Fisher CI
	tolCov    = 1e-10 // covariance (relative — sums of products lose precision)
)

var (
	corrRef  = &refTable{path: "testdata/correlation_reference.txt"}
	corrDump = &labelledFloats{path: "testdata/correlation_data_dump.txt"}
)

func cClose(a, b, tol float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsInf(a, 0) && math.IsInf(b, 0) && math.Signbit(a) == math.Signbit(b) {
		return true
	}
	if math.IsNaN(a) != math.IsNaN(b) || math.IsInf(a, 0) != math.IsInf(b, 0) {
		return false
	}
	if b == 0 {
		return math.Abs(a) <= tol
	}
	return math.Abs(a-b) <= tol*math.Max(1, math.Abs(b))
}

// ============================================================
// Pearson
// ============================================================

type pearsonCase struct {
	name   string
	x, y   []float64
	prefix string
}

func TestPearsonCorrelation_R(t *testing.T) {
	cases := []pearsonCase{
		{"existing", []float64{10, 20, 30, 40, 50}, []float64{15, 22, 29, 41, 48}, "p_existing"},
		{"perfect_pos", []float64{1, 2, 3, 4, 5}, []float64{10, 20, 30, 40, 50}, "p_perfectpos"},
		{"perfect_neg", []float64{1, 2, 3, 4, 5}, []float64{50, 40, 30, 20, 10}, "p_perfectneg"},
		{"random_n5", []float64{5, 1, 3, 2, 4}, []float64{10, 7, 8, 6, 9}, "p_random"},
		{"n10_moderate", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			[]float64{1, 2, 3, 5, 4, 6, 8, 7, 10, 9}, "p_n10"},
		{"n50_weak", corrDump.get(t, "n50_x"), corrDump.get(t, "n50_y"), "p_n50"},
		{"n20_negative", corrDump.get(t, "n20neg_x"), corrDump.get(t, "n20neg_y"), "p_n20neg"},
		{"with_ties", []float64{1, 2, 2, 3, 3, 3, 4, 5, 5, 6},
			[]float64{1, 1, 2, 3, 3, 4, 5, 5, 6, 7}, "p_ties"},
		{"huge_magnitude",
			[]float64{1.0e9, 1.0001e9, 0.9999e9, 1.00005e9, 1.00015e9},
			[]float64{2.0e9, 2.0002e9, 1.9998e9, 2.00010e9, 2.00030e9},
			"p_huge"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x := insyra.NewDataList(c.x)
			y := insyra.NewDataList(c.y)
			r, err := stats.Correlation(x, y, stats.PearsonCorrelation)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expR := corrRef.get(t, c.prefix+".r")
			expP := corrRef.get(t, c.prefix+".p")
			expDF := corrRef.get(t, c.prefix+".df")
			expLo := corrRef.get(t, c.prefix+".cilo")
			expHi := corrRef.get(t, c.prefix+".cihi")

			if !cClose(r.Statistic, expR, tolCorr) {
				t.Errorf("r: got %.17g, want %.17g (Δ=%g)", r.Statistic, expR, math.Abs(r.Statistic-expR))
			}
			if !cClose(r.PValue, expP, tolCorrP) {
				t.Errorf("p: got %.17g, want %.17g (Δ=%g)", r.PValue, expP, math.Abs(r.PValue-expP))
			}
			if r.DF == nil || *r.DF != expDF {
				t.Errorf("df: got %v, want %v", r.DF, expDF)
			}
			if r.CI == nil {
				t.Fatal("CI must not be nil for Pearson")
			}
			if !cClose(r.CI[0], expLo, tolCorrCI) {
				t.Errorf("ci_lo: got %.17g, want %.17g", r.CI[0], expLo)
			}
			if !cClose(r.CI[1], expHi, tolCorrCI) {
				t.Errorf("ci_hi: got %.17g, want %.17g", r.CI[1], expHi)
			}
		})
	}
}

// ============================================================
// Spearman
// ============================================================

type spearmanCase struct {
	name   string
	x, y   []float64
	prefix string
}

func TestSpearmanCorrelation_R(t *testing.T) {
	cases := []spearmanCase{
		{"existing", []float64{10, 20, 30, 40, 50}, []float64{15, 22, 29, 41, 48}, "s_existing"},
		{"perfect_pos", []float64{1, 2, 3, 4, 5}, []float64{10, 20, 30, 40, 50}, "s_perfectpos"},
		{"perfect_neg", []float64{1, 2, 3, 4, 5}, []float64{50, 40, 30, 20, 10}, "s_perfectneg"},
		{"random_n5", []float64{5, 1, 3, 2, 4}, []float64{10, 7, 8, 6, 9}, "s_random"},
		{"n10_moderate", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			[]float64{1, 2, 3, 5, 4, 6, 8, 7, 10, 9}, "s_n10"},
		{"n50_weak", corrDump.get(t, "n50_x"), corrDump.get(t, "n50_y"), "s_n50"},
		{"n20_negative", corrDump.get(t, "n20neg_x"), corrDump.get(t, "n20neg_y"), "s_n20neg"},
		{"with_ties", []float64{1, 2, 2, 3, 3, 3, 4, 5, 5, 6},
			[]float64{1, 1, 2, 3, 3, 4, 5, 5, 6, 7}, "s_ties"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x := insyra.NewDataList(c.x)
			y := insyra.NewDataList(c.y)
			r, err := stats.Correlation(x, y, stats.SpearmanCorrelation)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expR := corrRef.get(t, c.prefix+".rho")
			expP := corrRef.get(t, c.prefix+".p")
			expDF := corrRef.get(t, c.prefix+".df")
			expLo := corrRef.get(t, c.prefix+".cilo")
			expHi := corrRef.get(t, c.prefix+".cihi")

			if !cClose(r.Statistic, expR, tolCorr) {
				t.Errorf("rho: got %.17g, want %.17g (Δ=%g)", r.Statistic, expR, math.Abs(r.Statistic-expR))
			}
			if !cClose(r.PValue, expP, tolCorrP) {
				t.Errorf("p: got %.17g, want %.17g (Δ=%g)", r.PValue, expP, math.Abs(r.PValue-expP))
			}
			if r.DF == nil || *r.DF != expDF {
				t.Errorf("df: got %v, want %v", r.DF, expDF)
			}
			if r.CI == nil {
				t.Fatal("CI must not be nil for Spearman")
			}
			if !cClose(r.CI[0], expLo, tolCorrCI) {
				t.Errorf("ci_lo: got %.17g, want %.17g", r.CI[0], expLo)
			}
			if !cClose(r.CI[1], expHi, tolCorrCI) {
				t.Errorf("ci_hi: got %.17g, want %.17g", r.CI[1], expHi)
			}
		})
	}
}

// ============================================================
// Kendall
// ============================================================

type kendallCase struct {
	name   string
	x, y   []float64
	prefix string
}

func TestKendallCorrelation_R(t *testing.T) {
	cases := []kendallCase{
		{"existing", []float64{10, 20, 30, 40, 50}, []float64{15, 22, 29, 41, 48}, "k_existing"},
		{"perfect_pos", []float64{1, 2, 3, 4, 5}, []float64{10, 20, 30, 40, 50}, "k_perfectpos"},
		{"perfect_neg", []float64{1, 2, 3, 4, 5}, []float64{50, 40, 30, 20, 10}, "k_perfectneg"},
		{"random_n5", []float64{5, 1, 3, 2, 4}, []float64{10, 7, 8, 6, 9}, "k_random"},
		// n=8: boundary — switches from exact permutation (n≤7) to normal approx (n>7)
		{"n8_boundary", []float64{1, 2, 3, 4, 5, 6, 7, 8},
			[]float64{2, 1, 4, 3, 6, 5, 8, 7}, "k_n8"},
		{"n10_moderate", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			[]float64{1, 2, 3, 5, 4, 6, 8, 7, 10, 9}, "k_n10"},
		{"n50_weak", corrDump.get(t, "n50_x"), corrDump.get(t, "n50_y"), "k_n50"},
		{"n20_negative", corrDump.get(t, "n20neg_x"), corrDump.get(t, "n20neg_y"), "k_n20neg"},
		{"with_ties", []float64{1, 2, 2, 3, 3, 3, 4, 5, 5, 6},
			[]float64{1, 1, 2, 3, 3, 4, 5, 5, 6, 7}, "k_ties"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x := insyra.NewDataList(c.x)
			y := insyra.NewDataList(c.y)
			r, err := stats.Correlation(x, y, stats.KendallCorrelation)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expTau := corrRef.get(t, c.prefix+".tau")
			expP := corrRef.get(t, c.prefix+".p")
			if !cClose(r.Statistic, expTau, tolCorr) {
				t.Errorf("tau: got %.17g, want %.17g (Δ=%g)", r.Statistic, expTau, math.Abs(r.Statistic-expTau))
			}
			if !cClose(r.PValue, expP, tolCorrP) {
				t.Errorf("p: got %.17g, want %.17g (Δ=%g)", r.PValue, expP, math.Abs(r.PValue-expP))
			}
			// Kendall does not populate DF or CI in insyra
			if r.DF != nil {
				t.Errorf("Kendall DF must be nil, got %v", *r.DF)
			}
			if r.CI != nil {
				t.Errorf("Kendall CI must be nil, got %v", *r.CI)
			}
		})
	}
}

// ============================================================
// Error / edge paths
// ============================================================

func TestCorrelation_Errors(t *testing.T) {
	const_ := insyra.NewDataList([]float64{3, 3, 3, 3, 3})
	x := insyra.NewDataList([]float64{1, 2, 3, 4, 5})
	for _, m := range []stats.CorrelationMethod{
		stats.PearsonCorrelation, stats.SpearmanCorrelation, stats.KendallCorrelation,
	} {
		if _, err := stats.Correlation(x, const_, m); err == nil {
			t.Errorf("method %v: expected error for constant data, got nil", m)
		}
	}
	short := insyra.NewDataList([]float64{1, 2})
	long := insyra.NewDataList([]float64{1, 2, 3, 4, 5})
	if _, err := stats.Correlation(short, long, stats.PearsonCorrelation); err == nil {
		t.Error("expected error for unequal lengths")
	}
}

// ============================================================
// Covariance
// ============================================================

func TestCovariance_R(t *testing.T) {
	cases := []struct {
		name   string
		x, y   []float64
		prefix string
	}{
		{"basic", []float64{10, 20, 30, 40, 50}, []float64{15, 22, 29, 41, 48}, "cov_basic"},
		{"negative", []float64{1, 2, 3, 4, 5}, []float64{50, 40, 30, 20, 10}, "cov_neg"},
		{"n30_random", corrDump.get(t, "cov_n30_x"), corrDump.get(t, "cov_n30_y"), "cov_n30"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x := insyra.NewDataList(c.x)
			y := insyra.NewDataList(c.y)
			got, err := stats.Covariance(x, y)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			exp := corrRef.get(t, c.prefix+".cov")
			if !cClose(got, exp, tolCov) {
				t.Errorf("cov: got %.17g, want %.17g (Δ=%g)", got, exp, math.Abs(got-exp))
			}
		})
	}
}

func TestCovariance_Errors(t *testing.T) {
	short := insyra.NewDataList([]float64{1, 2})
	long := insyra.NewDataList([]float64{1, 2, 3, 4, 5})
	if _, err := stats.Covariance(short, long); err == nil {
		t.Error("expected error for unequal lengths")
	}
	if _, err := stats.Covariance(insyra.NewDataList([]float64{1}), insyra.NewDataList([]float64{2})); err == nil {
		t.Error("expected error for n<2")
	}
}

// ============================================================
// Bartlett sphericity
// ============================================================

func TestBartlettSphericity_R(t *testing.T) {
	dt3 := insyra.NewDataTable(
		insyra.NewDataList(corrDump.get(t, "bs_3v_v1")),
		insyra.NewDataList(corrDump.get(t, "bs_3v_v2")),
		insyra.NewDataList(corrDump.get(t, "bs_3v_v3")),
	)
	t.Run("3vars", func(t *testing.T) {
		chi, p, df, err := stats.BartlettSphericity(dt3)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		expChi := corrRef.get(t, "bs_3v.chisq")
		expP := corrRef.get(t, "bs_3v.p")
		expDF := int(corrRef.get(t, "bs_3v.df"))
		if !cClose(chi, expChi, tolCorr) {
			t.Errorf("chisq: got %.17g, want %.17g", chi, expChi)
		}
		if !cClose(p, expP, tolCorrP) {
			t.Errorf("p: got %.17g, want %.17g", p, expP)
		}
		if df != expDF {
			t.Errorf("df: got %d, want %d", df, expDF)
		}
	})

	dt4 := insyra.NewDataTable(
		insyra.NewDataList(corrDump.get(t, "bs_4v_a")),
		insyra.NewDataList(corrDump.get(t, "bs_4v_b")),
		insyra.NewDataList(corrDump.get(t, "bs_4v_c")),
		insyra.NewDataList(corrDump.get(t, "bs_4v_d")),
	)
	t.Run("4vars", func(t *testing.T) {
		chi, p, df, err := stats.BartlettSphericity(dt4)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		expChi := corrRef.get(t, "bs_4v.chisq")
		expP := corrRef.get(t, "bs_4v.p")
		expDF := int(corrRef.get(t, "bs_4v.df"))
		if !cClose(chi, expChi, tolCorr) {
			t.Errorf("chisq: got %.17g, want %.17g", chi, expChi)
		}
		if !cClose(p, expP, tolCorrP) {
			t.Errorf("p: got %.17g, want %.17g", p, expP)
		}
		if df != expDF {
			t.Errorf("df: got %d, want %d", df, expDF)
		}
	})
}

func TestBartlettSphericity_Errors(t *testing.T) {
	dt := insyra.NewDataTable(insyra.NewDataList([]float64{1, 2, 3}))
	if _, _, _, err := stats.BartlettSphericity(dt); err == nil {
		t.Error("expected error for fewer than two columns")
	}
}
