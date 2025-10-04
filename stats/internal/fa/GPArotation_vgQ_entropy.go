// fa/GPArotation_vgQ_entropy.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQEntropy computes the objective and gradient for entropy rotation.
// Mirrors GPArotation::vgQ.entropy(L)
//
// Returns: Gq (gradient), f (objective), method
func vgQEntropy(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// logTerm = log(L2 + (L2 == 0))
	logTerm := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l2 := L2.At(i, j)
			if l2 == 0 {
				logTerm.Set(i, j, 0) // log(0 + 1) = 0
			} else {
				logTerm.Set(i, j, math.Log(l2))
			}
		}
	}

	// Gq = -(L * logTerm + L)
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			gq := -(l*logTerm.At(i, j) + l)
			Gq.Set(i, j, gq)
		}
	}

	// f = -sum(L2 * logTerm) / 2
	f = 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			f -= L2.At(i, j) * logTerm.At(i, j)
		}
	}
	f /= 2

	method = "Minimum entropy"
	return
}
