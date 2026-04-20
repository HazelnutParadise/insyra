package linalg

import "math"

// GaussianElimination solves Ax=b using Gaussian elimination with partial pivoting.
func GaussianElimination(A [][]float64, b []float64) []float64 {
	n := len(A)
	if n == 0 || len(b) != n {
		return nil
	}

	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, n+1)
		copy(aug[i][:n], A[i])
		aug[i][n] = b[i]
	}

	for i := range n {
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(aug[k][i]) > math.Abs(aug[maxRow][i]) {
				maxRow = k
			}
		}

		aug[i], aug[maxRow] = aug[maxRow], aug[i]

		if math.Abs(aug[i][i]) < 1e-12 {
			return nil
		}

		for k := i + 1; k < n; k++ {
			factor := aug[k][i] / aug[i][i]
			for j := i; j <= n; j++ {
				aug[k][j] -= factor * aug[i][j]
			}
		}
	}

	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		x[i] = aug[i][n]
		for j := i + 1; j < n; j++ {
			x[i] -= aug[i][j] * x[j]
		}
		x[i] /= aug[i][i]
	}

	return x
}

// InvertMatrix computes a matrix inverse via Gauss-Jordan elimination.
func InvertMatrix(A [][]float64) [][]float64 {
	n := len(A)
	if n == 0 {
		return nil
	}

	aug := make([][]float64, n)
	for i := range aug {
		aug[i] = make([]float64, 2*n)
		copy(aug[i][:n], A[i])
		aug[i][n+i] = 1.0
	}

	for i := range n {
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(aug[k][i]) > math.Abs(aug[maxRow][i]) {
				maxRow = k
			}
		}

		aug[i], aug[maxRow] = aug[maxRow], aug[i]
		if math.Abs(aug[i][i]) < 1e-12 {
			return nil
		}

		pivot := aug[i][i]
		for j := 0; j < 2*n; j++ {
			aug[i][j] /= pivot
		}

		for k := 0; k < n; k++ {
			if k == i {
				continue
			}
			factor := aug[k][i]
			for j := 0; j < 2*n; j++ {
				aug[k][j] -= factor * aug[i][j]
			}
		}
	}

	inv := make([][]float64, n)
	for i := range inv {
		inv[i] = make([]float64, n)
		copy(inv[i], aug[i][n:])
	}
	return inv
}

// DeterminantGauss computes determinant via Gaussian elimination with partial pivoting.
func DeterminantGauss(matrix [][]float64) float64 {
	n := len(matrix)
	if n == 1 {
		return matrix[0][0]
	}
	if n == 2 {
		return matrix[0][0]*matrix[1][1] - matrix[0][1]*matrix[1][0]
	}

	A := make([][]float64, n)
	for i := range matrix {
		A[i] = make([]float64, n)
		copy(A[i], matrix[i])
	}

	det := 1.0
	for i := 0; i < n; i++ {
		maxRow := i
		for j := i + 1; j < n; j++ {
			if math.Abs(A[j][i]) > math.Abs(A[maxRow][i]) {
				maxRow = j
			}
		}

		if math.Abs(A[maxRow][i]) < 1e-10 {
			return 0.0
		}

		if maxRow != i {
			A[i], A[maxRow] = A[maxRow], A[i]
			det *= -1
		}

		det *= A[i][i]
		for j := i + 1; j < n; j++ {
			factor := A[j][i] / A[i][i]
			for k := i; k < n; k++ {
				A[j][k] -= factor * A[i][k]
			}
		}
	}

	return det
}
