// fa/GPArotation_vgQ_lp_wls.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQLpWls computes the objective and gradient for weighted least squares for Lp rotation.
// Mirrors GPArotation::vgQ.lp.wls(L, W)
//
// Returns: Gq (gradient), f (objective), method
func vgQLpWls(L, W *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// Gq = 2 * W * L / p
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			gq := 2 * W.At(i, j) * L.At(i, j) / float64(p)
			Gq.Set(i, j, gq)
		}
	}

	// f = sum(W * L * L) / p
	f = 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			f += W.At(i, j) * l * l
		}
	}
	f /= float64(p)

	method = "Weighted least squares for Lp rotation"
	return
}
