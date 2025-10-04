// fa/GPArotation_vgQ_oblimax.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQOblimax computes the objective and gradient for oblimax rotation.
// Mirrors GPArotation::vgQ.oblimax(L)
//
// Returns: Gq (gradient), f (objective), method
func vgQOblimax(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// sumL2 = sum(L^2)
	sumL2 := 0.0
	// sumL4 = sum(L^4)
	sumL4 := 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			l2 := l * l
			sumL2 += l2
			sumL4 += l2 * l2
		}
	}

	// Gq = -(4 * L^3 / sumL4 - 4 * L / sumL2)
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			l3 := l * l * l
			gq := -(4*l3/sumL4 - 4*l/sumL2)
			Gq.Set(i, j, gq)
		}
	}

	f = -(math.Log(sumL4) - 2*math.Log(sumL2))

	method = "Oblimax"
	return
}
