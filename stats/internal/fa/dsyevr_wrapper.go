// fa/dsyevr_wrapper.go
//
// Adapter that exposes our pure-Go dsyevr implementation with the same
// interface as statslinalg.SymmetricEigenDescending — eigenvalues in
// DESCENDING order, eigenvectors as columns of a Dense matrix.
package fa

import "gonum.org/v1/gonum/mat"

// SymmetricEigenDescendingDsyevr is the exported wrapper. R-bit-perfect
// alternative to statslinalg.SymmetricEigenDescending. Use this for
// downstream stages (e.g. inverse-symmetric-sqrt in Anderson-Rubin
// scoring) that must agree with R's eigen() at ULP level.
func SymmetricEigenDescendingDsyevr(a mat.Matrix) ([]float64, *mat.Dense, bool) {
	return symmetricEigenDescendingDsyevr(a)
}

// symmetricEigenDescendingDsyevr computes eigenvalues+eigenvectors of a
// symmetric matrix using the LAPACK MRRR algorithm (dsyevr), returning
// them sorted from largest to smallest (matching R's eigen()).
//
// This bypasses gonum's mat.EigenSym (which uses dsyev / QL+QR) and
// follows R's eigen() / dsyevr computational path.
func symmetricEigenDescendingDsyevr(a mat.Matrix) ([]float64, *mat.Dense, bool) {
	rows, cols := a.Dims()
	if rows != cols {
		return nil, nil, false
	}
	n := rows

	// Pack matrix into column-major flat (lda = n), symmetrising.
	aMine := make([]float64, n*n)
	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			v := 0.5 * (a.At(i, j) + a.At(j, i))
			aMine[j*n+i] = v
			if i != j {
				aMine[i*n+j] = v
			}
		}
	}

	w := make([]float64, n)
	z := make([]float64, n*n)
	isuppz := make([]int, 2*n)
	work := make([]float64, 26*n)
	iwork := make([]int, 10*n)

	m, info := dsyevr('V', 'A', 'L', n, aMine, n,
		0, 0, 0, 0, 0,
		w, z, n, isuppz, work, iwork)
	if info != 0 || m != n {
		return nil, nil, false
	}

	// dsyevr returns ascending; reverse to descending.
	values := make([]float64, n)
	vectors := mat.NewDense(n, n, nil)
	for j := 0; j < n; j++ {
		src := n - 1 - j
		values[j] = w[src]
		// z is column-major flat: column src starts at z[src*n].
		for i := 0; i < n; i++ {
			vectors.Set(i, j, z[src*n+i])
		}
	}
	return values, vectors, true
}
