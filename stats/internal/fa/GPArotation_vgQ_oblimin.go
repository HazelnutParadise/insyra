// fa/GPArotation_vgQ_oblimin.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQOblimin computes the Oblimin criterion for GPA rotation.
// Mirrors GPArotation::vgQ.oblimin
//
// Oblimin is a generalization of quartimin with parameter gamma:
// - gamma = 0: quartimin
// - gamma > 0: more oblique solutions
// - gamma < 0: more orthogonal solutions
func vgQOblimin(L *mat.Dense, gamma float64) (*mat.Dense, float64, error) {
	rows, cols := L.Dims()

	// Compute L^2 element-wise
	L2 := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := L.At(i, j)
			L2.Set(i, j, val*val)
		}
	}

	// Create off-diagonal matrix (ones except diagonal is zero)
	offDiag := mat.NewDense(cols, cols, nil)
	for i := 0; i < cols; i++ {
		for j := 0; j < cols; j++ {
			if i != j {
				offDiag.Set(i, j, 1.0)
			} else {
				offDiag.Set(i, j, 0.0)
			}
		}
	}

	// Compute X = L^2 %*% offDiag
	X := mat.NewDense(rows, cols, nil)
	X.Mul(L2, offDiag)

	// Apply gamma correction if gamma != 0
	if gamma != 0.0 {
		gammaOverRows := gamma / float64(rows)
		for i := 0; i < rows; i++ {
			// Compute row sum of L2
			rowSum := 0.0
			for j := 0; j < cols; j++ {
				rowSum += L2.At(i, j)
			}
			// Apply correction: X = X - (gamma/rows) * (rowSum - L2_ij) for each j
			for j := 0; j < cols; j++ {
				correction := gammaOverRows * (rowSum - L2.At(i, j))
				X.Set(i, j, X.At(i, j)-correction)
			}
		}
	}

	// Compute gradient: Gq = L * X (element-wise)
	Gq := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			Gq.Set(i, j, L.At(i, j)*X.At(i, j))
		}
	}

	// Compute objective: f = sum(L^2 * X)
	f := 0.0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			f += L2.At(i, j) * X.At(i, j)
		}
	}

	return Gq, f, nil
}

// VgQOblimin is the exported version for testing
func VgQOblimin(L *mat.Dense, gamma float64) (*mat.Dense, float64, error) {
	return vgQOblimin(L, gamma)
}
