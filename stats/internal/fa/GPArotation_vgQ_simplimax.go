// fa/GPArotation_vgQ_simplimax.go
package fa

import (
	"sort"

	"gonum.org/v1/gonum/mat"
)

// vgQSimplimax computes the objective and gradient for simplimax rotation.
// Mirrors GPArotation::vgQ.simplimax(L, k = nrow(L))
//
// Imat <- sign(L^2 <= sort(L^2)[k])
// Gq <- 2 * Imat * L
// f <- sum(Imat * L^2)
//
// Returns: Gq (gradient), f (objective), method
func vgQSimplimax(L *mat.Dense, k int) (Gq *mat.Dense, f float64, method string) {
	p, q := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// Flatten L2 to slice for sorting
	l2Slice := make([]float64, p*q)
	idx := 0
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			l2Slice[idx] = L2.At(i, j)
			idx++
		}
	}

	// Sort the slice
	sort.Float64s(l2Slice)

	// threshold = sort(L^2)[k] - the k-th smallest element (0-indexed)
	threshold := l2Slice[k-1] // k is 1-indexed in R, so k-1 in Go

	// Imat = sign(L^2 <= threshold)
	Imat := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			if L2.At(i, j) <= threshold {
				Imat.Set(i, j, 1.0)
			} else {
				Imat.Set(i, j, 0.0)
			}
		}
	}

	// Gq = 2 * Imat * L
	Gq = mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			Gq.Set(i, j, 2.0*Imat.At(i, j)*L.At(i, j))
		}
	}

	// f = sum(Imat * L^2)
	f = 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			f += Imat.At(i, j) * L2.At(i, j)
		}
	}

	method = "Simplimax"
	return
}
