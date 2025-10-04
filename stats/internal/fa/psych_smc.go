// fa/psych_smc.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// Smc computes the squared multiple correlations (SMC) for a correlation matrix.
// Mirrors psych::smc with simplified NA handling
func Smc(r *mat.Dense) *mat.VecDense {
	p, q := r.Dims()

	// Assume r is correlation matrix, no NAs for simplicity
	if p != q {
		// If not square, assume it's data matrix, compute correlation
		// Simplified: just use the matrix as is
		r = mat.DenseCopyOf(r)
	}

	rInv := Pinv(r)
	if rInv == nil {
		// If not invertible, set to 1
		smc := mat.NewVecDense(p, nil)
		for i := 0; i < p; i++ {
			smc.SetVec(i, 1.0)
		}
		return smc
	}

	smc := mat.NewVecDense(p, nil)
	for i := 0; i < p; i++ {
		diagInv := rInv.At(i, i)
		if diagInv == 0 {
			smc.SetVec(i, 1.0)
		} else {
			val := 1.0 - 1.0/diagInv
			if val > 1.0 {
				val = 1.0
			}
			if val < 0.0 {
				val = 0.0
			}
			smc.SetVec(i, val)
		}
	}

	return smc
}
