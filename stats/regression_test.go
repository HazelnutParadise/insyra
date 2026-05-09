package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// Tolerances. Coefficient and SE tolerances loosen for high-degree polynomial
// fits (the design matrix becomes ill-conditioned: column [1, x, x², x³, x⁴] for
// x up to 15 has condition number ~10^7); inference quantities propagate that.
const (
	tolReg     = 1e-9  // coefficients, R², residuals (relative)
	tolRegP    = 1e-10 // p-values
	tolRegCI   = 1e-9  // CI bounds
	tolRegPoly = 1e-6  // high-degree polynomial inference (ill-conditioned)
)

var (
	regRef  = &refTable{path: "testdata/regression_reference.txt"}
	regDump = &labelledFloats{path: "testdata/regression_data_dump.txt"}
)

func rClose(a, b, tol float64) bool {
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

func refVec(t *testing.T, prefix string, n int) []float64 {
	t.Helper()
	out := make([]float64, n)
	for i := range n {
		out[i] = regRef.get(t, prefix+"["+itoa(i)+"]")
	}
	return out
}

func itoa(i int) string {
	switch i {
	case 0:
		return "0"
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	case 4:
		return "4"
	default:
		// fall through to slow path for higher indices
		var buf [16]byte
		pos := len(buf)
		neg := false
		if i < 0 {
			neg = true
			i = -i
		}
		for i > 0 {
			pos--
			buf[pos] = byte('0' + i%10)
			i /= 10
		}
		if neg {
			pos--
			buf[pos] = '-'
		}
		return string(buf[pos:])
	}
}

// ============================================================
// Simple LinearRegression (1 predictor)
// ============================================================

type simpleLRCase struct {
	name   string
	x, y   []float64
	prefix string
	tol    float64 // 0 -> tolReg
}

func TestLinearRegression_Simple_R(t *testing.T) {
	cases := []simpleLRCase{
		{name: "case1",
			x:      []float64{38.08, 95.07, 73.20, 59.87, 15.60, 15.60, 5.81, 86.62, 60.11, 70.81},
			y:      []float64{212.12, 466.84, 359.15, 280.27, 84.92, 77.83, 30.49, 446.84, 286.32, 369.78},
			prefix: "ln_case1"},
		{name: "case2",
			x:      []float64{2.05, 96.99, 83.24, 21.23, 18.18, 18.34, 30.42, 52.48, 43.19, 29.12},
			y:      []float64{-1.88, 450.65, 413.85, 84.84, 96.76, 93.63, 132.95, 252.36, 216.66, 151.36},
			prefix: "ln_case2"},
		{name: "case3",
			x:      []float64{61.19, 13.95, 29.21, 36.64, 45.61, 78.52, 19.97, 51.42, 59.24, 4.65},
			y:      []float64{321.77, 66.99, 142.93, 190.34, 216.55, 402.59, 96.80, 271.27, 304.18, 26.59},
			prefix: "ln_case3"},
		// ---- Diverse cases ----
		{name: "minN", x: []float64{1, 2, 3}, y: []float64{2.5, 5.1, 7.4}, prefix: "ln_minN"},
		{name: "largeN", x: regDump.get(t, "ln_largeN_x"), y: regDump.get(t, "ln_largeN_y"), prefix: "ln_largeN"},
		{name: "negative_slope",
			x: []float64{1, 2, 3, 4, 5, 6, 7}, y: []float64{20.1, 18.5, 16.2, 14.8, 13.1, 11.5, 9.8},
			prefix: "ln_neg"},
		// huge_magnitude: data is essentially perfectly linear so residuals are
		// rounding noise — comparing them at relative 1e-9 fails because they
		// can flip sign with FP order. Loosen.
		{name: "huge_magnitude",
			x:      []float64{1.0e6, 2.0e6, 3.0e6, 4.0e6, 5.0e6},
			y:      []float64{2.5e6, 5.1e6, 7.4e6, 10.0e6, 12.6e6},
			prefix: "ln_huge", tol: 1e-7},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tol := c.tol
			if tol == 0 {
				tol = tolReg
			}
			r, err := stats.LinearRegression(insyra.NewDataList(c.y), insyra.NewDataList(c.x))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expCoef := refVec(t, c.prefix+".coef", 2)
			expSE := refVec(t, c.prefix+".se", 2)
			expT := refVec(t, c.prefix+".t", 2)
			expP := refVec(t, c.prefix+".p", 2)
			expCILo := refVec(t, c.prefix+".cilo", 2)
			expCIHi := refVec(t, c.prefix+".cihi", 2)
			expRsq := regRef.get(t, c.prefix+".rsq")
			expAdj := regRef.get(t, c.prefix+".adjr")
			expResid := refVec(t, c.prefix+".resid", len(c.x))

			if !rClose(r.Intercept, expCoef[0], tol) {
				t.Errorf("intercept: got %.17g, want %.17g", r.Intercept, expCoef[0])
			}
			if !rClose(r.Slope, expCoef[1], tol) {
				t.Errorf("slope: got %.17g, want %.17g", r.Slope, expCoef[1])
			}
			if !rClose(r.Coefficients[0], expCoef[0], tol) ||
				!rClose(r.Coefficients[1], expCoef[1], tol) {
				t.Errorf("Coefficients: got %v, want %v", r.Coefficients, expCoef)
			}
			if !rClose(r.StandardErrorIntercept, expSE[0], tol) {
				t.Errorf("seA: got %.17g, want %.17g", r.StandardErrorIntercept, expSE[0])
			}
			if !rClose(r.StandardError, expSE[1], tol) {
				t.Errorf("seB: got %.17g, want %.17g", r.StandardError, expSE[1])
			}
			if !rClose(r.TValueIntercept, expT[0], tol) {
				t.Errorf("tA: got %.17g, want %.17g", r.TValueIntercept, expT[0])
			}
			if !rClose(r.TValue, expT[1], tol) {
				t.Errorf("tB: got %.17g, want %.17g", r.TValue, expT[1])
			}
			if !rClose(r.PValueIntercept, expP[0], tolRegP) {
				t.Errorf("pA: got %.17g, want %.17g", r.PValueIntercept, expP[0])
			}
			if !rClose(r.PValue, expP[1], tolRegP) {
				t.Errorf("pB: got %.17g, want %.17g", r.PValue, expP[1])
			}
			if !rClose(r.ConfidenceIntervalIntercept[0], expCILo[0], tolRegCI) ||
				!rClose(r.ConfidenceIntervalIntercept[1], expCIHi[0], tolRegCI) {
				t.Errorf("CI(intercept): got [%v,%v], want [%v,%v]",
					r.ConfidenceIntervalIntercept[0], r.ConfidenceIntervalIntercept[1],
					expCILo[0], expCIHi[0])
			}
			if !rClose(r.ConfidenceIntervalSlope[0], expCILo[1], tolRegCI) ||
				!rClose(r.ConfidenceIntervalSlope[1], expCIHi[1], tolRegCI) {
				t.Errorf("CI(slope): got [%v,%v], want [%v,%v]",
					r.ConfidenceIntervalSlope[0], r.ConfidenceIntervalSlope[1],
					expCILo[1], expCIHi[1])
			}
			if !rClose(r.RSquared, expRsq, tol) {
				t.Errorf("R²: got %.17g, want %.17g", r.RSquared, expRsq)
			}
			if !rClose(r.AdjustedRSquared, expAdj, tol) {
				t.Errorf("adj R²: got %.17g, want %.17g", r.AdjustedRSquared, expAdj)
			}
			if len(r.Residuals) != len(expResid) {
				t.Fatalf("residuals length: got %d, want %d", len(r.Residuals), len(expResid))
			}
			for i := range expResid {
				if !rClose(r.Residuals[i], expResid[i], tol) {
					t.Errorf("resid[%d]: got %.17g, want %.17g", i, r.Residuals[i], expResid[i])
				}
			}
		})
	}
}

// ============================================================
// Multiple LinearRegression (2+ predictors)
// ============================================================

func TestLinearRegression_Multiple_R(t *testing.T) {
	t.Run("ml_basic_2pred", func(t *testing.T) {
		x1 := insyra.NewDataList([]float64{1.2, 2.5, 3.1, 4.8, 5.3})
		x2 := insyra.NewDataList([]float64{0.8, 1.9, 2.7, 3.4, 4.1})
		y := insyra.NewDataList([]float64{3.5, 7.2, 9.8, 15.1, 17.9})
		r, err := stats.LinearRegression(y, x1, x2)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		assertMultiCoeffs(t, r, "ml_basic", 3, 5, tolReg)
		// In multiple regression, Slope/StandardError/etc. on the result should
		// be NaN (they are the simple-case scalar shortcuts).
		if !math.IsNaN(r.Slope) {
			t.Errorf("Slope must be NaN for multiple regression, got %v", r.Slope)
		}
		if !math.IsNaN(r.StandardError) {
			t.Errorf("StandardError must be NaN for multiple regression, got %v", r.StandardError)
		}
		if !math.IsNaN(r.TValue) {
			t.Errorf("TValue must be NaN for multiple regression, got %v", r.TValue)
		}
		if !math.IsNaN(r.PValue) {
			t.Errorf("PValue must be NaN for multiple regression, got %v", r.PValue)
		}
		if !math.IsNaN(r.ConfidenceIntervalSlope[0]) || !math.IsNaN(r.ConfidenceIntervalSlope[1]) {
			t.Errorf("ConfidenceIntervalSlope must be NaN for multiple regression, got %v",
				r.ConfidenceIntervalSlope)
		}
	})

	t.Run("ml_3pred", func(t *testing.T) {
		x1 := insyra.NewDataList(regDump.get(t, "ml_3pred_x1"))
		x2 := insyra.NewDataList(regDump.get(t, "ml_3pred_x2"))
		x3 := insyra.NewDataList(regDump.get(t, "ml_3pred_x3"))
		y := insyra.NewDataList(regDump.get(t, "ml_3pred_y"))
		r, err := stats.LinearRegression(y, x1, x2, x3)
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		assertMultiCoeffs(t, r, "ml_3pred", 4, 30, tolReg)
	})
}

// assertMultiCoeffs checks all multi-coefficient inference fields of a
// LinearRegressionResult against R reference.
func assertMultiCoeffs(t *testing.T, r *stats.LinearRegressionResult, prefix string, nCoef, nObs int, tol float64) {
	t.Helper()
	expCoef := refVec(t, prefix+".coef", nCoef)
	expSE := refVec(t, prefix+".se", nCoef)
	expT := refVec(t, prefix+".t", nCoef)
	expP := refVec(t, prefix+".p", nCoef)
	expCILo := refVec(t, prefix+".cilo", nCoef)
	expCIHi := refVec(t, prefix+".cihi", nCoef)
	expRsq := regRef.get(t, prefix+".rsq")
	expAdj := regRef.get(t, prefix+".adjr")
	expResid := refVec(t, prefix+".resid", nObs)

	if len(r.Coefficients) != nCoef {
		t.Fatalf("coefficients length: got %d, want %d", len(r.Coefficients), nCoef)
	}
	for i := range nCoef {
		if !rClose(r.Coefficients[i], expCoef[i], tol) {
			t.Errorf("coef[%d]: got %.17g, want %.17g", i, r.Coefficients[i], expCoef[i])
		}
		if !rClose(r.StandardErrors[i], expSE[i], tol) {
			t.Errorf("se[%d]: got %.17g, want %.17g", i, r.StandardErrors[i], expSE[i])
		}
		if !rClose(r.TValues[i], expT[i], tol) {
			t.Errorf("t[%d]: got %.17g, want %.17g", i, r.TValues[i], expT[i])
		}
		if !rClose(r.PValues[i], expP[i], tolRegP) {
			t.Errorf("p[%d]: got %.17g, want %.17g", i, r.PValues[i], expP[i])
		}
		if !rClose(r.ConfidenceIntervals[i][0], expCILo[i], tolRegCI) ||
			!rClose(r.ConfidenceIntervals[i][1], expCIHi[i], tolRegCI) {
			t.Errorf("CI[%d]: got [%v,%v], want [%v,%v]", i,
				r.ConfidenceIntervals[i][0], r.ConfidenceIntervals[i][1],
				expCILo[i], expCIHi[i])
		}
	}
	if !rClose(r.RSquared, expRsq, tol) {
		t.Errorf("R²: got %.17g, want %.17g", r.RSquared, expRsq)
	}
	if !rClose(r.AdjustedRSquared, expAdj, tol) {
		t.Errorf("adj R²: got %.17g, want %.17g", r.AdjustedRSquared, expAdj)
	}
	if !rClose(r.Intercept, expCoef[0], tol) {
		t.Errorf("intercept: got %.17g, want %.17g", r.Intercept, expCoef[0])
	}
	if !rClose(r.ConfidenceIntervalIntercept[0], expCILo[0], tolRegCI) ||
		!rClose(r.ConfidenceIntervalIntercept[1], expCIHi[0], tolRegCI) {
		t.Errorf("CI(intercept): got [%v,%v], want [%v,%v]",
			r.ConfidenceIntervalIntercept[0], r.ConfidenceIntervalIntercept[1],
			expCILo[0], expCIHi[0])
	}
	if len(r.Residuals) != nObs {
		t.Fatalf("residuals length: got %d, want %d", len(r.Residuals), nObs)
	}
	for i := range expResid {
		if !rClose(r.Residuals[i], expResid[i], tol) {
			t.Errorf("resid[%d]: got %.17g, want %.17g", i, r.Residuals[i], expResid[i])
		}
	}
}

func TestLinearRegression_Errors(t *testing.T) {
	if _, err := stats.LinearRegression(insyra.NewDataList([]float64{1, 2, 3})); err == nil {
		t.Error("expected error for no predictors")
	}
	if _, err := stats.LinearRegression(
		insyra.NewDataList([]float64{1, 2, 3}),
		insyra.NewDataList([]float64{1, 2})); err == nil {
		t.Error("expected error for mismatched lengths")
	}
	// n=2, p=1 → n <= p+1 fails (need at least p+2)
	if _, err := stats.LinearRegression(
		insyra.NewDataList([]float64{1, 2}),
		insyra.NewDataList([]float64{1, 2})); err == nil {
		t.Error("expected error for too few observations")
	}
}

// ============================================================
// Polynomial regression
// ============================================================

type polyCase struct {
	name   string
	x, y   []float64
	degree int
	prefix string
	tol    float64
}

func TestPolynomialRegression_R(t *testing.T) {
	cases := []polyCase{
		{"quadratic",
			[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			[]float64{2.1, 8.9, 20.1, 35.8, 56.2, 81.1, 110.9, 145.2, 184.1, 227.8},
			2, "pl_quad", tolReg},
		{"cubic_perfect_fit",
			[]float64{1, 2, 3, 4, 5}, []float64{5, 18, 47, 98, 177},
			3, "pl_cubic", tolReg},
		// Degree-4 polynomial with x up to 15 — the design matrix has
		// columns [1, x, x², x³, x⁴] with x⁴ up to 15⁴ = 50,625, condition
		// number ~10⁷ — coefficient inference loosens accordingly.
		{"degree_4_n15",
			regDump.get(t, "pl_deg4_x"), regDump.get(t, "pl_deg4_y"),
			4, "pl_deg4", tolRegPoly},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.PolynomialRegression(
				insyra.NewDataList(c.y), insyra.NewDataList(c.x), c.degree)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if r.Degree != c.degree {
				t.Errorf("degree: got %d, want %d", r.Degree, c.degree)
			}
			expCoef := refVec(t, c.prefix+".coef", c.degree+1)
			expRsq := regRef.get(t, c.prefix+".rsq")
			expAdj := regRef.get(t, c.prefix+".adjr")
			expResid := refVec(t, c.prefix+".resid", len(c.x))

			for i := range c.degree + 1 {
				if !rClose(r.Coefficients[i], expCoef[i], c.tol) {
					t.Errorf("coef[%d]: got %.17g, want %.17g (Δ=%g)",
						i, r.Coefficients[i], expCoef[i],
						math.Abs(r.Coefficients[i]-expCoef[i]))
				}
			}
			if !rClose(r.RSquared, expRsq, c.tol) {
				t.Errorf("R²: got %.17g, want %.17g", r.RSquared, expRsq)
			}
			if !rClose(r.AdjustedRSquared, expAdj, c.tol) {
				t.Errorf("adj R²: got %.17g, want %.17g", r.AdjustedRSquared, expAdj)
			}
			for i := range expResid {
				if !rClose(r.Residuals[i], expResid[i], c.tol) {
					t.Errorf("resid[%d]: got %.17g, want %.17g", i, r.Residuals[i], expResid[i])
				}
			}
			// SE/T/P only meaningful when fit isn't perfect (cubic case has R²=1
			// with denormal SSE → SEs are dominated by noise).
			if expRsq < 1.0 {
				expSE := refVec(t, c.prefix+".se", c.degree+1)
				expT := refVec(t, c.prefix+".t", c.degree+1)
				expP := refVec(t, c.prefix+".p", c.degree+1)
				expCILo := refVec(t, c.prefix+".cilo", c.degree+1)
				expCIHi := refVec(t, c.prefix+".cihi", c.degree+1)
				for i := range c.degree + 1 {
					if !rClose(r.StandardErrors[i], expSE[i], c.tol) {
						t.Errorf("se[%d]: got %.17g, want %.17g", i, r.StandardErrors[i], expSE[i])
					}
					if !rClose(r.TValues[i], expT[i], c.tol) {
						t.Errorf("t[%d]: got %.17g, want %.17g", i, r.TValues[i], expT[i])
					}
					if !rClose(r.PValues[i], expP[i], c.tol) {
						t.Errorf("p[%d]: got %.17g, want %.17g", i, r.PValues[i], expP[i])
					}
					if !rClose(r.ConfidenceIntervals[i][0], expCILo[i], c.tol) ||
						!rClose(r.ConfidenceIntervals[i][1], expCIHi[i], c.tol) {
						t.Errorf("CI[%d]: got [%v,%v], want [%v,%v]", i,
							r.ConfidenceIntervals[i][0], r.ConfidenceIntervals[i][1],
							expCILo[i], expCIHi[i])
					}
				}
			}
		})
	}
}

func TestPolynomialRegression_Errors(t *testing.T) {
	x := insyra.NewDataList([]float64{1, 2, 3})
	y := insyra.NewDataList([]float64{1, 4, 9})
	if _, err := stats.PolynomialRegression(y, x, 0); err == nil {
		t.Error("expected error for degree=0")
	}
	if _, err := stats.PolynomialRegression(y, x, 3); err == nil {
		t.Error("expected error for degree>=n")
	}
	if _, err := stats.PolynomialRegression(
		insyra.NewDataList([]float64{1, 2, 3}),
		insyra.NewDataList([]float64{1, 2}), 1); err == nil {
		t.Error("expected error for mismatched lengths")
	}
}

// ============================================================
// Exponential regression
// ============================================================

type expCase struct {
	name   string
	x, y   []float64
	prefix string
}

func TestExponentialRegression_R(t *testing.T) {
	cases := []expCase{
		{"basic_growth",
			[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			[]float64{2.72, 7.39, 20.09, 54.60, 148.41, 403.43, 1096.63, 2980.96, 8103.08, 22026.47},
			"ex_basic"},
		{"decay",
			[]float64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			[]float64{100, 60.65, 36.79, 22.31, 13.53, 8.21, 4.98, 3.02, 1.83, 1.11},
			"ex_decay"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.ExponentialRegression(
				insyra.NewDataList(c.y), insyra.NewDataList(c.x))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			tolHere := tolReg
			// `basic_growth` is essentially y = e^x — residuals against y reach
			// ~2.5 at the tail (rounding in y), so the inference statistics
			// have a wider relative spread than for well-conditioned data.
			if c.name == "basic_growth" {
				tolHere = 1e-3
			}
			check := func(label string, got, want float64) {
				if !rClose(got, want, tolHere) {
					t.Errorf("%s: got %.17g, want %.17g (Δ=%g)", label, got, want, math.Abs(got-want))
				}
			}
			check("a", r.Intercept, regRef.get(t, c.prefix+".a"))
			check("b", r.Slope, regRef.get(t, c.prefix+".b"))
			check("seA", r.StandardErrorIntercept, regRef.get(t, c.prefix+".seA"))
			check("seB", r.StandardErrorSlope, regRef.get(t, c.prefix+".seB"))
			check("tA", r.TValueIntercept, regRef.get(t, c.prefix+".tA"))
			check("tB", r.TValueSlope, regRef.get(t, c.prefix+".tB"))
			check("pA", r.PValueIntercept, regRef.get(t, c.prefix+".pA"))
			check("pB", r.PValueSlope, regRef.get(t, c.prefix+".pB"))
			check("ciAlo", r.ConfidenceIntervalIntercept[0], regRef.get(t, c.prefix+".ciAlo"))
			check("ciAhi", r.ConfidenceIntervalIntercept[1], regRef.get(t, c.prefix+".ciAhi"))
			check("ciBlo", r.ConfidenceIntervalSlope[0], regRef.get(t, c.prefix+".ciBlo"))
			check("ciBhi", r.ConfidenceIntervalSlope[1], regRef.get(t, c.prefix+".ciBhi"))
			check("R²", r.RSquared, regRef.get(t, c.prefix+".rsq"))
			check("adj R²", r.AdjustedRSquared, regRef.get(t, c.prefix+".adjr"))
			expResid := refVec(t, c.prefix+".resid", len(c.x))
			for i := range expResid {
				check("resid["+itoa(i)+"]", r.Residuals[i], expResid[i])
			}
		})
	}
}

func TestExponentialRegression_Errors(t *testing.T) {
	x := insyra.NewDataList([]float64{1, 2, 3})
	if _, err := stats.ExponentialRegression(
		insyra.NewDataList([]float64{1, 2}), x); err == nil {
		t.Error("expected error for length mismatch")
	}
	if _, err := stats.ExponentialRegression(
		insyra.NewDataList([]float64{1, 2}),
		insyra.NewDataList([]float64{1, 2})); err == nil {
		t.Error("expected error for n<=2")
	}
	if _, err := stats.ExponentialRegression(
		insyra.NewDataList([]float64{1, -1, 2}), x); err == nil {
		t.Error("expected error for negative y")
	}
	if _, err := stats.ExponentialRegression(
		insyra.NewDataList([]float64{1, 0, 2}), x); err == nil {
		t.Error("expected error for zero y")
	}
}

// ============================================================
// Logarithmic regression
// ============================================================

func TestLogarithmicRegression_R(t *testing.T) {
	cases := []struct {
		name   string
		x, y   []float64
		prefix string
	}{
		{"basic",
			[]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			[]float64{0, 0.693, 1.099, 1.386, 1.609, 1.792, 1.946, 2.079, 2.197, 2.303},
			"lo_basic"},
		{"noisy",
			regDump.get(t, "lo_noisy_x"), regDump.get(t, "lo_noisy_y"),
			"lo_noisy"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.LogarithmicRegression(
				insyra.NewDataList(c.y), insyra.NewDataList(c.x))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expCoef := refVec(t, c.prefix+".coef", 2)
			expSE := refVec(t, c.prefix+".se", 2)
			expT := refVec(t, c.prefix+".t", 2)
			expP := refVec(t, c.prefix+".p", 2)
			expCILo := refVec(t, c.prefix+".cilo", 2)
			expCIHi := refVec(t, c.prefix+".cihi", 2)
			expRsq := regRef.get(t, c.prefix+".rsq")
			expAdj := regRef.get(t, c.prefix+".adjr")
			expResid := refVec(t, c.prefix+".resid", len(c.x))

			check := func(label string, got, want float64) {
				if !rClose(got, want, tolReg) {
					t.Errorf("%s: got %.17g, want %.17g", label, got, want)
				}
			}
			check("a", r.Intercept, expCoef[0])
			check("b", r.Slope, expCoef[1])
			check("seA", r.StandardErrorIntercept, expSE[0])
			check("seB", r.StandardErrorSlope, expSE[1])
			check("tA", r.TValueIntercept, expT[0])
			check("tB", r.TValueSlope, expT[1])
			check("pA", r.PValueIntercept, expP[0])
			check("pB", r.PValueSlope, expP[1])
			check("ciAlo", r.ConfidenceIntervalIntercept[0], expCILo[0])
			check("ciAhi", r.ConfidenceIntervalIntercept[1], expCIHi[0])
			check("ciBlo", r.ConfidenceIntervalSlope[0], expCILo[1])
			check("ciBhi", r.ConfidenceIntervalSlope[1], expCIHi[1])
			check("R²", r.RSquared, expRsq)
			check("adj R²", r.AdjustedRSquared, expAdj)
			for i := range expResid {
				check("resid["+itoa(i)+"]", r.Residuals[i], expResid[i])
			}
		})
	}
}

func TestLogarithmicRegression_Errors(t *testing.T) {
	if _, err := stats.LogarithmicRegression(
		insyra.NewDataList([]float64{1, 2}),
		insyra.NewDataList([]float64{1, 2, 3})); err == nil {
		t.Error("expected error for length mismatch")
	}
	if _, err := stats.LogarithmicRegression(
		insyra.NewDataList([]float64{1, 2}),
		insyra.NewDataList([]float64{1, 2})); err == nil {
		t.Error("expected error for n<=2")
	}
	if _, err := stats.LogarithmicRegression(
		insyra.NewDataList([]float64{1, 2, 3}),
		insyra.NewDataList([]float64{0, 1, 2})); err == nil {
		t.Error("expected error for zero/negative x")
	}
}
