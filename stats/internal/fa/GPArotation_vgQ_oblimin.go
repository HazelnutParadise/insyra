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
		colSums := make([]float64, cols)
		for j := 0; j < cols; j++ {
			for i := 0; i < rows; i++ {
				colSums[j] += X.At(i, j)
			}
		}
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				X.Set(i, j, X.At(i, j)-gammaOverRows*colSums[j])
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

	return Gq, f / 4, nil
}

// VgQOblimin is the exported version for testing
func VgQOblimin(L *mat.Dense, gamma float64) (*mat.Dense, float64, error) {
	return vgQOblimin(L, gamma)
}
