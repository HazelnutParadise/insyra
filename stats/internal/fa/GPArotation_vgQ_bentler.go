// fa/GPArotation_vgQ_bentler.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQBentler computes the objective and gradient for Bentler's criterion rotation.
// Mirrors GPArotation::vgQ.bentler(L)
//
// Bentler's criterion maximizes the determinant of the factor correlation matrix
// while minimizing the determinants of individual factor variances.
//
// L2 <- L^2
// M <- crossprod(L2) = t(L2) %*% L2  [factor correlation matrix]
// D <- diag(diag(M))  [diagonal matrix of factor variances]
// Gq <- -L * (L2 %*% (solve(M) - solve(D)))
// f <- -(log(det(M)) - log(det(D)))/4
//
// Returns: Gq (gradient), f (objective), method, error
func vgQBentler(L *mat.Dense) (Gq *mat.Dense, f float64, method string, err error) {
	rows, cols := L.Dims()

	// Compute L^2 element-wise
	L2 := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := L.At(i, j)
			L2.Set(i, j, val*val)
		}
	}

	// Compute factor correlation matrix: M = t(L2) %*% L2
	var factorCorr mat.Dense
	factorCorr.Mul(L2.T(), L2)

	// Create diagonal matrix of factor variances: D = diag(diag(M))
	factorVars := mat.NewDense(cols, cols, nil)
	for i := 0; i < cols; i++ {
		factorVars.Set(i, i, factorCorr.At(i, i))
	}

	// Compute inverse of factor correlation matrix
	var invFactorCorr mat.Dense
	if err = invFactorCorr.Inverse(&factorCorr); err != nil {
		return nil, 0, "", err
	}

	// Compute inverse of factor variances matrix
	var invFactorVars mat.Dense
	if err = invFactorVars.Inverse(factorVars); err != nil {
		return nil, 0, "", err
	}

	// Compute difference: inv(M) - inv(D)
	var diff mat.Dense
	diff.Sub(&invFactorCorr, &invFactorVars)

	// Compute L2 %*% diff
	var L2_diff mat.Dense
	L2_diff.Mul(L2, &diff)

	// Gradient: Gq = -L * L2_diff (element-wise)
	Gq = mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			Gq.Set(i, j, -L.At(i, j)*L2_diff.At(i, j))
		}
	}

	// Compute determinants
	detFactorCorr := mat.Det(&factorCorr)
	detFactorVars := mat.Det(factorVars)

	// Objective function: f = -(log(det(M)) - log(det(D)))/4
	f = -(math.Log(detFactorCorr) - math.Log(detFactorVars)) / 4.0

	method = "Bentler's criterion"
	return
}
