// fa/GPArotation_vgQ_oblimin.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQOblimin computes the Oblimin criterion for GPA rotation.
// Mirrors GPArotation::vgQ.oblimin
func vgQOblimin(L *mat.Dense, gam float64) (*mat.Dense, float64, error) {
	p, q := L.Dims()

	// X <- L^2 %*% (!diag(TRUE, ncol(L)))
	// First compute L^2 (element-wise square)
	L2 := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			L2.Set(i, j, L.At(i, j)*L.At(i, j))
		}
	}

	// Create !diag(TRUE, q) which is a matrix of TRUEs except diagonal is FALSE
	notDiag := mat.NewDense(q, q, nil)
	for i := 0; i < q; i++ {
		for j := 0; j < q; j++ {
			if i != j {
				notDiag.Set(i, j, 1.0) // TRUE becomes 1.0
			} else {
				notDiag.Set(i, j, 0.0) // FALSE becomes 0.0
			}
		}
	}

	// X <- L^2 %*% (!diag(TRUE, ncol(L)))
	X := mat.NewDense(p, q, nil)
	X.Mul(L2, notDiag)

	// if (0 != gam) {
	//     p <- nrow(L)
	//     X <- (diag(1, p) - matrix(gam/p, p, p)) %*% X
	// }
	if gam != 0.0 {
		// Create diag(1, p)
		diag1 := mat.NewDense(p, p, nil)
		for i := 0; i < p; i++ {
			diag1.Set(i, i, 1.0)
		}

		// Create matrix(gam/p, p, p)
		gamOverP := gam / float64(p)
		gamMat := mat.NewDense(p, p, nil)
		for i := 0; i < p; i++ {
			for j := 0; j < p; j++ {
				gamMat.Set(i, j, gamOverP)
			}
		}

		// diag(1, p) - matrix(gam/p, p, p)
		temp := mat.NewDense(p, p, nil)
		temp.Sub(diag1, gamMat)

		// X <- temp %*% X
		Xnew := mat.NewDense(p, q, nil)
		Xnew.Mul(temp, X)
		X = Xnew
	}

	// Gq = L * X (element-wise multiplication)
	Gq := mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			Gq.Set(i, j, L.At(i, j)*X.At(i, j))
		}
	}

	// f = sum(L^2 * X)/4
	f := 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			f += L2.At(i, j) * X.At(i, j)
		}
	}
	f /= 4.0

	return Gq, f, nil
}
