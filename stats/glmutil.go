package stats

import "math"

func computeGLMInference(beta []float64, covUnscaled [][]float64, dispersion float64) (se, z, p []float64) {
	n := len(beta)
	se = make([]float64, n)
	z = make([]float64, n)
	p = make([]float64, n)
	for i := range n {
		se[i] = math.NaN()
		z[i] = math.NaN()
		p[i] = math.NaN()
	}

	if dispersion < 0 || math.IsNaN(dispersion) {
		return se, z, p
	}
	for i := range n {
		if covUnscaled == nil || i >= len(covUnscaled) || i >= len(covUnscaled[i]) || covUnscaled[i][i] < 0 {
			continue
		}
		se[i] = math.Sqrt(dispersion * covUnscaled[i][i])
		if se[i] == 0 {
			switch {
			case beta[i] > 0:
				z[i] = math.Inf(1)
				p[i] = 0
			case beta[i] < 0:
				z[i] = math.Inf(-1)
				p[i] = 0
			}
			continue
		}
		if se[i] > 0 {
			z[i] = beta[i] / se[i]
			p[i] = zPValue(z[i], TwoSided)
		}
	}
	return se, z, p
}

func buildGLMCoeffCIs(beta, se []float64, cl float64) [][2]float64 {
	cl = resolveConfidenceLevel(cl)
	cis := make([][2]float64, len(beta))
	zcrit := zQuantile(1 - (1-cl)/2)
	for i := range beta {
		if i >= len(se) || math.IsNaN(se[i]) {
			cis[i] = nanCI()
			continue
		}
		margin := zcrit * se[i]
		cis[i] = [2]float64{beta[i] - margin, beta[i] + margin}
	}
	return cis
}

// buildGLMCoeffCIsT builds coefficient CIs using the t critical value on df
// degrees of freedom. Use this for families with an estimated (non-fixed)
// dispersion (e.g. Gaussian) so the CIs are consistent with the t-distributed
// p-values instead of being too narrow with a z quantile.
func buildGLMCoeffCIsT(beta, se []float64, cl, df float64) [][2]float64 {
	cl = resolveConfidenceLevel(cl)
	cis := make([][2]float64, len(beta))
	if df <= 0 {
		for i := range cis {
			cis[i] = nanCI()
		}
		return cis
	}
	tcrit := tQuantile(1-(1-cl)/2, df)
	for i := range beta {
		if i >= len(se) || math.IsNaN(se[i]) {
			cis[i] = nanCI()
			continue
		}
		margin := tcrit * se[i]
		cis[i] = [2]float64{beta[i] - margin, beta[i] + margin}
	}
	return cis
}

func mcFaddenR2(logLik, nullLogLik float64) float64 {
	if nullLogLik == 0 || math.IsNaN(logLik) || math.IsNaN(nullLogLik) {
		return math.NaN()
	}
	return 1 - logLik/nullLogLik
}

func coxSnellR2(logLik, nullLogLik float64, n int) float64 {
	if n <= 0 || math.IsNaN(logLik) || math.IsNaN(nullLogLik) {
		return math.NaN()
	}
	return 1 - math.Exp((2/float64(n))*(nullLogLik-logLik))
}

func nagelkerkeR2(logLik, nullLogLik float64, n int) float64 {
	if n <= 0 || math.IsNaN(logLik) || math.IsNaN(nullLogLik) {
		return math.NaN()
	}
	denom := 1 - math.Exp((2/float64(n))*nullLogLik)
	if denom == 0 {
		return math.NaN()
	}
	return coxSnellR2(logLik, nullLogLik, n) / denom
}

func pearsonChiSq(y, mu, weightsPriorW []float64, fam glmFamily) float64 {
	if len(y) != len(mu) || len(y) != len(weightsPriorW) {
		return math.NaN()
	}
	sum := 0.0
	for i := range y {
		v := fam.variance(mu[i])
		if v <= 0 || math.IsNaN(v) {
			return math.NaN()
		}
		d := y[i] - mu[i]
		sum += weightsPriorW[i] * d * d / v
	}
	return sum
}

func pearsonDispersion(pearsonChi2 float64, dfResidual int) float64 {
	if dfResidual <= 0 {
		return math.NaN()
	}
	return pearsonChi2 / float64(dfResidual)
}
