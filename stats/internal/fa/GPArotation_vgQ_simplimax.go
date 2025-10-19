// fa/GPArotation_vgQ_simplimax.go
package fa

import (
	"sort"

	"gonum.org/v1/gonum/mat"
)

// vgQSimplimax computes the objective and gradient for simplimax rotation.
// Mirrors GPArotation::vgQ.simplimax(L, k = nrow(L))
//
// Simplimax minimizes the number of variables with high loadings on each factor.
// It uses a threshold based on the k-th smallest squared loading.
//
// Imat <- sign(L^2 <= sort(L^2)[k])  [indicator matrix for small loadings]
// Gq <- 2 * Imat * L
// f <- sum(Imat * L^2)
//
// Returns: Gq (gradient), f (objective), method
func vgQSimplimax(L *mat.Dense, k int) (Gq *mat.Dense, f float64, method string) {
	rows, cols := L.Dims()

	// Compute L^2 element-wise
	L2 := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := L.At(i, j)
			L2.Set(i, j, val*val)
		}
	}

	// Flatten L2 to slice for sorting to find threshold
	l2Values := make([]float64, rows*cols)
	idx := 0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			l2Values[idx] = L2.At(i, j)
			idx++
		}
	}

	// Sort to find the k-th smallest value (k is 1-indexed)
	sort.Float64s(l2Values)
	threshold := l2Values[k-1]

	// Create indicator matrix: 1 where L^2 <= threshold, 0 otherwise
	indicator := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			if L2.At(i, j) <= threshold {
				indicator.Set(i, j, 1.0)
			} else {
				indicator.Set(i, j, 0.0)
			}
		}
	}

	// Gradient: Gq = 2 * indicator * L (element-wise)
	Gq = mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			Gq.Set(i, j, 2.0*indicator.At(i, j)*L.At(i, j))
		}
	}

	// Objective: f = sum(indicator * L^2)
	f = 0.0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			f += indicator.At(i, j) * L2.At(i, j)
		}
	}

	method = "Simplimax"
	return
}
