// fa/GPArotation_vgQ_quartimax.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQQuartimax computes the objective and gradient for quartimax rotation.
// Mirrors GPArotation::vgQ.quartimax(L)
//
// f <- -sum(diag(crossprod(L^2)))/4
// Gq <- -L^3
//
// Returns: Gq (gradient), f (objective), method
func vgQQuartimax(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, q := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// crossprod = t(L2) %*% L2, but sum(diag(crossprod(L2))) = sum(colSums(L2))
	sumDiag := 0.0
	for j := 0; j < q; j++ {
		colSum := 0.0
		for i := 0; i < p; i++ {
			colSum += L2.At(i, j)
		}
		sumDiag += colSum
	}

	f = -sumDiag / 4

	// Gq = -L^3
	Gq = mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			l := L.At(i, j)
			Gq.Set(i, j, -l*l*l)
		}
	}

	method = "Quartimax"
	return
}
