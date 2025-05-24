package stats

import "gonum.org/v1/gonum/stat/distuv"

var (
	defaultConfidenceLevel = 0.95
	norm                   = distuv.Normal{Mu: 0, Sigma: 1}
)
