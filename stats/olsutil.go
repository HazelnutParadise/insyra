package stats

import "math"

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

func computeCoeffInference(coeffs []float64, xtxInv [][]float64, mse, df float64) (standardErrors, tValues, pValues []float64) {
	n := len(coeffs)
	standardErrors = make([]float64, n)
	tValues = make([]float64, n)
	pValues = make([]float64, n)

	for i := 0; i < n; i++ {
		if xtxInv == nil || i >= len(xtxInv) || i >= len(xtxInv[i]) || xtxInv[i][i] < 0 || math.IsNaN(mse) {
			continue
		}
		standardErrors[i] = math.Sqrt(mse * xtxInv[i][i])
		if standardErrors[i] > 0 {
			tValues[i] = coeffs[i] / standardErrors[i]
			if df > 0 {
				pValues[i] = tTwoTailedPValue(tValues[i], df)
			} else {
				pValues[i] = math.NaN()
			}
		}
	}

	return standardErrors, tValues, pValues
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
