package stats

import "math"

type glmFamily interface {
	canonicalLink() glmLink
	variance(mu float64) float64
	devianceResidualSq(y, mu, w float64) float64
	logLikContrib(y, mu, w float64) float64
	initMu(y, w float64) float64
	dispersionFixed() (phi float64, fixed bool)
	name() string
}

type binomialFamily struct{}

func (binomialFamily) canonicalLink() glmLink { return logitLink{} }

func (binomialFamily) variance(mu float64) float64 {
	mu = clampProbability(mu)
	return math.Max(mu*(1-mu), glmSmall)
}

func (binomialFamily) devianceResidualSq(y, mu, w float64) float64 {
	mu = clampProbability(mu)
	term := 0.0
	if y > 0 {
		term += y * math.Log(y/mu)
	}
	if y < 1 {
		term += (1 - y) * math.Log((1-y)/(1-mu))
	}
	return 2 * w * term
}

func (binomialFamily) logLikContrib(y, mu, w float64) float64 {
	mu = clampProbability(mu)
	return w * (y*math.Log(mu) + (1-y)*math.Log(1-mu))
}

func (binomialFamily) initMu(y, w float64) float64 {
	if w <= 0 {
		return 0.5
	}
	return clampProbability((w*y + 0.5) / (w + 1))
}

func (binomialFamily) dispersionFixed() (float64, bool) { return 1, true }
func (binomialFamily) name() string                     { return string(Binomial) }

type poissonFamily struct{}

func (poissonFamily) canonicalLink() glmLink { return logLink{} }

func (poissonFamily) variance(mu float64) float64 {
	if mu < glmProbEps {
		return glmProbEps
	}
	return mu
}

func (poissonFamily) devianceResidualSq(y, mu, w float64) float64 {
	if mu < glmProbEps {
		mu = glmProbEps
	}
	term := -y + mu
	if y > 0 {
		term += y * math.Log(y/mu)
	}
	return 2 * w * term
}

func (poissonFamily) logLikContrib(y, mu, w float64) float64 {
	if mu < glmProbEps {
		mu = glmProbEps
	}
	lgamma, _ := math.Lgamma(y + 1)
	return w * (y*math.Log(mu) - mu - lgamma)
}

func (poissonFamily) initMu(y, w float64) float64 {
	if y < 0 {
		return glmProbEps
	}
	return math.Max(y+0.1, glmProbEps)
}

func (poissonFamily) dispersionFixed() (float64, bool) { return 1, true }
func (poissonFamily) name() string                     { return string(Poisson) }

type gaussianFamily struct{}

func (gaussianFamily) canonicalLink() glmLink { return identityLink{} }
func (gaussianFamily) variance(float64) float64 {
	return 1
}

func (gaussianFamily) devianceResidualSq(y, mu, w float64) float64 {
	d := y - mu
	return w * d * d
}

func (gaussianFamily) logLikContrib(y, mu, w float64) float64 {
	d := y - mu
	return -0.5 * w * (math.Log(2*math.Pi) + d*d)
}

func (gaussianFamily) initMu(y, w float64) float64      { return y }
func (gaussianFamily) dispersionFixed() (float64, bool) { return math.NaN(), false }
func (gaussianFamily) name() string                     { return string(Gaussian) }
