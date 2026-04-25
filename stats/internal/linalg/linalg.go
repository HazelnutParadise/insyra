package linalg

import (
	"math"
	"sort"

	"gonum.org/v1/gonum/mat"
)

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

// IdentityDense returns an n by n identity matrix.
func IdentityDense(n int) *mat.Dense {
	identity := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		identity.Set(i, i, 1)
	}
	return identity
}

// InverseOrIdentityDense returns the inverse of m, or identity if inversion fails.
func InverseOrIdentityDense(m *mat.Dense, n int) *mat.Dense {
	if m == nil {
		return IdentityDense(n)
	}
	var inv mat.Dense
	if err := inv.Inverse(m); err != nil {
		return IdentityDense(n)
	}
	return mat.DenseCopyOf(&inv)
}

// CorrelationDense computes a Pearson correlation matrix from complete dense data.
func CorrelationDense(data *mat.Dense) *mat.Dense {
	n, p := data.Dims()
	corr := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			if i == j {
				corr.Set(i, j, 1)
				continue
			}
			meanI, meanJ := 0.0, 0.0
			for k := 0; k < n; k++ {
				meanI += data.At(k, i)
				meanJ += data.At(k, j)
			}
			meanI /= float64(n)
			meanJ /= float64(n)

			varI, varJ, cov := 0.0, 0.0, 0.0
			for k := 0; k < n; k++ {
				devI := data.At(k, i) - meanI
				devJ := data.At(k, j) - meanJ
				varI += devI * devI
				varJ += devJ * devJ
				cov += devI * devJ
			}
			if varI > 0 && varJ > 0 {
				corr.Set(i, j, cov/math.Sqrt(varI*varJ))
			}
		}
	}
	return corr
}

// CovarianceDense computes a sample covariance matrix from complete dense data.
func CovarianceDense(data *mat.Dense) *mat.Dense {
	n, p := data.Dims()
	cov := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			meanI, meanJ := 0.0, 0.0
			for k := 0; k < n; k++ {
				meanI += data.At(k, i)
				meanJ += data.At(k, j)
			}
			meanI /= float64(n)
			meanJ /= float64(n)

			covVal := 0.0
			for k := 0; k < n; k++ {
				covVal += (data.At(k, i) - meanI) * (data.At(k, j) - meanJ)
			}
			cov.Set(i, j, covVal/float64(n-1))
		}
	}
	return cov
}

// CorrelationDensePairwise computes Pearson correlations using pairwise complete observations.
func CorrelationDensePairwise(data *mat.Dense) *mat.Dense {
	n, p := data.Dims()
	corr := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			if i == j {
				corr.Set(i, j, 1)
				continue
			}
			validPairs := 0
			sumI, sumJ, sumI2, sumJ2, sumIJ := 0.0, 0.0, 0.0, 0.0, 0.0
			for k := 0; k < n; k++ {
				valI, valJ := data.At(k, i), data.At(k, j)
				if !math.IsNaN(valI) && !math.IsNaN(valJ) {
					validPairs++
					sumI += valI
					sumJ += valJ
					sumI2 += valI * valI
					sumJ2 += valJ * valJ
					sumIJ += valI * valJ
				}
			}
			if validPairs > 1 {
				meanI := sumI / float64(validPairs)
				meanJ := sumJ / float64(validPairs)
				varI := (sumI2 - float64(validPairs)*meanI*meanI) / float64(validPairs-1)
				varJ := (sumJ2 - float64(validPairs)*meanJ*meanJ) / float64(validPairs-1)
				cov := (sumIJ - float64(validPairs)*meanI*meanJ) / float64(validPairs-1)
				if varI > 0 && varJ > 0 {
					corr.Set(i, j, cov/math.Sqrt(varI*varJ))
				}
			}
		}
	}
	return corr
}

// CovarianceDensePairwise computes sample covariance using pairwise complete observations.
func CovarianceDensePairwise(data *mat.Dense) *mat.Dense {
	n, p := data.Dims()
	cov := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			validPairs := 0
			sumI, sumJ, sumIJ := 0.0, 0.0, 0.0
			for k := 0; k < n; k++ {
				valI, valJ := data.At(k, i), data.At(k, j)
				if !math.IsNaN(valI) && !math.IsNaN(valJ) {
					validPairs++
					sumI += valI
					sumJ += valJ
					sumIJ += valI * valJ
				}
			}
			if validPairs > 1 {
				meanI := sumI / float64(validPairs)
				meanJ := sumJ / float64(validPairs)
				cov.Set(i, j, (sumIJ-float64(validPairs)*meanI*meanJ)/float64(validPairs-1))
			}
		}
	}
	return cov
}

// SymmetricEigenDescending computes eigenvalues/eigenvectors for a symmetric
// matrix and returns them sorted from largest to smallest, matching R's eigen().
func SymmetricEigenDescending(a mat.Matrix) ([]float64, *mat.Dense, bool) {
	rows, cols := a.Dims()
	if rows != cols {
		return nil, nil, false
	}

	sym := mat.NewSymDense(rows, nil)
	for i := 0; i < rows; i++ {
		for j := i; j < rows; j++ {
			sym.SetSym(i, j, 0.5*(a.At(i, j)+a.At(j, i)))
		}
	}

	var eig mat.EigenSym
	if !eig.Factorize(sym, true) {
		return nil, nil, false
	}

	values := eig.Values(nil)
	vectors := mat.NewDense(rows, rows, nil)
	eig.VectorsTo(vectors)

	order := make([]int, rows)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		return values[order[i]] > values[order[j]]
	})

	sortedValues := make([]float64, rows)
	sortedVectors := mat.NewDense(rows, rows, nil)
	for newCol, oldCol := range order {
		sortedValues[newCol] = values[oldCol]
		for row := 0; row < rows; row++ {
			sortedVectors.Set(row, newCol, vectors.At(row, oldCol))
		}
	}

	return sortedValues, sortedVectors, true
}
