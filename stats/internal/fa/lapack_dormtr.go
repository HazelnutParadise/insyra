// fa/lapack_dormtr.go
//
// dormtr — apply the orthogonal Q (from dsytrd's tridiagonal reduction)
// to a matrix C: C := Q*C, Q^T*C, C*Q, or C*Q^T. Used by dsyevr to
// transform tridiagonal-basis eigenvectors back to the original basis.
//
// Implementation strategy: we cannot reuse gonum's Dormqr because gonum
// uses ROW-MAJOR storage (dense[i*ld + j]), while our entire LAPACK port
// is column-major (Fortran convention dense[(j-1)*ld + (i-1)]). So we:
//   1. Build the explicit Q via our dorgtr (column-major, already
//      validated against gonum's EigenSym in dsyev).
//   2. Multiply Q (or Q^T) by C in column-major into a temporary,
//      copy back to C.
package fa

import "math"

// dormtr applies Q from dsytrd to C, in place.
//
//	side  : 'L' (Q on the left) or 'R' (Q on the right)
//	uplo  : 'L' or 'U' (must match dsytrd's uplo)
//	trans : 'N' (apply Q) or 'T' (apply Q^T)
//	m, n  : rows / cols of C
//	a     : the symmetric matrix returned by dsytrd, lda*nq column-major,
//	         where nq = m if side='L' else n. NOT modified.
//	tau   : Householder scalars from dsytrd, length nq-1
//	c     : the matrix to overwrite, ldc*n column-major
//	work  : workspace, length >= nq*nq + nq for the Q construction +
//	        m*n for the multiply temporary.
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
	_ = upper
	_ = math.NaN

	// Build a private copy of A for dorgtr (which destroys its input).
	// We need the lda × nq portion.
	aCopy := make([]float64, lda*nq)
	for j := 0; j < nq; j++ {
		base := j * lda
		copy(aCopy[base:base+nq], a[base:base+nq])
	}
	tauCopy := make([]float64, nq-1)
	copy(tauCopy, tau[:nq-1])

	// dorgtr: produce explicit Q (nq × nq, column-major) into aCopy.
	dorgtr(uplo, nq, aCopy, lda, tauCopy)

	// Now compute C := op(Q) * C (left) or C := C * op(Q) (right),
	// where op(Q) is Q for trans='N' or Q^T for trans='T'.
	// We do the multiply column-by-column into a temp, then copy back.
	transpose := trans == 'T' || trans == 't'

	if left {
		// C is m × n = nq × n. New C[i, j] = sum_k op(Q)[i, k] * C[k, j].
		// op(Q)[i, k] = Q[i, k] if !transpose else Q[k, i].
		tmp := make([]float64, nq)
		for j := 0; j < n; j++ {
			cCol := c[j*ldc : j*ldc+nq]
			for i := 0; i < nq; i++ {
				s := 0.0
				if transpose {
					qCol := aCopy[i*lda : i*lda+nq] // column i of Q
					for k := 0; k < nq; k++ {
						s += qCol[k] * cCol[k]
					}
				} else {
					for k := 0; k < nq; k++ {
						s += aCopy[k*lda+i] * cCol[k]
					}
				}
				tmp[i] = s
			}
			copy(cCol, tmp)
		}
	} else {
		// C is m × n; nq = n. New C[i, j] = sum_k C[i, k] * op(Q)[k, j].
		tmp := make([]float64, n)
		for i := 0; i < m; i++ {
			for j := 0; j < n; j++ {
				s := 0.0
				if transpose {
					// op(Q)[k, j] = Q[j, k]
					for k := 0; k < n; k++ {
						s += c[k*ldc+i] * aCopy[k*lda+j]
					}
				} else {
					for k := 0; k < n; k++ {
						s += c[k*ldc+i] * aCopy[j*lda+k]
					}
				}
				tmp[j] = s
			}
			for j := 0; j < n; j++ {
				c[j*ldc+i] = tmp[j]
			}
		}
	}

	work[0] = 1
	return 0
}
