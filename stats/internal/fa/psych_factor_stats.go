// fa/psych_factor_stats.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// FactorStats computes factor statistics.
// Mirrors psych::factor.stats by calling FaStats
func FactorStats(r *mat.Dense, f *mat.Dense, phi *mat.Dense, nObs float64, npObs *mat.VecDense, alpha float64, fm string, smooth, coarse bool) interface{} {
	return FaStats(r, f, phi, nObs, npObs, alpha, fm, smooth, coarse)
}

// FaStats computes goodness of fit statistics for factor analysis.
// Mirrors psych::fa.stats
func FaStats(r *mat.Dense, f *mat.Dense, phi *mat.Dense, nObs float64, npObs *mat.VecDense, alpha float64, fm string, smooth, coarse bool) map[string]interface{} {
	p, _ := r.Dims()
	nf, _ := f.Dims()

	// Compute model implied correlation matrix R_hat = F * Phi * F^T + U^2
	// where U^2 = diag(R) - diag(F * Phi * F^T)

	var FFt mat.Dense
	FFt.Mul(f, f.T())

	var R_hat mat.Dense
	if phi != nil {
		var FPhi mat.Dense
		FPhi.Mul(f, phi)
		R_hat.Mul(&FPhi, f.T())
	} else {
		R_hat = FFt
	}

	// Add uniquenesses to diagonal
	u2 := make([]float64, p)
	for i := 0; i < p; i++ {
		modelVar := R_hat.At(i, i)
		u2[i] = r.At(i, i) - modelVar
		R_hat.Set(i, i, modelVar+u2[i])
	}

	// Compute residuals
	var residual mat.Dense
	residual.Sub(r, &R_hat)

	// RMSR
	rms := 0.0
	count := 0
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			if i != j {
				rms += residual.At(i, j) * residual.At(i, j)
				count++
			}
		}
	}
	rms = math.Sqrt(rms / float64(count))

	// Degrees of freedom
	dof := float64((p * (p - 1) / 2) - (p*nf - nf*(nf-1)/2))

	// Chi-square approximation (simplified)
	chi2 := nObs * float64(count) * rms * rms

	// RMSEA
	rmsea := math.Sqrt(math.Max(chi2/dof-1/(nObs-1), 0))

	// TLI
	nullChi2 := 0.0
	nullDf := float64((p * (p - 1)) / 2)
	for i := 0; i < p; i++ {
		for j := 0; j < i; j++ {
			r_ij := r.At(i, j)
			nullChi2 += -math.Log(1 - r_ij*r_ij)
		}
	}
	nullChi2 *= nObs
	tli := (nullChi2/nullDf - chi2/dof) / (nullChi2/nullDf - 1)

	// BIC
	bic := chi2 - dof*math.Log(nObs)

	result := map[string]interface{}{
		"dof":       dof,
		"objective": chi2,
		"rms":       rms,
		"crms":      rms, // Simplified
		"RMSEA":     rmsea,
		"TLI":       tli,
		"BIC":       bic,
		"SABIC":     bic - math.Log((math.Log(nObs)+1)/2), // Simplified
		"CAIC":      bic + math.Log(nObs+1),
		"EBIC":      bic + 2*alpha*math.Log(float64(p*(p-1)/2)),
	}

	return result
}
