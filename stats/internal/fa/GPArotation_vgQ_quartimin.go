// fa/GPArotation_vgQ_quartimin.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQQuartimin computes the objective and gradient for quartimin rotation.
// Mirrors GPArotation::vgQ.quartimin(L)
//
// Quartimin criterion: f = (1/4) * sum(sum(L^2 * (L^2 %*% (J - I))))
// where J is all-ones matrix, I is identity matrix
//
// Returns: Gq (gradient), f (objective), method
func vgQQuartimin(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	rows, cols := L.Dims()

	// Compute L^2 element-wise
	L2 := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := L.At(i, j)
			L2.Set(i, j, val*val)
		}
	}

	// Create non-diagonal matrix: J - I (all ones except diagonal is zero)
	nonDiag := mat.NewDense(cols, cols, nil)
	for i := 0; i < cols; i++ {
		for j := 0; j < cols; j++ {
			if i == j {
				nonDiag.Set(i, j, 0) // diagonal elements
			} else {
				nonDiag.Set(i, j, 1) // off-diagonal elements
			}
		}
	}

	// Compute X = L2 %*% nonDiag
	X := mat.NewDense(rows, cols, nil)
	X.Mul(L2, nonDiag)

	// Compute gradient: Gq = L * X (element-wise multiplication)
	Gq = mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			Gq.Set(i, j, L.At(i, j)*X.At(i, j))
		}
	}

	// Compute objective function: f = sum(L^2 * X) / 4
	f = 0.0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			f += L2.At(i, j) * X.At(i, j)
		}
	}
	f /= 4.0

	method = "Quartimin"
	return
}
