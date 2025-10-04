// fa/psych_fa_wrapper.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// Fa is the wrapper for factor analysis.
// Mirrors psych::fa
func Fa(r *mat.Dense, nfactors int, nObs float64, nIter int, rotate string, scores string, residuals bool, SMC bool, covar bool, missing bool, impute string, minErr float64, maxIter int, symmetric bool, warnings bool, fm string, alpha float64, p float64, obliqueScores bool, npObs *mat.VecDense, use string, cor string, correct float64, weight *mat.VecDense, nRotations int, hyper float64, smooth bool) interface{} {
	// Calls Fac
	return Fac(r, nfactors, nObs, rotate, scores, residuals, SMC, covar, missing, impute, minErr, maxIter, symmetric, warnings, fm, alpha, obliqueScores, npObs, use, cor, correct, weight, nRotations, hyper, smooth)
}
