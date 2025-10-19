// fa/GPArotation_vgQ_varimax.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// vgQVarimax computes the objective and gradient for varimax rotation.
// Mirrors GPArotation::vgQ.varimax(L)
//
// Varimax criterion maximizes the variance of squared loadings within each factor:
// QL <- L^2 - colMeans(L^2)  [center each column]
// f <- -sum(diag(t(QL) %*% QL)) / 4  [negative because GPA minimizes]
// Gq <- -L * QL
//
// Returns: Gq (gradient), f (objective), method
func vgQVarimax(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	rows, cols := L.Dims()

	// Compute L^2 element-wise
	L2 := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := L.At(i, j)
			L2.Set(i, j, val*val)
		}
	}

	// Compute column means of L2
	colMeans := make([]float64, cols)
	for j := 0; j < cols; j++ {
		sum := 0.0
		for i := 0; i < rows; i++ {
			sum += L2.At(i, j)
		}
		colMeans[j] = sum / float64(rows)
	}

	// Center L2 by subtracting column means: QL = L2 - colMeans
	QL := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			QL.Set(i, j, L2.At(i, j)-colMeans[j])
		}
	}

	// Compute cross product: t(QL) %*% QL
	var crossProd mat.Dense
	crossProd.Mul(QL.T(), QL)

	// Sum of diagonal elements of cross product
	sumDiag := 0.0
	for i := 0; i < cols; i++ {
		sumDiag += crossProd.At(i, i)
	}

	// Objective function: f = -sum(diag(t(QL)%*%QL)) / 4
	// (negative because GPA minimizes, but varimax maximizes)
	f = -sumDiag / 4.0

	// Gradient: Gq = -L * QL (element-wise multiplication)
	Gq = mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			Gq.Set(i, j, -L.At(i, j)*QL.At(i, j))
		}
	}

	method = "varimax"
	return
}
