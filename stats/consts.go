package stats

import "gonum.org/v1/gonum/stat/distuv"

type GLMFamily string
type GLMLink string
type SeparationPolicy string

const (
	Binomial GLMFamily = "binomial"
	Poisson  GLMFamily = "poisson"
	Gaussian GLMFamily = "gaussian"
)

const (
	Log      GLMLink = "log"
	Logit    GLMLink = "logit"
	Identity GLMLink = "identity"
	Probit   GLMLink = "probit"
	Cloglog  GLMLink = "cloglog"
)

const (
	SepWarn  SeparationPolicy = "warn"
	SepError SeparationPolicy = "error"
	SepRidge SeparationPolicy = "ridge"
)

var (
	defaultConfidenceLevel = 0.95
	norm                   = distuv.Normal{Mu: 0, Sigma: 1}
)
