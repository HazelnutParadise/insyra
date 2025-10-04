// fa/GPArotation_vgQ_bifactor.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQBifactor computes the objective and gradient for bifactor rotation.
// Mirrors GPArotation::vgQ.bifactor(L)
//
// Returns: Gq (gradient), f (objective), method
func vgQBifactor(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// Lt = L[, 2:k]
	Lt := mat.NewDense(p, k-1, nil)
	for i := 0; i < p; i++ {
		for j := 1; j < k; j++ {
			Lt.Set(i, j-1, L.At(i, j))
		}
	}

	// Lt2 = Lt^2
	Lt2 := mat.NewDense(p, k-1, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k-1; j++ {
			lt := Lt.At(i, j)
			Lt2.Set(i, j, lt*lt)
		}
	}

	// N = ones - diag
	N := mat.NewDense(k-1, k-1, nil)
	for i := 0; i < k-1; i++ {
		for j := 0; j < k-1; j++ {
			if i == j {
				N.Set(i, j, 0)
			} else {
				N.Set(i, j, 1)
			}
		}
	}

	// Lt2N = Lt2 %*% N
	Lt2N := mat.NewDense(p, k-1, nil)
	Lt2N.Mul(Lt2, N)

	// f = sum(Lt2 * Lt2N)
	f = 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k-1; j++ {
			f += Lt2.At(i, j) * Lt2N.At(i, j)
		}
	}

	// Gt = 4 * Lt * Lt2N
	Gt := mat.NewDense(p, k-1, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k-1; j++ {
			Gt.Set(i, j, 4*Lt.At(i, j)*Lt2N.At(i, j))
		}
	}

	// Gq = cbind(0, Gt)
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		Gq.Set(i, 0, 0) // first column 0
		for j := 1; j < k; j++ {
			Gq.Set(i, j, Gt.At(i, j-1))
		}
	}

	method = "Bifactor Biquartimin"
	return
}
