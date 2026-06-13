package stats

import "math"

const (
	glmProbEps   = 1e-10
	glmSmall     = 1e-12
	glmLogEtaMax = 700.0
)

type glmLink interface {
	eta(mu float64) float64
	mu(eta float64) float64
	muEta(eta float64) float64
	name() string
}

type logitLink struct{}

func (logitLink) eta(mu float64) float64 {
	mu = clampProbability(mu)
	return math.Log(mu / (1 - mu))
}

func (logitLink) mu(eta float64) float64 {
	return clampProbability(sigmoid(eta))
}

func (logitLink) muEta(eta float64) float64 {
	mu := logitLink{}.mu(eta)
	return math.Max(mu*(1-mu), glmSmall)
}

func (logitLink) name() string { return string(Logit) }

type logLink struct{}

func (logLink) eta(mu float64) float64 {
	if mu < glmProbEps {
		mu = glmProbEps
	}
	return math.Log(mu)
}

func (logLink) mu(eta float64) float64 {
	if eta > glmLogEtaMax {
		eta = glmLogEtaMax
	}
	mu := math.Exp(eta)
	if mu < glmProbEps {
		return glmProbEps
	}
	return mu
}

func (logLink) muEta(eta float64) float64 {
	return math.Max(logLink{}.mu(eta), glmSmall)
}

func (logLink) name() string { return string(Log) }

type identityLink struct{}

func (identityLink) eta(mu float64) float64 { return mu }
func (identityLink) mu(eta float64) float64 { return eta }
func (identityLink) muEta(float64) float64  { return 1 }
func (identityLink) name() string           { return string(Identity) }

type probitLink struct{}  // TODO: implement in a follow-up issue.
type cloglogLink struct{} // TODO: implement in a follow-up issue.

func clampProbability(mu float64) float64 {
	if math.IsNaN(mu) {
		return mu
	}
	if mu < glmProbEps {
		return glmProbEps
	}
	if mu > 1-glmProbEps {
		return 1 - glmProbEps
	}
	return mu
}
