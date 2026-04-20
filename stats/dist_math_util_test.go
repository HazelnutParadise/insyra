package stats

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/stat/distuv"
)

func TestDistUtilsAgainstDistuv(t *testing.T) {
	t.Run("t utils", func(t *testing.T) {
		df := 12.0
		tVal := 1.732
		d := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}

		wantCDF := d.CDF(tVal)
		if !distMathAlmostEqual(tCDF(tVal, df), wantCDF, 1e-12) {
			t.Fatalf("tCDF mismatch: got %v want %v", tCDF(tVal, df), wantCDF)
		}

		wantP := 2 * (1 - d.CDF(math.Abs(tVal)))
		if !distMathAlmostEqual(tTwoTailedPValue(tVal, df), wantP, 1e-12) {
			t.Fatalf("tTwoTailedPValue mismatch: got %v want %v", tTwoTailedPValue(tVal, df), wantP)
		}

		p := 0.975
		wantQ := d.Quantile(p)
		if !distMathAlmostEqual(tQuantile(p, df), wantQ, 1e-12) {
			t.Fatalf("tQuantile mismatch: got %v want %v", tQuantile(p, df), wantQ)
		}
	})

	t.Run("f utils", func(t *testing.T) {
		f := 2.41
		df1, df2 := 4.0, 18.0
		d := distuv.F{D1: df1, D2: df2}

		wantOneTail := 1 - d.CDF(f)
		if !distMathAlmostEqual(fOneTailedPValue(f, df1, df2), wantOneTail, 1e-12) {
			t.Fatalf("fOneTailedPValue mismatch: got %v want %v", fOneTailedPValue(f, df1, df2), wantOneTail)
		}

		wantTwoTail := 2 * math.Min(d.CDF(f), 1-d.CDF(f))
		if !distMathAlmostEqual(fTwoTailedPValue(f, df1, df2), wantTwoTail, 1e-12) {
			t.Fatalf("fTwoTailedPValue mismatch: got %v want %v", fTwoTailedPValue(f, df1, df2), wantTwoTail)
		}
	})

	t.Run("chi-square utils", func(t *testing.T) {
		chi2 := 9.87
		df := 5.0
		d := distuv.ChiSquared{K: df}
		want := 1 - d.CDF(chi2)
		if !distMathAlmostEqual(chiSquaredPValue(chi2, df), want, 1e-12) {
			t.Fatalf("chiSquaredPValue mismatch: got %v want %v", chiSquaredPValue(chi2, df), want)
		}
	})

	t.Run("z utils", func(t *testing.T) {
		z := -1.23
		wantCDF := norm.CDF(z)
		if !distMathAlmostEqual(zCDF(z), wantCDF, 1e-12) {
			t.Fatalf("zCDF mismatch: got %v want %v", zCDF(z), wantCDF)
		}

		wantTwoSided := 2 * (1 - norm.CDF(math.Abs(z)))
		if !distMathAlmostEqual(zPValue(z, TwoSided), wantTwoSided, 1e-12) {
			t.Fatalf("zPValue two-sided mismatch: got %v want %v", zPValue(z, TwoSided), wantTwoSided)
		}
	})
}

func TestMathUtilsBasics(t *testing.T) {
	if got := resolveConfidenceLevel(0); got != defaultConfidenceLevel {
		t.Fatalf("resolveConfidenceLevel invalid mismatch: got %v want %v", got, defaultConfidenceLevel)
	}
	if got := resolveConfidenceLevel(0.99); got != 0.99 {
		t.Fatalf("resolveConfidenceLevel valid mismatch: got %v want 0.99", got)
	}

	ci := symmetricCI(10, 2.5)
	if !distMathAlmostEqual(ci[0], 7.5, 0) || !distMathAlmostEqual(ci[1], 12.5, 0) {
		t.Fatalf("symmetricCI mismatch: got %v", ci)
	}

	if !math.IsNaN(nanCI()[0]) || !math.IsNaN(nanCI()[1]) {
		t.Fatalf("nanCI mismatch: got %v", nanCI())
	}

	df := 20.0
	se := 1.2
	wantTMargin := tQuantile(1-(1-0.95)/2, df) * se
	if !distMathAlmostEqual(tMarginOfError(0.95, df, se), wantTMargin, 1e-12) {
		t.Fatalf("tMarginOfError mismatch: got %v want %v", tMarginOfError(0.95, df, se), wantTMargin)
	}

	wantZMargin := norm.Quantile(1-(1-0.95)/2) * se
	if !distMathAlmostEqual(zMarginOfError(0.95, se), wantZMargin, 1e-12) {
		t.Fatalf("zMarginOfError mismatch: got %v want %v", zMarginOfError(0.95, se), wantZMargin)
	}

	if got := cohenDEffectSizes(0.4); len(got) != 1 || got[0].Type != "cohen_d" || !distMathAlmostEqual(got[0].Value, 0.4, 0) {
		t.Fatalf("cohenDEffectSizes mismatch: got %+v", got)
	}

	if !distMathAlmostEqual(etaSquared(30, 70), 0.3, 1e-12) {
		t.Fatalf("etaSquared mismatch")
	}

	if !distMathAlmostEqual(fRatio(30, 3, 70, 7), 1.0, 1e-12) {
		t.Fatalf("fRatio mismatch")
	}

	r := 0.62
	n := 25.0
	wantT := r * math.Sqrt(n-2) / math.Sqrt(1-r*r)
	if !distMathAlmostEqual(correlationToT(r, n), wantT, 1e-12) {
		t.Fatalf("correlationToT mismatch: got %v want %v", correlationToT(r, n), wantT)
	}

	z := fisherZTransform(r)
	if !distMathAlmostEqual(fisherZInverse(z), r, 1e-12) {
		t.Fatalf("fisher transform inverse mismatch: got %v want %v", fisherZInverse(z), r)
	}

	ci2 := pearsonFisherCI(r, n, 0.95)
	zCritical := norm.Quantile(1 - (1-0.95)/2)
	seZ := 1 / math.Sqrt(n-3)
	wantLower := fisherZInverse(z - zCritical*seZ)
	wantUpper := fisherZInverse(z + zCritical*seZ)
	if !distMathAlmostEqual(ci2[0], wantLower, 1e-12) || !distMathAlmostEqual(ci2[1], wantUpper, 1e-12) {
		t.Fatalf("pearsonFisherCI mismatch: got %v want [%v %v]", ci2, wantLower, wantUpper)
	}
}

func distMathAlmostEqual(a, b, tol float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.IsNaN(a) && math.IsNaN(b)
	}
	return math.Abs(a-b) <= tol
}
