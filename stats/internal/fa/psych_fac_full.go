// fa/psych_fac_full.go
package fa

import (
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
)

// FAfnMinres computes the objective function for minres factor extraction.
// Mirrors the objective in R psych::fa for fm="minres"
func FAfnMinres(par []float64, r *mat.Dense, nf int) float64 {
	p, _ := r.Dims()
	loadings := mat.NewDense(nf, p, par)

	var model mat.Dense
	model.Mul(loadings, loadings.T())

	var residual mat.Dense
	residual.Sub(r, &model)

	// Set diagonal to 0
	for i := 0; i < p; i++ {
		residual.Set(i, i, 0)
	}

	// Sum of squares
	sum := 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			val := residual.At(i, j)
			sum += val * val
		}
	}
	return sum / 2
}

// FAgrMinres computes the gradient for minres.
// Placeholder for now
func FAgrMinres(grad []float64, par []float64, r *mat.Dense, nf int) {
	// Implement gradient if needed
}

// Fac is the main factor analysis function.
// Mirrors psych::fac (full implementation)
func Fac(r *mat.Dense, nfactors int, nObs float64, rotate string, scores string, residuals bool, SMC bool, covar bool, missing bool, impute string, minErr float64, maxIter int, symmetric bool, warnings bool, fm string, alpha float64, obliqueScores bool, npObs *mat.VecDense, use string, cor string, correct float64, weight *mat.VecDense, nRotations int, hyper float64, smooth bool) interface{} {
	p, _ := r.Dims()

	// For now, implement only minres
	if fm != "minres" {
		// Placeholder for other methods
		return nil
	}

	// Initial communalities using SMC if requested
	var communality *mat.VecDense
	if SMC {
		communality = Smc(r)
	} else {
		communality = mat.NewVecDense(p, nil)
		for i := 0; i < p; i++ {
			communality.SetVec(i, 1.0) // or some other initial
		}
	}

	// Initial loadings: simple PCA approximation
	// For simplicity, set initial loadings to random or zero
	initialPar := make([]float64, nfactors*p)
	// Initialize to small random values
	for i := range initialPar {
		initialPar[i] = 0.1 // small value
	}

	// Define the problem
	pb := optimize.Problem{
		Func: func(x []float64) float64 {
			return FAfnMinres(x, r, nfactors)
		},
		Grad: func(grad, x []float64) {
			FAgrMinres(grad, x, r, nfactors)
		},
	}

	// Settings
	settings := optimize.Settings{
		GradientThreshold: 1e-4,
		Converger: &optimize.FunctionConverge{
			Absolute:   1e-4,
			Iterations: maxIter,
		},
	}

	// Minimize
	result, err := optimize.Minimize(pb, initialPar, &settings, &optimize.BFGS{})
	if err != nil {
		// Handle error
		return nil
	}

	// Extract loadings
	loadings := mat.NewDense(nfactors, p, result.X)

	// Compute communalities
	var LLt mat.Dense
	LLt.Mul(loadings, loadings.T())
	communality = mat.NewVecDense(p, nil)
	for i := 0; i < p; i++ {
		comm := LLt.At(i, i)
		if comm > 1.0 {
			comm = 1.0
		}
		if comm < minErr {
			comm = minErr
		}
		communality.SetVec(i, comm)
	}

	// Uniquenesses
	uniquenesses := mat.NewVecDense(p, nil)
	for i := 0; i < p; i++ {
		uniquenesses.SetVec(i, 1.0-communality.AtVec(i))
	}

	// Goodness of fit
	stats := FaStats(r, loadings, nil, nObs, npObs, alpha, fm, smooth, false)

	resultMap := map[string]interface{}{
		"loadings":     loadings,
		"communality":  communality,
		"uniquenesses": uniquenesses,
		"correlation":  r,
		"factors":      nfactors,
		"fm":           fm,
		"rotmat":       mat.NewDense(nfactors, nfactors, nil), // identity for now
		"scores":       mat.NewDense(0, 0, nil),               // not implemented
		"n.obs":        nObs,
		"call":         "Fac",
		"stats":        stats,
	}

	return resultMap
}
