// fa/GPArotation_vgQ_oblimin.go
package fa

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

// vgQOblimin computes the objective and gradient for oblimin rotation.
// Mirrors GPArotation::vgQ.oblimin(L, gam = 0)
//
// X <- L^2 %*% (!diag(TRUE, ncol(L)))
// if (gam != 0) X <- (diag(1, p) - matrix(gam/p, p, p)) %*% X
// Gq <- L * X
// f <- sum(L^2 * X)/4
//
// Returns: Gq (gradient), f (objective), method
func vgQOblimin(L *mat.Dense, gam float64) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// nonDiag = ones - diag
	ones := mat.NewDense(k, k, nil)
	for i := 0; i < k; i++ {
		for j := 0; j < k; j++ {
			ones.Set(i, j, 1)
		}
	}
	diagMat := mat.NewDense(k, k, nil)
	for i := 0; i < k; i++ {
		diagMat.Set(i, i, 1)
	}
	nonDiag := mat.NewDense(k, k, nil)
	nonDiag.Sub(ones, diagMat)

	// X = L2 %*% nonDiag
	X := mat.NewDense(p, k, nil)
	X.Mul(L2, nonDiag)

	if gam != 0 {
		pFloat := float64(p)
		adjust := mat.NewDense(p, p, nil)
		for i := 0; i < p; i++ {
			for j := 0; j < p; j++ {
				if i == j {
					adjust.Set(i, j, 1-gam/pFloat)
				} else {
					adjust.Set(i, j, -gam/pFloat)
				}
			}
		}
		XTemp := mat.NewDense(p, k, nil)
		XTemp.Mul(adjust, X)
		X = XTemp
	}

	// Gq = L * X
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			Gq.Set(i, j, L.At(i, j)*X.At(i, j))
		}
	}

	// f = sum(L^2 * X) / 4
	f = 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			f += L2.At(i, j) * X.At(i, j)
		}
	}
	f /= 4

	// method
	if gam == 0 {
		method = "Oblimin Quartimin"
	} else if gam == 0.5 {
		method = "Oblimin Biquartimin"
	} else {
		method = fmt.Sprintf("Oblimin g=%.1f", gam)
	}

	return
}
