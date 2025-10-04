// fa/GPArotation_vgQ_geomin.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQGeomin computes the objective and gradient for geomin rotation.
// Mirrors GPArotation::vgQ.geomin(L, delta = 0.01)
//
// k <- ncol(L)
// p <- nrow(L)
// L2 <- L^2 + delta
// pro <- exp(rowSums(log(L2))/k)
// Gq <- (2/k) * (L/L2) * matrix(rep(pro, k), p)
// f <- sum(pro)
//
// Returns: Gq (gradient), f (objective), method
func vgQGeomin(L *mat.Dense, delta float64) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// L2 = L^2 + delta
	L2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l+delta)
		}
	}

	// log(L2)
	logL2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			logL2.Set(i, j, math.Log(L2.At(i, j)))
		}
	}

	// rowSums(log(L2))
	rowSums := make([]float64, p)
	for i := 0; i < p; i++ {
		sum := 0.0
		for j := 0; j < k; j++ {
			sum += logL2.At(i, j)
		}
		rowSums[i] = sum
	}

	// pro = exp(rowSums / k)
	pro := make([]float64, p)
	for i := 0; i < p; i++ {
		pro[i] = math.Exp(rowSums[i] / float64(k))
	}

	// f = sum(pro)
	f = 0.0
	for i := 0; i < p; i++ {
		f += pro[i]
	}

	// L_div_L2 = L / L2
	L_div_L2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			L_div_L2.Set(i, j, L.At(i, j)/L2.At(i, j))
		}
	}

	// matrix(rep(pro, k), p) - create p x k matrix with pro repeated
	proMat := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			proMat.Set(i, j, pro[i])
		}
	}

	// Gq = (2/k) * L_div_L2 * proMat
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			Gq.Set(i, j, (2.0/float64(k))*L_div_L2.At(i, j)*proMat.At(i, j))
		}
	}

	method = "Geomin"
	return
}
