package stats

import (
	"math"
	"testing"
)

// R baseline values in this file are generated with:
//   Rscript tmp_r_primitives.R
// where tmp_r_primitives.R uses pt/qt/pf/pchisq/pnorm/lm/solve/det on fixed inputs.

func TestDistPrimitivesAgainstR(t *testing.T) {
	const tol = 1e-12

	const (
		tVal = 2.123
		tDF  = 14.5
		tP   = 0.975

		fVal = 3.25
		df1  = 5.0
		df2  = 17.0

		chiVal = 9.87
		chiDF  = 4.0
		zVal   = -1.2345
	)

	if !rAlmostEqual(tCDF(tVal, tDF), 0.97429348438019814, tol) {
		t.Fatalf("tCDF mismatch")
	}
	if !rAlmostEqual(tTwoTailedPValue(tVal, tDF), 0.051413031239603724, tol) {
		t.Fatalf("tTwoTailedPValue mismatch")
	}
	if !rAlmostEqual(tQuantile(tP, tDF), 2.1378689157851158, tol) {
		t.Fatalf("tQuantile mismatch")
	}

	if !rAlmostEqual(fOneTailedPValue(fVal, df1, df2), 0.030621010467787091, tol) {
		t.Fatalf("fOneTailedPValue mismatch")
	}
	if !rAlmostEqual(fTwoTailedPValue(fVal, df1, df2), 0.061242020935574182, tol) {
		t.Fatalf("fTwoTailedPValue mismatch")
	}

	if !rAlmostEqual(chiSquaredPValue(chiVal, chiDF), 0.042675385713895397, tol) {
		t.Fatalf("chiSquaredPValue mismatch")
	}

	if !rAlmostEqual(zCDF(zVal), 0.10850832336267018, tol) {
		t.Fatalf("zCDF mismatch")
	}
	if !rAlmostEqual(zPValue(zVal, TwoSided), 0.21701664672534027, tol) {
		t.Fatalf("zPValue TwoSided mismatch")
	}
	if !rAlmostEqual(zPValue(zVal, Greater), 0.89149167663732987, tol) {
		t.Fatalf("zPValue Greater mismatch")
	}
	if !rAlmostEqual(zPValue(zVal, Less), 0.10850832336267018, tol) {
		t.Fatalf("zPValue Less mismatch")
	}
}

func TestSamplePrimitivesAgainstR(t *testing.T) {
	const tol = 1e-12

	if !rAlmostEqual(sampleSE(2.5, 16), 0.625, tol) {
		t.Fatalf("sampleSE mismatch")
	}

	var1, var2 := 4.2, 7.8
	n1, n2 := 12.0, 15.0

	if !rAlmostEqual(pooledVariance(var1, var2, n1, n2), 6.2160000000000002, tol) {
		t.Fatalf("pooledVariance mismatch")
	}
	se, pvar := pooledSE(var1, var2, n1, n2)
	if !rAlmostEqual(pvar, 6.2160000000000002, tol) {
		t.Fatalf("pooledSE pooled variance mismatch")
	}
	if !rAlmostEqual(se, 0.96560861636586492, tol) {
		t.Fatalf("pooledSE mismatch")
	}
	if !rAlmostEqual(welchDF(var1, var2, n1, n2), 24.856612786283964, tol) {
		t.Fatalf("welchDF mismatch")
	}
	if !rAlmostEqual(twoSampleSE(var1, var2, n1, n2), 0.93273790530888157, tol) {
		t.Fatalf("twoSampleSE mismatch")
	}
}

func TestMathPrimitivesAgainstR(t *testing.T) {
	const tol = 1e-12

	if resolveConfidenceLevel(0) != defaultConfidenceLevel {
		t.Fatalf("resolveConfidenceLevel invalid mismatch")
	}
	if resolveConfidenceLevel(0.9) != 0.9 {
		t.Fatalf("resolveConfidenceLevel valid mismatch")
	}

	ci := symmetricCI(10, 2.5)
	if !rAlmostEqual(ci[0], 7.5, tol) || !rAlmostEqual(ci[1], 12.5, tol) {
		t.Fatalf("symmetricCI mismatch")
	}
	nci := nanCI()
	if !math.IsNaN(nci[0]) || !math.IsNaN(nci[1]) {
		t.Fatalf("nanCI mismatch")
	}

	if !rAlmostEqual(tMarginOfError(0.9, 10, 1.2), 2.1749533473740108, tol) {
		t.Fatalf("tMarginOfError mismatch")
	}
	if !rAlmostEqual(zMarginOfError(0.9, 1.2), 1.9738243523417658, tol) {
		t.Fatalf("zMarginOfError mismatch")
	}

	es := cohenDEffectSizes(0.3)
	if len(es) != 1 || es[0].Type != "cohen_d" || !rAlmostEqual(es[0].Value, 0.3, tol) {
		t.Fatalf("cohenDEffectSizes mismatch")
	}

	if !rAlmostEqual(etaSquared(23.4, 50.6), 0.31621621621621621, tol) {
		t.Fatalf("etaSquared mismatch")
	}
	if !rAlmostEqual(fRatio(45, 3, 30, 20), 10.0, tol) {
		t.Fatalf("fRatio mismatch")
	}
	if !rAlmostEqual(correlationToT(0.56, 25), 3.2416289898997559, tol) {
		t.Fatalf("correlationToT mismatch")
	}
	if !rAlmostEqual(fisherZTransform(0.56), 0.63283318666563804, tol) {
		t.Fatalf("fisherZTransform mismatch")
	}
	if !rAlmostEqual(fisherZInverse(0.63283318666563804), 0.56000000000000005, tol) {
		t.Fatalf("fisherZInverse mismatch")
	}

	fci := pearsonFisherCI(0.56, 25, 0.95)
	if !rAlmostEqual(fci[0], 0.2117162567230573, tol) || !rAlmostEqual(fci[1], 0.7820779314329579, tol) {
		t.Fatalf("pearsonFisherCI mismatch")
	}
}

func TestOLSPrimitivesAgainstR(t *testing.T) {
	const tol = 1e-12

	xs := []float64{1.2, 2.5, 3.1, 4.8, 5.3, 6.0}
	ys := []float64{3.2, 5.7, 7.0, 9.8, 11.1, 12.4}

	intercept, slope, ok := simpleOLSCoeffs(xs, ys)
	if !ok {
		t.Fatalf("simpleOLSCoeffs returned !ok")
	}
	if !rAlmostEqual(intercept, 0.96488205931290694, tol) {
		t.Fatalf("simpleOLSCoeffs intercept mismatch")
	}
	if !rAlmostEqual(slope, 1.8956640892629937, tol) {
		t.Fatalf("simpleOLSCoeffs slope mismatch")
	}

	if !rAlmostEqual(simpleOLSSxx(xs), 17.028333333333332, tol) {
		t.Fatalf("simpleOLSSxx mismatch")
	}

	yhat := []float64{
		0.96488205931290694 + 1.8956640892629937*1.2,
		0.96488205931290694 + 1.8956640892629937*2.5,
		0.96488205931290694 + 1.8956640892629937*3.1,
		0.96488205931290694 + 1.8956640892629937*4.8,
		0.96488205931290694 + 1.8956640892629937*5.3,
		0.96488205931290694 + 1.8956640892629937*6.0,
	}
	residuals, r2, adjR2, sse, fitOK := computeGoodnessOfFit(ys, func(i int) float64 { return yhat[i] }, 4)
	if !fitOK {
		t.Fatalf("computeGoodnessOfFit returned !ok")
	}
	if !rAlmostEqual(sse, 0.10796319859058381, tol) {
		t.Fatalf("computeGoodnessOfFit SSE mismatch")
	}
	if !rAlmostEqual(r2, 0.99823877326932164, tol) {
		t.Fatalf("computeGoodnessOfFit R2 mismatch")
	}
	if !rAlmostEqual(adjR2, 0.99779846658665206, tol) {
		t.Fatalf("computeGoodnessOfFit adjR2 mismatch")
	}
	wantResiduals := []float64{
		-0.0396789664284989,
		-0.00404228247039118,
		0.158559263971813,
		-0.264069687775276,
		0.0880982675932263,
		0.0611334051091301,
	}
	if len(residuals) != len(wantResiduals) {
		t.Fatalf("residual length mismatch")
	}
	for i := range residuals {
		if !rAlmostEqual(residuals[i], wantResiduals[i], tol) {
			t.Fatalf("residual[%d] mismatch: got %v want %v", i, residuals[i], wantResiduals[i])
		}
	}

	coeffs := []float64{0.96488205931290694, 1.8956640892629937}
	xtxInv := [][]float64{
		{1.02211999608496, -0.224136243515709},
		{-0.224136243515709, 0.0587256533228932},
	}
	mse := 0.026990799647645954
	se, tv, pv := computeCoeffInference(coeffs, xtxInv, mse, 4)
	if !rAlmostEqual(se[0], 0.166095863976746, tol) || !rAlmostEqual(se[1], 0.0398127158457612, tol) {
		t.Fatalf("computeCoeffInference SE mismatch")
	}
	if !rAlmostEqual(tv[0], 5.80918775586124, tol) || !rAlmostEqual(tv[1], 47.6145384456314, tol) {
		t.Fatalf("computeCoeffInference t mismatch")
	}
	if !rAlmostEqual(pv[0], 0.00436938108954035, tol) || !rAlmostEqual(pv[1], 1.16390342412039e-06, tol) {
		t.Fatalf("computeCoeffInference p mismatch")
	}

	ciA, ciB := buildTwoCoeffCIs(coeffs[0], coeffs[1], se[0], se[1], 4)
	if !rAlmostEqual(ciA[0], 0.503726010781071, tol) || !rAlmostEqual(ciA[1], 1.42603810784474, tol) {
		t.Fatalf("buildTwoCoeffCIs intercept mismatch")
	}
	if !rAlmostEqual(ciB[0], 1.7851262692284, tol) || !rAlmostEqual(ciB[1], 2.00620190929759, tol) {
		t.Fatalf("buildTwoCoeffCIs slope mismatch")
	}

	allCI := buildMultiCoeffCIs(coeffs, se, 4)
	if !rAlmostEqual(allCI[0][0], 0.503726010781071, tol) || !rAlmostEqual(allCI[0][1], 1.42603810784474, tol) {
		t.Fatalf("buildMultiCoeffCIs intercept mismatch")
	}
	if !rAlmostEqual(allCI[1][0], 1.7851262692284, tol) || !rAlmostEqual(allCI[1][1], 2.00620190929759, tol) {
		t.Fatalf("buildMultiCoeffCIs slope mismatch")
	}
}

func rAlmostEqual(got, want, tol float64) bool {
	return math.Abs(got-want) <= tol
}
