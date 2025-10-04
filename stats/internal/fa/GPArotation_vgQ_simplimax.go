// fa/GPArotation_vgQ_simplimax.go
package fa

import (
	"sort"

	"gonum.org/v1/gonum/mat"
)

// vgQSimplimax computes the objective and gradient for simplimax rotation.
// Mirrors GPArotation::vgQ.simplimax(L, k = nrow(L))
//
// Returns: Gq (gradient), f (objective), method
func vgQSimplimax(L *mat.Dense, k int) (Gq *mat.Dense, f float64, method string) {
	p, q := L.Dims()

	// Collect all L^2 values
	allL2 := make([]float64, 0, p*q)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			l := L.At(i, j)
			allL2 = append(allL2, l*l)
		}
	}

	// Sort allL2
	sortedL2 := make([]float64, len(allL2))
	copy(sortedL2, allL2)
	sort.Float64s(sortedL2)

	// threshold = sortedL2[k-1] since 0-based
	threshold := sortedL2[k-1]

	// Imat = 1 if L^2 <= threshold, else 0
	Imat := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			l2 := allL2[i*q+j]
			if l2 <= threshold {
				Imat.Set(i, j, 1)
			} else {
				Imat.Set(i, j, 0)
			}
		}
	}

	// Gq = 2 * Imat * L
	Gq = mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			Gq.Set(i, j, 2*Imat.At(i, j)*L.At(i, j))
		}
	}

	// f = sum(Imat * L^2)
	f = 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			f += Imat.At(i, j) * allL2[i*q+j]
		}
	}

	method = "Simplimax"
	return
}
