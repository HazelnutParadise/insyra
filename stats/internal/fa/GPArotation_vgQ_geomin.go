// fa/GPArotation_vgQ_geomin.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// VgQGeomin computes the objective and gradient for geomin rotation.
// Mirrors GPArotation::vgQ.geomin(L, delta = 0.01)
//
// L2 <- L^2 + delta
// pro <- exp(rowSums(log(L2))/k)
// Gq <- (2/k) * (L/L2) * matrix(rep(pro, k), p)
// f <- sum(pro)
//
// Returns: Gq (gradient), f (objective), method
func VgQGeomin(L *mat.Dense, delta float64) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// L2 = L^2 + delta
	L2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l+delta)
		}
	}

	// pro = exp(rowSums(log(L2))/k)
	pro := make([]float64, p)
	for i := 0; i < p; i++ {
		sumLog := 0.0
		for j := 0; j < k; j++ {
			sumLog += math.Log(L2.At(i, j))
		}
		pro[i] = math.Exp(sumLog / float64(k))
	}

	// f = sum(pro)
	f = 0.0
	for _, v := range pro {
		f += v
	}

	// Gq = (2/k) * (L/L2) * matrix(rep(pro, k), p)
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			gq := (2.0 / float64(k)) * (L.At(i, j) / L2.At(i, j)) * pro[i]
			Gq.Set(i, j, gq)
		}
	}

	method = "Geomin"
	return
}
