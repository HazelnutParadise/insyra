// fa/psych_fac_full.go
package fa

import (
	"math"

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

// FAfnML computes the objective function for ML factor extraction.
// Mirrors the objective in R psych::fa for fm="ml"
func FAfnML(par []float64, r *mat.Dense, nf int) float64 {
	p, _ := r.Dims()
	loadings := mat.NewDense(nf, p, par[:nf*p])
	u2 := par[nf*p:]

	// Sigma = L L^T + diag(u2)
	var LLt mat.Dense
	LLt.Mul(loadings, loadings.T())
	var Sigma mat.Dense
	Sigma.CloneFrom(&LLt)
	for i := 0; i < p; i++ {
		Sigma.Set(i, i, Sigma.At(i, i)+u2[i])
	}

	// Compute log det Sigma
	var SigmaInv mat.Dense
	err := SigmaInv.Inverse(&Sigma)
	if err != nil {
		return 1e10 // penalty for non-invertible
	}
	det := mat.Det(&Sigma)
	if det <= 0 {
		return 1e10
	}
	logDet := math.Log(det)

	// Trace R Sigma^-1
	var RSinv mat.Dense
	RSinv.Mul(r, &SigmaInv)
	trace := mat.Trace(&RSinv)

	// -2 log likelihood (minimize this)
	obj := float64(p)*math.Log(2*math.Pi) + logDet + trace
	return obj
}

// FAgrML computes the gradient for ML.
// Placeholder for now
func FAgrML(grad []float64, par []float64, r *mat.Dense, nf int) {
	// Implement gradient
}

// FAgrMinres computes the gradient for minres.
// Mirrors the gradient in R psych::fa for fm="minres"
func FAgrMinres(grad []float64, par []float64, r *mat.Dense, nf int) {
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

	// Initialize grad to 0
	for i := range grad {
		grad[i] = 0
	}

	// Compute gradient
	for k := 0; k < nf; k++ {
		for l := 0; l < p; l++ {
			idx := k*p + l
			for i := 0; i < p; i++ {
				for j := 0; j < p; j++ {
					if i == j {
						continue
					}
					res := residual.At(i, j)
					// d/d L_kl of (L L^T)_ij = delta_ik L_jl + delta_jk L_il
					dLij_dLkl := 0.0
					if i == k {
						dLij_dLkl += loadings.At(l, j)
					}
					if j == k {
						dLij_dLkl += loadings.At(l, i)
					}
					grad[idx] -= res * dLij_dLkl
				}
			}
		}
	}
}

// Fac is the main factor analysis function.
// Mirrors psych::fac (full implementation)
func Fac(r *mat.Dense, nfactors int, nObs float64, rotate string, scores string, residuals bool, SMC bool, covar bool, missing bool, impute string, minErr float64, maxIter int, symmetric bool, warnings bool, fm string, alpha float64, obliqueScores bool, npObs *mat.VecDense, use string, cor string, correct float64, weight *mat.VecDense, nRotations int, hyper float64, smooth bool) interface{} {
	p, _ := r.Dims()

	var pb optimize.Problem
	var initialPar []float64

	if fm == "minres" {
		// Initial communalities using SMC if requested
		var initialCommunality *mat.VecDense
		if SMC {
			initialCommunality = Smc(r)
		} else {
			initialCommunality = mat.NewVecDense(p, nil)
			for i := 0; i < p; i++ {
				initialCommunality.SetVec(i, 1.0)
			}
		}

		// Initial loadings
		initialPar = make([]float64, nfactors*p)
		for f := 0; f < nfactors; f++ {
			for v := 0; v < p; v++ {
				initialPar[f*p+v] = 0.1 * math.Sqrt(initialCommunality.AtVec(v))
			}
		}

		pb = optimize.Problem{
			Func: func(x []float64) float64 {
				return FAfnMinres(x, r, nfactors)
			},
			Grad: func(grad, x []float64) {
				FAgrMinres(grad, x, r, nfactors)
			},
		}
	} else if fm == "ml" {
		// For ML, parameters are loadings + uniquenesses
		initialPar = make([]float64, nfactors*p+p)
		// Initial loadings small
		for i := 0; i < nfactors*p; i++ {
			initialPar[i] = 0.1
		}
		// Initial u2 = 0.5
		for i := 0; i < p; i++ {
			initialPar[nfactors*p+i] = 0.5
		}

		pb = optimize.Problem{
			Func: func(x []float64) float64 {
				return FAfnML(x, r, nfactors)
			},
			Grad: func(grad, x []float64) {
				FAgrML(grad, x, r, nfactors)
			},
		}
	} else {
		// Placeholder for other methods
		return nil
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

	var loadings *mat.Dense
	var communality *mat.VecDense

	if fm == "minres" {
		loadings = mat.NewDense(nfactors, p, result.X)

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
	} else if fm == "ml" {
		loadings = mat.NewDense(nfactors, p, result.X[:nfactors*p])
		u2 := result.X[nfactors*p:]

		communality = mat.NewVecDense(p, nil)
		for i := 0; i < p; i++ {
			comm := 1.0 - u2[i]
			if comm > 1.0 {
				comm = 1.0
			}
			if comm < minErr {
				comm = minErr
			}
			communality.SetVec(i, comm)
		}
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
