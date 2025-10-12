// fa/GPArotation_vgQ_varimax.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQVarimax computes the objective and gradient for varimax rotation.
// Mirrors GPArotation::vgQ.varimax(L)
//
// QL <- sweep(L^2, 2, colMeans(L^2), "-")
// f <- -sqrt(sum(diag(crossprod(QL))))^2 / 4
// Gq <- -L * QL
//
// Returns: Gq (gradient), f (objective), method
func vgQVarimax(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, q := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, q, nil)
	for i := range p {
		for j := 0; j < q; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// colMeans
	colMeans := make([]float64, q)
	for j := range q {
		sum := 0.0
		for i := range p {
			sum += L2.At(i, j)
		}
		colMeans[j] = sum / float64(p)
	}

	// QL = L2 - colMeans
	QL := mat.NewDense(p, q, nil)
	for i := range p {
		for j := 0; j < q; j++ {
			QL.Set(i, j, L2.At(i, j)-colMeans[j])
		}
	}

	// crossprod = t(QL) %*% QL
	var crossprod mat.Dense
	crossprod.Mul(QL.T(), QL)

	// sumDiag = sum(diag(crossprod))
	sumDiag := 0.0
	for i := range q {
		sumDiag += crossprod.At(i, i)
	}

	// f = - (sqrt(sumDiag))^2 / 4
	f = -math.Pow(math.Sqrt(sumDiag), 2) / 4

	// Gq = -L * QL
	Gq = mat.NewDense(p, q, nil)
	for i := range p {
		for j := range q {
			Gq.Set(i, j, -L.At(i, j)*QL.At(i, j))
		}
	}

	method = "varimax"
	return
}
