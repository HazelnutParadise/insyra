// fa/GPArotation_vgQ_bigeomin.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQBiGeomin computes the objective and gradient for bi-geomin rotation.
// Mirrors GPArotation::vgQ.bigeomin(L, delta = 0.01)
//
// Returns: Gq (gradient), f (objective), method
func vgQBiGeomin(L *mat.Dense, delta float64) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// Lg = L[, -1]
	Lg := mat.NewDense(p, k-1, nil)
	for i := 0; i < p; i++ {
		for j := 1; j < k; j++ {
			Lg.Set(i, j-1, L.At(i, j))
		}
	}

	// out = vgQGeomin(Lg, delta)
	outGq, outF, _ := vgQGeomin(Lg, delta)

	// Gq = cbind(0, out$Gq)
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		Gq.Set(i, 0, 0)
		for j := 1; j < k; j++ {
			Gq.Set(i, j, outGq.At(i, j-1))
		}
	}

	f = outF
	method = "Bi-Geomin"
	return
}
