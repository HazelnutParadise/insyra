// fa/GPArotation_vgQ_quartimin.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQQuartimin computes the objective and gradient for quartimin rotation.
// Mirrors GPArotation::vgQ.quartimin(L)
//
// Returns: Gq (gradient), f (objective), method
func vgQQuartimin(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, k, nil)
	for i := range p {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// nonDiag = ones - diag
	nonDiag := mat.NewDense(k, k, nil)
	for i := range k {
		for j := range k {
			if i == j {
				nonDiag.Set(i, j, 0)
			} else {
				nonDiag.Set(i, j, 1)
			}
		}
	}

	// X = L2 %*% nonDiag
	X := mat.NewDense(p, k, nil)
	X.Mul(L2, nonDiag)

	// Gq = L * X
	Gq = mat.NewDense(p, k, nil)
	for i := range p {
		for j := range k {
			Gq.Set(i, j, L.At(i, j)*X.At(i, j))
		}
	}

	// f = sum(L^2 * X) / 4
	f = 0.0
	for i := range p {
		for j := range k {
			f += L2.At(i, j) * X.At(i, j)
		}
	}
	f /= 4

	method = "Quartimin"
	return
}
