// fa/GPArotation_vgQ_geomin.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQGeomin computes the objective and gradient for geomin rotation.
// Mirrors GPArotation::vgQ.geomin(L, delta = 0.01)
//
// Geomin minimizes the product of squared loadings within each row (variable).
// It uses a small epsilon (delta) to avoid log(0) problems.
//
// k <- ncol(L)  [number of factors]
// p <- nrow(L)  [number of variables]
// L2 <- L^2 + delta
// pro <- exp(rowSums(log(L2))/k)  [geometric mean for each variable]
// Gq <- (2/k) * (L/L2) * matrix(rep(pro, k), p)
// f <- sum(pro)
//
// Returns: Gq (gradient), f (objective), method
func vgQGeomin(L *mat.Dense, delta float64) (Gq *mat.Dense, f float64, method string) {
	rows, cols := L.Dims()

	// L2 = L^2 + delta (add small epsilon to avoid log(0))
	L2 := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := L.At(i, j)
			L2.Set(i, j, val*val+delta)
		}
	}

	// Compute log(L2) element-wise
	logL2 := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			logL2.Set(i, j, math.Log(L2.At(i, j)))
		}
	}

	// Compute row sums of log(L2)
	rowSumsLog := make([]float64, rows)
	for i := 0; i < rows; i++ {
		sum := 0.0
		for j := 0; j < cols; j++ {
			sum += logL2.At(i, j)
		}
		rowSumsLog[i] = sum
	}

	// Compute geometric mean for each row: pro = exp(rowSums(log(L2)) / k)
	geometricMeans := make([]float64, rows)
	for i := 0; i < rows; i++ {
		geometricMeans[i] = math.Exp(rowSumsLog[i] / float64(cols))
	}

	// Objective function: f = sum(pro)
	f = 0.0
	for i := 0; i < rows; i++ {
		f += geometricMeans[i]
	}

	// Compute L / L2 element-wise
	L_over_L2 := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			L_over_L2.Set(i, j, L.At(i, j)/L2.At(i, j))
		}
	}

	// Create matrix by repeating geometric means across columns
	geomMeanMat := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			geomMeanMat.Set(i, j, geometricMeans[i])
		}
	}

	// Gradient: Gq = (2/k) * (L/L2) * geomMeanMat (element-wise)
	Gq = mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			Gq.Set(i, j, (2.0/float64(cols))*L_over_L2.At(i, j)*geomMeanMat.At(i, j))
		}
	}

	method = "Geomin"
	return
}
