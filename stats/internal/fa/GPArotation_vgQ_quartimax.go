// fa/GPArotation_vgQ_quartimax.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQQuartimax computes the objective and gradient for quartimax rotation.
// Mirrors GPArotation::vgQ.quartimax(L)
//
// Quartimax criterion minimizes the sum of squares of all loadings:
// f <- -sum(L^2)/4  [negative because GPA minimizes]
// Gq <- -L^3
//
// Returns: Gq (gradient), f (objective), method
func vgQQuartimax(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	rows, cols := L.Dims()

	// Compute sum of squares of all elements: sum(L^2)
	sumSquares := 0.0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := L.At(i, j)
			sumSquares += val * val
		}
	}

	// Objective function: f = -sum(L^2)/4
	f = -sumSquares / 4.0

	// Gradient: Gq = -L^3 (element-wise)
	Gq = mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := L.At(i, j)
			Gq.Set(i, j, -val*val*val)
		}
	}

	method = "Quartimax"
	return
}
