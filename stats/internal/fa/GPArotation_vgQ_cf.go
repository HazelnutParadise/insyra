// fa/GPArotation_vgQ_cf.go
package fa

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

// vgQCf computes the objective and gradient for Crawford-Ferguson rotation.
// Mirrors GPArotation::vgQ.cf(L, kappa = 0)
//
// Returns: Gq (gradient), f (objective), method
func vgQCf(L *mat.Dense, kappa float64) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// N = ones(k,k) - diag(k)
	N := mat.NewDense(k, k, nil)
	for i := 0; i < k; i++ {
		for j := 0; j < k; j++ {
			if i == j {
				N.Set(i, j, 0)
			} else {
				N.Set(i, j, 1)
			}
		}
	}

	// M = ones(p,p) - diag(p)
	M := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			if i == j {
				M.Set(i, j, 0)
			} else {
				M.Set(i, j, 1)
			}
		}
	}

	// L2 = L^2
	L2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// L2N = L2 %*% N
	L2N := mat.NewDense(p, k, nil)
	L2N.Mul(L2, N)

	// crossprod1 = crossprod(L2, L2N) = t(L2) %*% L2N
	crossprod1 := mat.NewDense(k, k, nil)
	crossprod1.Mul(L2.T(), L2N)

	// f1 = (1 - kappa) * sum(diag(crossprod1)) / 4
	f1 := (1 - kappa) * mat.Trace(crossprod1) / 4

	// ML2 = M %*% L2
	ML2 := mat.NewDense(p, k, nil)
	ML2.Mul(M, L2)

	// crossprod2 = crossprod(L2, ML2) = t(L2) %*% ML2
	crossprod2 := mat.NewDense(k, k, nil)
	crossprod2.Mul(L2.T(), ML2)

	// f2 = kappa * sum(diag(crossprod2)) / 4
	f2 := kappa * mat.Trace(crossprod2) / 4

	f = f1 + f2

	// Gq = (1 - kappa) * L * L2N + kappa * L * ML2
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			gq1 := (1 - kappa) * L.At(i, j) * L2N.At(i, j)
			gq2 := kappa * L.At(i, j) * ML2.At(i, j)
			Gq.Set(i, j, gq1+gq2)
		}
	}

	// method
	if kappa == 0 {
		method = "Crawford-Ferguson Quartimax/Quartimin"
	} else if kappa == 1/float64(p) {
		method = "Crawford-Ferguson Varimax"
	} else if kappa == float64(k)/(2*float64(p)) {
		method = "Equamax"
	} else if kappa == float64(k-1)/float64(p+k-2) {
		method = "Parsimax"
	} else if kappa == 1 {
		method = "Factor Parsimony"
	} else {
		method = fmt.Sprintf("Crawford-Ferguson:k=%.3f", kappa)
	}

	return
}
