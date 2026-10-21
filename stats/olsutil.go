package stats

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

func simpleOLSCoeffs(xs, ys []float64) (intercept, slope float64, ok bool) {
	if len(xs) != len(ys) || len(xs) < 2 {
		return 0, 0, false
	}

	n := float64(len(xs))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
		sumXY += xs[i] * ys[i]
		sumX2 += xs[i] * xs[i]
	}

	denom := n*sumX2 - sumX*sumX
	if denom == 0 {
		return 0, 0, false
	}

	slope = (n*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / n
	return intercept, slope, true
}

func simpleOLSSxx(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	mean := sum / float64(len(xs))

	sxx := 0.0
	for _, x := range xs {
		diff := x - mean
		sxx += diff * diff
	}
	return sxx
}

func computeGoodnessOfFit(ys []float64, predict func(i int) float64, df float64) (residuals []float64, r2, adjR2, sse float64, ok bool) {
	n := len(ys)
	if n == 0 {
		return nil, math.NaN(), math.NaN(), math.NaN(), false
	}

	meanY := 0.0
	for _, y := range ys {
		meanY += y
	}
	meanY /= float64(n)

	residuals = make([]float64, n)
	sst := 0.0
	for i := range ys {
		yHat := predict(i)
		residuals[i] = ys[i] - yHat
		sse += residuals[i] * residuals[i]
		diff := ys[i] - meanY
		sst += diff * diff
	}

	if sst == 0 {
		return residuals, math.NaN(), math.NaN(), sse, false
	}

	r2 = 1.0 - sse/sst
	if df > 0 {
		adjR2 = 1.0 - (1.0-r2)*(float64(n-1))/df
	} else {
		adjR2 = math.NaN()
	}
	return residuals, r2, adjR2, sse, true
}

// computeCoeffInference returns the classical OLS standard errors, t values
// and two-tailed p values: each variance is mse * (XᵀX)⁻¹[i][i].
func computeCoeffInference(coeffs []float64, xtxInv [][]float64, mse, df float64) (standardErrors, tValues, pValues []float64) {
	variances := make([]float64, len(coeffs))
	for i := range variances {
		variances[i] = math.NaN()
		if xtxInv == nil || i >= len(xtxInv) || i >= len(xtxInv[i]) || xtxInv[i][i] < 0 || math.IsNaN(mse) {
			continue
		}
		variances[i] = mse * xtxInv[i][i]
	}
	return coeffInferenceFromVariances(coeffs, variances, df)
}

// coeffInferenceFromVariances turns one variance per coefficient into standard
// errors, t values and two-tailed p values from the t distribution with df
// degrees of freedom. A NaN or negative variance leaves that coefficient's
// three values NaN.
func coeffInferenceFromVariances(coeffs, variances []float64, df float64) (standardErrors, tValues, pValues []float64) {
	n := len(coeffs)
	standardErrors = make([]float64, n)
	tValues = make([]float64, n)
	pValues = make([]float64, n)
	for i := 0; i < n; i++ {
		standardErrors[i] = math.NaN()
		tValues[i] = math.NaN()
		pValues[i] = math.NaN()
	}

	for i := 0; i < n; i++ {
		if i >= len(variances) || math.IsNaN(variances[i]) || variances[i] < 0 {
			continue
		}
		standardErrors[i] = math.Sqrt(variances[i])
		if standardErrors[i] == 0 {
			switch {
			case coeffs[i] > 0:
				tValues[i] = math.Inf(1)
				if df > 0 {
					pValues[i] = 0
				}
			case coeffs[i] < 0:
				tValues[i] = math.Inf(-1)
				if df > 0 {
					pValues[i] = 0
				}
			}
		} else if standardErrors[i] > 0 {
			tValues[i] = coeffs[i] / standardErrors[i]
			if df > 0 {
				pValues[i] = tTwoTailedPValue(tValues[i], df)
			}
		}
	}

	return standardErrors, tValues, pValues
}

func inferTwoCoeffStats(intercept, slope, seIntercept, seSlope, df float64) (tIntercept, tSlope, pIntercept, pSlope float64, ciIntercept, ciSlope [2]float64) {
	tIntercept = intercept / seIntercept
	tSlope = slope / seSlope
	pIntercept = tTwoTailedPValue(tIntercept, df)
	pSlope = tTwoTailedPValue(tSlope, df)
	ciIntercept, ciSlope = buildTwoCoeffCIs(intercept, slope, seIntercept, seSlope, df)
	return
}

func buildTwoCoeffCIs(intercept, slope, seIntercept, seSlope, df float64) (ciIntercept, ciSlope [2]float64) {
	if df <= 0 {
		return nanCI(), nanCI()
	}

	criticalValue := tQuantile(1-(1-defaultConfidenceLevel)/2, df)
	marginIntercept := criticalValue * seIntercept
	marginSlope := criticalValue * seSlope

	ciIntercept = [2]float64{intercept - marginIntercept, intercept + marginIntercept}
	ciSlope = [2]float64{slope - marginSlope, slope + marginSlope}
	return ciIntercept, ciSlope
}

func buildMultiCoeffCIs(coeffs, standardErrors []float64, df float64) [][2]float64 {
	cis := make([][2]float64, len(coeffs))
	if df <= 0 {
		for i := range cis {
			cis[i] = nanCI()
		}
		return cis
	}

	criticalValue := tQuantile(1-(1-defaultConfidenceLevel)/2, df)
	for i := range coeffs {
		se := math.NaN()
		if i < len(standardErrors) {
			se = standardErrors[i]
		}
		margin := criticalValue * se
		cis[i] = [2]float64{coeffs[i] - margin, coeffs[i] + margin}
	}
	return cis
}

// olsSandwichVariances returns the diagonal of the sandwich covariance
// B·M·B of OLS coefficients, B = (XᵀX)⁻¹, where M is built from the scaled
// scores u_t = c_t·x_t:
//
//	c_t = e_t / (1 - h_t)^leveragePower   (leveragePower is 0, 0.5 or 1)
//	M   = meatScale · ( Σ u_t u_tᵀ + Σ_{j=1..lag} (1 - j/(lag+1)) · (Γ_j + Γ_jᵀ) )
//	Γ_j = Σ_{t=j..n-1} u_t u_{t-j}ᵀ
//
// h_t is the leverage x_tᵀ B x_t. leveragePower 0 with meatScale 1 is HC0,
// meatScale n/(n-k) is HC1, leveragePower 0.5 is HC2 and 1 is HC3, the
// estimators R's sandwich::vcovHC computes. lag > 0 adds the Bartlett-weighted
// autocovariances of sandwich::NeweyWest(prewhite = FALSE, adjust = FALSE);
// with lag 0 it runs exactly the HC0 arithmetic. A leverage of 1 makes HC2 and
// HC3 divide by zero, and the affected variances come out NaN or Inf, as they
// do in sandwich.
func olsSandwichVariances(X *mat.Dense, residuals []float64, xtxInv [][]float64, leveragePower, meatScale float64, lag int) []float64 {
	n, k := X.Dims()
	bread := mat.NewDense(k, k, nil)
	for i := range k {
		for j := range k {
			bread.Set(i, j, xtxInv[i][j])
		}
	}

	scores := mat.NewDense(n, k, nil)
	row := make([]float64, k)
	for t := range n {
		mat.Row(row, t, X)
		c := residuals[t]
		if leveragePower != 0 {
			h := 0.0
			for a := range k {
				inner := 0.0
				for b := range k {
					inner += xtxInv[a][b] * row[b]
				}
				h += row[a] * inner
			}
			if leveragePower == 1 {
				c /= 1 - h
			} else { // 0.5
				c /= math.Sqrt(1 - h)
			}
		}
		for a := range k {
			scores.Set(t, a, c*row[a])
		}
	}

	var meat mat.Dense
	meat.Mul(scores.T(), scores)
	for j := 1; j <= lag; j++ {
		var gamma mat.Dense
		gamma.Mul(scores.Slice(j, n, 0, k).T(), scores.Slice(0, n-j, 0, k))
		w := 1 - float64(j)/float64(lag+1)
		for a := range k {
			for b := range k {
				meat.Set(a, b, meat.At(a, b)+w*(gamma.At(a, b)+gamma.At(b, a)))
			}
		}
	}
	if meatScale != 1 {
		meat.Scale(meatScale, &meat)
	}

	var left, cov mat.Dense
	left.Mul(bread, &meat)
	cov.Mul(&left, bread)
	variances := make([]float64, k)
	for i := range k {
		variances[i] = cov.At(i, i)
	}
	return variances
}
