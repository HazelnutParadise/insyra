// fa/GPArotation_vgQ_tandemI.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQTandemI computes the objective and gradient for tandem I rotation.
// Mirrors GPArotation::vgQ.tandemI(L)
//
// Returns: Gq (gradient), f (objective), method
func vgQTandemI(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// LL = L %*% t(L)
	LL := mat.NewDense(p, p, nil)
	LL.Mul(L, L.T())

	// LL2 = LL^2
	LL2 := mat.NewDense(p, p, nil)
	LL2.Mul(LL, LL)

	// LL2_L2 = LL2 %*% L2
	LL2_L2 := mat.NewDense(p, k, nil)
	LL2_L2.Mul(LL2, L2)

	// Gq1 = 4 * L * LL2_L2
	Gq1 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			Gq1.Set(i, j, 4*L.At(i, j)*LL2_L2.At(i, j))
		}
	}

	// L2_t = t(L2)
	L2_t := L2.T()

	// L2_t_L2 = L2_t %*% L2
	L2_t_L2 := mat.NewDense(k, k, nil)
	L2_t_L2.Mul(L2_t, L2)

	// LL_L2_t_L2 = LL * L2_t_L2
	LL_L2_t_L2 := mat.NewDense(p, k, nil)
	LL_L2_t_L2.Mul(LL, L2_t_L2)

	// Gq2 = 4 * LL_L2_t_L2 * L
	Gq2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			Gq2.Set(i, j, 4*LL_L2_t_L2.At(i, j)*L.At(i, j))
		}
	}

	// Gq = -Gq1 - Gq2
	Gq = mat.NewDense(p, k, nil)
	Gq.Sub(Gq1, Gq2)
	Gq.Scale(-1, Gq)

	// f = -sum(diag(t(L2) %*% LL2_L2))
	var crossprod mat.Dense
	crossprod.Mul(L2.T(), LL2_L2)
	f = 0.0
	for i := 0; i < k; i++ {
		f -= crossprod.At(i, i)
	}

	method = "Tandem I"
	return
}
