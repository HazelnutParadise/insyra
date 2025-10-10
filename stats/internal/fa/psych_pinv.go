// fa/psych_pinv.go
package fa

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

// Pinv computes the Moore-Penrose pseudo-inverse of a matrix.
// Mirrors psych::Pinv exactly
func Pinv(X *mat.Dense, tol float64) (*mat.Dense, error) {
	m, n := X.Dims()

	// svdX <- svd(X)
	var svd mat.SVD
	ok := svd.Factorize(X, mat.SVDThin)
	if !ok {
		return nil, fmt.Errorf("SVD factorization failed")
	}

	var U mat.Dense
	var V mat.Dense
	svd.UTo(&U)
	svd.VTo(&V)
	values := svd.Values(nil)

	// If tol is not provided or negative, use default
	if tol <= 0 {
		tol = math.Sqrt(2.220446e-16) // .Machine$double.eps in R
	}

	// p <- svdX$d > max(tol * svdX$d[1], 0)
	maxTol := tol * values[0]
	p := make([]bool, len(values))
	for i, val := range values {
		p[i] = val > maxTol
	}

	// Count true values in p
	pCount := 0
	for _, val := range p {
		if val {
			pCount++
		}
	}

	if pCount == 0 {
		return nil, fmt.Errorf("matrix is too singular, no singular values above tolerance")
	}

	if pCount == len(values) {
		// Pinv <- svdX$v %*% (1/svdX$d * t(svdX$u))
		// Create diagonal matrix with 1/values
		D := mat.NewDense(len(values), len(values), nil)
		for i := 0; i < len(values); i++ {
			D.Set(i, i, 1.0/values[i])
		}

		var Ut mat.Dense
		Ut.CloneFrom(&U)
		Ut.T()

		var temp mat.Dense
		temp.Mul(D, &Ut)

		var Pinv mat.Dense
		Pinv.Mul(&V, &temp)
		return &Pinv, nil
	} else {
		// Pinv <- svdX$v[, p, drop = FALSE] %*% (1/svdX$d[p] * t(svdX$u[, p, drop = FALSE]))
		// Extract columns where p is true
		Vp := mat.NewDense(n, pCount, nil)
		Up := mat.NewDense(m, pCount, nil)
		valuesP := make([]float64, pCount)

		colIdx := 0
		for i, isP := range p {
			if isP {
				for row := 0; row < n; row++ {
					Vp.Set(row, colIdx, V.At(row, i))
				}
				for row := 0; row < m; row++ {
					Up.Set(row, colIdx, U.At(row, i))
				}
				valuesP[colIdx] = values[i]
				colIdx++
			}
		}

		// Create diagonal matrix with 1/valuesP
		D := mat.NewDense(pCount, pCount, nil)
		for i := 0; i < pCount; i++ {
			D.Set(i, i, 1.0/valuesP[i])
		}

		var Upt mat.Dense
		Upt.CloneFrom(Up)
		Upt.T()

		var temp mat.Dense
		temp.Mul(D, &Upt)

		var Pinv mat.Dense
		Pinv.Mul(Vp, &temp)
		return &Pinv, nil
	}
}
