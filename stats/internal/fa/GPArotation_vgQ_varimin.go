// fa/GPArotation_vgQ_varimin.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQVarimin computes the objective and gradient for varimin rotation.
// Mirrors GPArotation::vgQ.varimin(L)
//
// QL <- sweep(L^2, 2, colMeans(L^2), "-")
// Gq <- L * QL
// f <- sqrt(sum(diag(crossprod(QL))))^2/4
//
// Returns: Gq (gradient), f (objective), method
func vgQVarimin(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// colMeans
	colMeans := make([]float64, k)
	for j := 0; j < k; j++ {
		sum := 0.0
		for i := 0; i < p; i++ {
			sum += L2.At(i, j)
		}
		colMeans[j] = sum / float64(p)
	}

	// QL = L2 - colMeans
	QL := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			QL.Set(i, j, L2.At(i, j)-colMeans[j])
		}
	}

	// Gq = L * QL
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			Gq.Set(i, j, L.At(i, j)*QL.At(i, j))
		}
	}

	// f = sqrt(sum(diag(crossprod(QL))))^2 / 4 = sum(diag(t(QL)%*%QL)) / 4
	var crossprod mat.Dense
	crossprod.Mul(QL.T(), QL)
	sumDiag := 0.0
	for i := 0; i < k; i++ {
		sumDiag += crossprod.At(i, i)
	}
	f = sumDiag / 4

	method = "varimin"
	return
}
