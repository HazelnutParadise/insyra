package stats

import "math"

func glmLogLik(y, mu, weights []float64, fam glmFamily, dispersion float64) float64 {
	if len(y) != len(mu) || len(y) != len(weights) {
		return math.NaN()
	}
	if fam.name() == string(Gaussian) {
		if dispersion <= 0 || math.IsNaN(dispersion) {
			return math.NaN()
		}
		sum := 0.0
		for i := range y {
			d := y[i] - mu[i]
			sum += -0.5 * weights[i] * (math.Log(2*math.Pi*dispersion) + d*d/dispersion)
		}
		return sum
	}
	sum := 0.0
	for i := range y {
		sum += fam.logLikContrib(y[i], mu[i], weights[i])
	}
	return sum
}

func glmAIC(logLik float64, k int) float64 {
	return -2*logLik + 2*float64(k)
}

func glmBIC(logLik float64, k, n int) float64 {
	if n <= 0 {
		return math.NaN()
	}
	return -2*logLik + float64(k)*math.Log(float64(n))
}

func responseResiduals(y, mu []float64) []float64 {
	out := make([]float64, len(y))
	for i := range y {
		out[i] = y[i] - mu[i]
	}
	return out
}

func pearsonResiduals(y, mu, weights []float64, fam glmFamily) []float64 {
	out := make([]float64, len(y))
	for i := range y {
		out[i] = math.Sqrt(weights[i]) * (y[i] - mu[i]) / math.Sqrt(fam.variance(mu[i]))
	}
	return out
}

func devianceResiduals(y, mu, weights []float64, fam glmFamily) []float64 {
	out := make([]float64, len(y))
	for i := range y {
		val := fam.devianceResidualSq(y[i], mu[i], weights[i])
		if val < 0 && val > -1e-12 {
			val = 0
		}
		sign := 1.0
		if y[i] < mu[i] {
			sign = -1
		}
		out[i] = sign * math.Sqrt(val)
	}
	return out
}

func expSlice(xs []float64) []float64 {
	out := make([]float64, len(xs))
	for i := range xs {
		out[i] = math.Exp(xs[i])
	}
	return out
}

func expCIs(cis [][2]float64) [][2]float64 {
	out := make([][2]float64, len(cis))
	for i := range cis {
		out[i] = [2]float64{math.Exp(cis[i][0]), math.Exp(cis[i][1])}
	}
	return out
}

func priorWeightsOrOnes(weights []float64, n int) []float64 {
	if weights != nil {
		return append([]float64(nil), weights...)
	}
	out := make([]float64, n)
	for i := range out {
		out[i] = 1
	}
	return out
}
