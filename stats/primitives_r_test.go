package stats

import (
	"math"
	"testing"
)

// R baseline values in this file are generated with:
//   Rscript tmp_r_primitives.R
// using fixed inputs and base R functions (pt/qt/pf/pchisq/pnorm/lm/confint).

type scalarPrimitiveCase struct {
	primitive string
	rFormula  string
	eval      func() float64
	want      float64
	tol       float64
}

type vectorPrimitiveCase struct {
	primitive string
	rFormula  string
	eval      func() []float64
	want      []float64
	tol       float64
}

type rulePrimitiveCase struct {
	primitive string
	rFormula  string
	eval      func() bool
}

type effectSizePrimitiveCase struct {
	primitive string
	rFormula  string
	eval      func() []EffectSizeEntry
	wantType  string
	want      float64
	tol       float64
}

func runScalarPrimitiveCases(t *testing.T, cases []scalarPrimitiveCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.primitive, func(t *testing.T) {
			got := tc.eval()
			if !rAlmostEqual(got, tc.want, tc.tol) {
				t.Fatalf("primitive=%s got=%.17g want=%.17g r_formula=%q", tc.primitive, got, tc.want, tc.rFormula)
			}
		})
	}
}

func runVectorPrimitiveCases(t *testing.T, cases []vectorPrimitiveCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.primitive, func(t *testing.T) {
			got := tc.eval()
			if len(got) != len(tc.want) {
				t.Fatalf("primitive=%s got_len=%d want_len=%d r_formula=%q", tc.primitive, len(got), len(tc.want), tc.rFormula)
			}
			for i := range got {
				if !rAlmostEqual(got[i], tc.want[i], tc.tol) {
					t.Fatalf("primitive=%s idx=%d got=%.17g want=%.17g r_formula=%q", tc.primitive, i, got[i], tc.want[i], tc.rFormula)
				}
			}
		})
	}
}

func runRulePrimitiveCases(t *testing.T, cases []rulePrimitiveCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.primitive, func(t *testing.T) {
			if !tc.eval() {
				t.Fatalf("primitive=%s failed r_formula=%q", tc.primitive, tc.rFormula)
			}
		})
	}
}

func runEffectSizePrimitiveCases(t *testing.T, cases []effectSizePrimitiveCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.primitive, func(t *testing.T) {
			got := tc.eval()
			if len(got) != 1 {
				t.Fatalf("primitive=%s expected single effect size got=%d r_formula=%q", tc.primitive, len(got), tc.rFormula)
			}
			if got[0].Type != tc.wantType {
				t.Fatalf("primitive=%s type=%q want=%q r_formula=%q", tc.primitive, got[0].Type, tc.wantType, tc.rFormula)
			}
			if !rAlmostEqual(got[0].Value, tc.want, tc.tol) {
				t.Fatalf("primitive=%s value=%.17g want=%.17g r_formula=%q", tc.primitive, got[0].Value, tc.want, tc.rFormula)
			}
		})
	}
}

func TestDistPrimitivesAgainstR(t *testing.T) {
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

	runScalarPrimitiveCases(t, []scalarPrimitiveCase{
		{primitive: "tCDF", rFormula: "pt(2.123, 14.5)", eval: func() float64 { return tCDF(tVal, tDF) }, want: 0.97429348438019814, tol: 1e-12},
		{primitive: "tTwoTailedPValue", rFormula: "2*(1-pt(abs(2.123), 14.5))", eval: func() float64 { return tTwoTailedPValue(tVal, tDF) }, want: 0.051413031239603724, tol: 1e-12},
		{primitive: "tQuantile", rFormula: "qt(0.975, 14.5)", eval: func() float64 { return tQuantile(tP, tDF) }, want: 2.1378689157851158, tol: 1e-12},
		{primitive: "fOneTailedPValue", rFormula: "1-pf(3.25, 5, 17)", eval: func() float64 { return fOneTailedPValue(fVal, df1, df2) }, want: 0.030621010467787091, tol: 1e-12},
		{primitive: "fTwoTailedPValue", rFormula: "2*min(pf(3.25,5,17), 1-pf(3.25,5,17))", eval: func() float64 { return fTwoTailedPValue(fVal, df1, df2) }, want: 0.061242020935574182, tol: 1e-12},
		{primitive: "chiSquaredPValue", rFormula: "1-pchisq(9.87, 4)", eval: func() float64 { return chiSquaredPValue(chiVal, chiDF) }, want: 0.042675385713895397, tol: 1e-12},
		{primitive: "zCDF", rFormula: "pnorm(-1.2345)", eval: func() float64 { return zCDF(zVal) }, want: 0.10850832336267018, tol: 1e-12},
		{primitive: "zPValue_two_sided", rFormula: "2*(1-pnorm(abs(-1.2345)))", eval: func() float64 { return zPValue(zVal, TwoSided) }, want: 0.21701664672534027, tol: 1e-12},
		{primitive: "zPValue_greater", rFormula: "1-pnorm(-1.2345)", eval: func() float64 { return zPValue(zVal, Greater) }, want: 0.89149167663732987, tol: 1e-12},
		{primitive: "zPValue_less", rFormula: "pnorm(-1.2345)", eval: func() float64 { return zPValue(zVal, Less) }, want: 0.10850832336267018, tol: 1e-12},
	})
}

func TestSamplePrimitivesAgainstR(t *testing.T) {
	var1, var2 := 4.2, 7.8
	n1, n2 := 12.0, 15.0

	runScalarPrimitiveCases(t, []scalarPrimitiveCase{
		{primitive: "sampleSE", rFormula: "2.5/sqrt(16)", eval: func() float64 { return sampleSE(2.5, 16) }, want: 0.625, tol: 1e-12},
		{primitive: "pooledVariance", rFormula: "((12-1)*4.2 + (15-1)*7.8)/(12+15-2)", eval: func() float64 { return pooledVariance(var1, var2, n1, n2) }, want: 6.2160000000000002, tol: 1e-12},
		{primitive: "pooledSE", rFormula: "sqrt((((12-1)*4.2 + (15-1)*7.8)/(12+15-2))*(1/12 + 1/15))", eval: func() float64 { se, _ := pooledSE(var1, var2, n1, n2); return se }, want: 0.96560861636586492, tol: 1e-12},
		{primitive: "pooledSE_returned_variance", rFormula: "((12-1)*4.2 + (15-1)*7.8)/(12+15-2)", eval: func() float64 { _, pvar := pooledSE(var1, var2, n1, n2); return pvar }, want: 6.2160000000000002, tol: 1e-12},
		{primitive: "welchDF", rFormula: "((4.2/12 + 7.8/15)^2)/(((4.2/12)^2)/(12-1)+((7.8/15)^2)/(15-1))", eval: func() float64 { return welchDF(var1, var2, n1, n2) }, want: 24.856612786283964, tol: 1e-12},
		{primitive: "twoSampleSE", rFormula: "sqrt(4.2/12 + 7.8/15)", eval: func() float64 { return twoSampleSE(var1, var2, n1, n2) }, want: 0.93273790530888157, tol: 1e-12},
	})
}

func TestMathPrimitivesAgainstR(t *testing.T) {
	runRulePrimitiveCases(t, []rulePrimitiveCase{
		{
			primitive: "resolveConfidenceLevel_invalid_returns_default",
			rFormula:  "if (cl <= 0 || cl >= 1) 0.95 else cl; cl=0",
			eval:      func() bool { return resolveConfidenceLevel(0) == defaultConfidenceLevel },
		},
		{
			primitive: "resolveConfidenceLevel_valid_passthrough",
			rFormula:  "if (cl <= 0 || cl >= 1) 0.95 else cl; cl=0.9",
			eval:      func() bool { return resolveConfidenceLevel(0.9) == 0.9 },
		},
	})

	runVectorPrimitiveCases(t, []vectorPrimitiveCase{
		{
			primitive: "symmetricCI",
			rFormula:  "c(10-2.5, 10+2.5)",
			eval: func() []float64 {
				ci := symmetricCI(10, 2.5)
				return []float64{ci[0], ci[1]}
			},
			want: []float64{7.5, 12.5},
			tol:  1e-12,
		},
		{
			primitive: "pearsonFisherCI",
			rFormula:  "z <- atanh(0.56); se <- 1/sqrt(25-3); c(tanh(z-qnorm(0.975)*se), tanh(z+qnorm(0.975)*se))",
			eval: func() []float64 {
				ci := pearsonFisherCI(0.56, 25, 0.95)
				return []float64{ci[0], ci[1]}
			},
			want: []float64{0.2117162567230573, 0.7820779314329579},
			tol:  1e-12,
		},
	})

	runRulePrimitiveCases(t, []rulePrimitiveCase{
		{
			primitive: "nanCI",
			rFormula:  "c(NaN, NaN)",
			eval: func() bool {
				ci := nanCI()
				return math.IsNaN(ci[0]) && math.IsNaN(ci[1])
			},
		},
	})

	runEffectSizePrimitiveCases(t, []effectSizePrimitiveCase{
		{
			primitive: "cohenDEffectSizes",
			rFormula:  "list(list(Type='cohen_d', Value=0.3))",
			eval:      func() []EffectSizeEntry { return cohenDEffectSizes(0.3) },
			wantType:  "cohen_d",
			want:      0.3,
			tol:       1e-12,
		},
	})

	runScalarPrimitiveCases(t, []scalarPrimitiveCase{
		{primitive: "tMarginOfError", rFormula: "qt(1-(1-0.9)/2, 10)*1.2", eval: func() float64 { return tMarginOfError(0.9, 10, 1.2) }, want: 2.1749533473740108, tol: 1e-12},
		{primitive: "zMarginOfError", rFormula: "qnorm(1-(1-0.9)/2)*1.2", eval: func() float64 { return zMarginOfError(0.9, 1.2) }, want: 1.9738243523417658, tol: 1e-12},
		{primitive: "etaSquared", rFormula: "23.4/(23.4+50.6)", eval: func() float64 { return etaSquared(23.4, 50.6) }, want: 0.31621621621621621, tol: 1e-12},
		{primitive: "fRatio", rFormula: "(45/3)/(30/20)", eval: func() float64 { return fRatio(45, 3, 30, 20) }, want: 10.0, tol: 1e-12},
		{primitive: "correlationToT", rFormula: "0.56*sqrt(25-2)/sqrt(1-0.56^2)", eval: func() float64 { return correlationToT(0.56, 25) }, want: 3.2416289898997559, tol: 1e-12},
		{primitive: "fisherZTransform", rFormula: "0.5*log((1+0.56)/(1-0.56))", eval: func() float64 { return fisherZTransform(0.56) }, want: 0.63283318666563804, tol: 1e-12},
		{primitive: "fisherZInverse", rFormula: "(exp(2*0.63283318666563804)-1)/(exp(2*0.63283318666563804)+1)", eval: func() float64 { return fisherZInverse(0.63283318666563804) }, want: 0.56000000000000005, tol: 1e-12},
	})
}

func TestOLSPrimitivesAgainstR(t *testing.T) {
	xs := []float64{1.2, 2.5, 3.1, 4.8, 5.3, 6.0}
	ys := []float64{3.2, 5.7, 7.0, 9.8, 11.1, 12.4}

	intercept, slope, coeffOK := simpleOLSCoeffs(xs, ys)
	if !coeffOK {
		t.Fatalf("simpleOLSCoeffs failed")
	}

	yhat := make([]float64, len(xs))
	for i := range xs {
		yhat[i] = intercept + slope*xs[i]
	}
	residuals, r2, adjR2, sse, fitOK := computeGoodnessOfFit(ys, func(i int) float64 { return yhat[i] }, 4)
	if !fitOK {
		t.Fatalf("computeGoodnessOfFit failed")
	}

	coeffs := []float64{intercept, slope}
	xtxInv := [][]float64{{1.02211999608496, -0.224136243515709}, {-0.224136243515709, 0.0587256533228932}}
	mse := 0.026990799647645954
	se, tv, pv := computeCoeffInference(coeffs, xtxInv, mse, 4)
	ciIntercept, ciSlope := buildTwoCoeffCIs(coeffs[0], coeffs[1], se[0], se[1], 4)
	allCI := buildMultiCoeffCIs(coeffs, se, 4)

	runScalarPrimitiveCases(t, []scalarPrimitiveCase{
		{primitive: "simpleOLSCoeffs_intercept", rFormula: "coef(lm(y~x))[1]", eval: func() float64 { return intercept }, want: 0.96488205931290694, tol: 1e-12},
		{primitive: "simpleOLSCoeffs_slope", rFormula: "coef(lm(y~x))[2]", eval: func() float64 { return slope }, want: 1.8956640892629937, tol: 1e-12},
		{primitive: "simpleOLSSxx", rFormula: "sum((x-mean(x))^2)", eval: func() float64 { return simpleOLSSxx(xs) }, want: 17.028333333333332, tol: 1e-12},
		{primitive: "computeGoodnessOfFit_sse", rFormula: "sum((residuals(lm(y~x)))^2)", eval: func() float64 { return sse }, want: 0.10796319859058381, tol: 1e-12},
		{primitive: "computeGoodnessOfFit_r2", rFormula: "summary(lm(y~x))$r.squared", eval: func() float64 { return r2 }, want: 0.99823877326932164, tol: 1e-12},
		{primitive: "computeGoodnessOfFit_adj_r2", rFormula: "summary(lm(y~x))$adj.r.squared", eval: func() float64 { return adjR2 }, want: 0.99779846658665206, tol: 1e-12},
	})

	runVectorPrimitiveCases(t, []vectorPrimitiveCase{
		{
			primitive: "computeGoodnessOfFit_residuals",
			rFormula:  "residuals(lm(y~x))",
			eval:      func() []float64 { return residuals },
			want:      []float64{-0.0396789664284989, -0.00404228247039118, 0.158559263971813, -0.264069687775276, 0.0880982675932263, 0.0611334051091301},
			tol:       1e-12,
		},
		{
			primitive: "computeCoeffInference_standard_errors",
			rFormula:  "sqrt(diag(summary(lm(y~x))$cov.unscaled * summary(lm(y~x))$sigma^2))",
			eval:      func() []float64 { return se },
			want:      []float64{0.166095863976746, 0.0398127158457612},
			tol:       1e-12,
		},
		{
			primitive: "computeCoeffInference_t_values",
			rFormula:  "summary(lm(y~x))$coefficients[,3]",
			eval:      func() []float64 { return tv },
			want:      []float64{5.80918775586124, 47.6145384456314},
			tol:       1e-12,
		},
		{
			primitive: "computeCoeffInference_p_values",
			rFormula:  "summary(lm(y~x))$coefficients[,4]",
			eval:      func() []float64 { return pv },
			want:      []float64{0.00436938108954035, 1.16390342412039e-06},
			tol:       1e-12,
		},
		{
			primitive: "buildTwoCoeffCIs_intercept",
			rFormula:  "confint(lm(y~x))[1,]",
			eval:      func() []float64 { return []float64{ciIntercept[0], ciIntercept[1]} },
			want:      []float64{0.503726010781071, 1.42603810784474},
			tol:       1e-12,
		},
		{
			primitive: "buildTwoCoeffCIs_slope",
			rFormula:  "confint(lm(y~x))[2,]",
			eval:      func() []float64 { return []float64{ciSlope[0], ciSlope[1]} },
			want:      []float64{1.7851262692284, 2.00620190929759},
			tol:       1e-12,
		},
		{
			primitive: "buildMultiCoeffCIs_intercept",
			rFormula:  "confint(lm(y~x))[1,]",
			eval:      func() []float64 { return []float64{allCI[0][0], allCI[0][1]} },
			want:      []float64{0.503726010781071, 1.42603810784474},
			tol:       1e-12,
		},
		{
			primitive: "buildMultiCoeffCIs_slope",
			rFormula:  "confint(lm(y~x))[2,]",
			eval:      func() []float64 { return []float64{allCI[1][0], allCI[1][1]} },
			want:      []float64{1.7851262692284, 2.00620190929759},
			tol:       1e-12,
		},
	})
}

func rAlmostEqual(got, want, tol float64) bool {
	return math.Abs(got-want) <= tol
}
