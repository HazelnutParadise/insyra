// fa/lapack_dormtr.go
//
// dormtr — apply the orthogonal Q (from dsytrd's tridiagonal reduction)
// to a matrix C: C := Q*C, Q^T*C, C*Q, or C*Q^T. Used by dsyevr to
// transform tridiagonal-basis eigenvectors back to the original basis.
//
// The reference LAPACK implementation dispatches to dormql (UPLO='U')
// or dormqr (UPLO='L'). Our implementation:
//   - UPLO='L': uses gonum's Dormqr directly (faithful BLAS-level ports)
//   - UPLO='U': panics — not yet implemented; not needed by our
//     factor-analysis pipeline (we always call dsytrd/dormtr with 'L').
package fa

import (
	"gonum.org/v1/gonum/blas"
	"gonum.org/v1/gonum/lapack/gonum"
)

// dormtr applies Q from dsytrd to C.
//
//	side  : 'L' (Q on the left) or 'R' (Q on the right)
//	uplo  : must match the uplo passed to dsytrd
//	trans : 'N' (apply Q) or 'T' (apply Q^T)
//	m, n  : rows / cols of C
//	a     : the symmetric matrix returned by dsytrd, lda*nq column-major,
//	         where nq = m if side='L' else n
//	tau   : Householder scalars from dsytrd, length nq-1
//	c     : the matrix to overwrite, ldc*n column-major
//	work  : workspace, length >= max(1, n) for side='L' or max(1, m) for side='R'
func dormtr(
	side, uplo, trans byte,
	m, n int,
	a []float64,
	lda int,
	tau, c []float64,
	ldc int,
	work []float64,
) (info int) {
	left := side == 'L' || side == 'l'
	upper := uplo == 'U' || uplo == 'u'

	var nq int
	if left {
		nq = m
	} else {
		nq = n
	}
	if m == 0 || n == 0 || nq == 1 {
		work[0] = 1
		return 0
	}

	if upper {
		panic("fa: dormtr UPLO='U' not implemented (port Dormql to enable)")
	}

	var mi, ni int
	if left {
		mi = m
		ni = n
	} else {
		mi = m
		ni = n
	}
	_ = mi
	_ = ni

	var sideBLAS blas.Side
	if left {
		sideBLAS = blas.Left
	} else {
		sideBLAS = blas.Right
	}
	var transBLAS blas.Transpose
	if trans == 'N' || trans == 'n' {
		transBLAS = blas.NoTrans
	} else {
		transBLAS = blas.Trans
	}

	// LAPACK dormtr (UPLO='L') reduced sizes:
	if left {
		mi = m - 1
		ni = n
	} else {
		mi = m
		ni = n - 1
	}

	// Shift A by one row: A(2:NQ, 1:NQ-1) — flat index (j-1)*lda + 1 for column j.
	// In Go we just pass a[1:] and it is column-major with the same lda.
	aShifted := a[1:]

	// Shift C: for SIDE='L' use C(2:M, :), i.e. c[1:].
	//          for SIDE='R' use C(:, 2:N), i.e. c[ldc:].
	var cShifted []float64
	if left {
		cShifted = c[1:]
	} else {
		cShifted = c[ldc:]
	}

	var impl gonum.Implementation
	// gonum.Dormqr(side, trans, m, n, k, a, lda, tau, c, ldc, work, lwork)
	// k = NQ-1
	impl.Dormqr(sideBLAS, transBLAS, mi, ni, nq-1, aShifted, lda, tau, cShifted, ldc, work, len(work))
	work[0] = 1
	return 0
}
