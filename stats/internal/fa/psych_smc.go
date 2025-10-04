// fa/psych_smc.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// Smc computes the squared multiple correlations (SMC) for a correlation matrix.
// Mirrors psych::smc
func Smc(r *mat.Dense) *mat.VecDense {
	p, _ := r.Dims()

	// Assume r is correlation matrix, no NAs
	var rInv mat.Dense
	err := rInv.Inverse(r)
	if err != nil {
		// If not invertible, use pseudo-inverse approximation
		// For now, set to 1
		smc := mat.NewVecDense(p, nil)
		for i := 0; i < p; i++ {
			smc.SetVec(i, 1.0)
		}
		return smc
	}

	smc := mat.NewVecDense(p, nil)
	for i := 0; i < p; i++ {
		diagInv := rInv.At(i, i)
		val := 1.0 - 1.0/diagInv
		if val > 1.0 {
			val = 1.0
		}
		if val < 0.0 {
			val = 0.0
		}
		smc.SetVec(i, val)
	}

	return smc
}
