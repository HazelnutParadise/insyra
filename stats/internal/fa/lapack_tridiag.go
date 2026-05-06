// fa/lapack_tridiag.go
//
// Tridiagonal-reduction layer for the dsyevr port:
//
//   dsymv  — symmetric matrix-vector multiply (BLAS-2)
//   dsyr2  — symmetric rank-2 update         (BLAS-2)
//   dlatrd — reduce nb panel of A to tridiagonal form (LAPACK aux)
//   dsytrd — reduce A to symmetric tridiagonal T = Q^T A Q (LAPACK)
//   dorgtr — generate the orthogonal Q from dsytrd's reflectors
//   dormtr — apply Q (or Q^T) to a matrix
//
// Faithful translations of LAPACK 3.12.1 reference Fortran. We use the
// UNBLOCKED path throughout (NB=1) since our matrices are small (n < 50)
// and blocking only matters for cache-friendly large-n performance.
// Numerical results of unblocked == blocked for symmetric tridiag
// reduction (the only difference is BLAS-3 vs BLAS-2 call mix).
//
// All matrices are column-major flat []float64; only the upper
// triangle is referenced when uplo='U', only the lower when uplo='L'.
package fa

// dsymv computes y := alpha*A*x + beta*y where A is n×n symmetric.
// Reference BLAS dsymv. uplo='U' references upper triangle, 'L' lower.
func dsymv(uplo byte, n int, alpha float64, a []float64, lda int,
	x []float64, incx int, beta float64, y []float64, incy int,
) {
	if n == 0 || (alpha == 0 && beta == 1) {
		return
	}
	kx := 0
	if incx < 0 {
		kx = -(n - 1) * incx
	}
	ky := 0
	if incy < 0 {
		ky = -(n - 1) * incy
	}
	// y := beta * y
	if beta != 1 {
		iy := ky
		for i := 0; i < n; i++ {
			if beta == 0 {
				y[iy] = 0
			} else {
				y[iy] *= beta
			}
			iy += incy
		}
	}
	if alpha == 0 {
		return
	}
	switch uplo {
	case 'U', 'u':
		// Upper triangle.
		jx := kx
		jy := ky
		for j := 0; j < n; j++ {
			temp1 := alpha * x[jx]
			temp2 := 0.0
			ix := kx
			iy := ky
			for i := 0; i < j; i++ {
				y[iy] += temp1 * a[j*lda+i]
				temp2 += a[j*lda+i] * x[ix]
				ix += incx
				iy += incy
			}
			// Reference BLAS dsymv left-fold:
			// y(j) = y(j) + temp1*a(j,j) + alpha*temp2
			y[jy] += temp1 * a[j*lda+j]
			y[jy] += alpha * temp2
			jx += incx
			jy += incy
		}
	case 'L', 'l':
		jx := kx
		jy := ky
		for j := 0; j < n; j++ {
			temp1 := alpha * x[jx]
			temp2 := 0.0
			y[jy] += temp1 * a[j*lda+j]
			ix := jx
			iy := jy
			for i := j + 1; i < n; i++ {
				ix += incx
				iy += incy
				y[iy] += temp1 * a[j*lda+i]
				temp2 += a[j*lda+i] * x[ix]
			}
			y[jy] += alpha * temp2
			jx += incx
			jy += incy
		}
	}
}

// dsyr2 computes A := alpha*x*y^T + alpha*y*x^T + A (symmetric rank-2).
// Reference BLAS dsyr2. Updates only upper or lower triangle.
func dsyr2(uplo byte, n int, alpha float64, x []float64, incx int,
	y []float64, incy int, a []float64, lda int,
) {
	if n == 0 || alpha == 0 {
		return
	}
	kx := 0
	if incx < 0 {
		kx = -(n - 1) * incx
	}
	ky := 0
	if incy < 0 {
		ky = -(n - 1) * incy
	}
	jx := kx
	jy := ky
	switch uplo {
	case 'U', 'u':
		for j := 0; j < n; j++ {
			if x[jx] != 0 || y[jy] != 0 {
				temp1 := alpha * y[jy]
				temp2 := alpha * x[jx]
				ix := kx
				iy := ky
				for i := 0; i <= j; i++ {
					// Reference BLAS dsyr2 left-fold:
					// a(i,j) = a(i,j) + x(i)*temp1 + y(i)*temp2
					a[j*lda+i] += x[ix] * temp1
					a[j*lda+i] += y[iy] * temp2
					ix += incx
					iy += incy
				}
			}
			jx += incx
			jy += incy
		}
	case 'L', 'l':
		for j := 0; j < n; j++ {
			if x[jx] != 0 || y[jy] != 0 {
				temp1 := alpha * y[jy]
				temp2 := alpha * x[jx]
				ix := jx
				iy := jy
				for i := j; i < n; i++ {
					// Reference BLAS dsyr2 left-fold:
					// a(i,j) = a(i,j) + x(i)*temp1 + y(i)*temp2
					a[j*lda+i] += x[ix] * temp1
					a[j*lda+i] += y[iy] * temp2
					ix += incx
					iy += incy
				}
			}
			jx += incx
			jy += incy
		}
	}
}

// dsytrd reduces a real symmetric matrix A to tridiagonal form T:
//
//	A = Q * T * Q^T
//
// where Q is a product of (n-1) Householder reflectors. Mirrors
// LAPACK dsytrd.f. Unblocked version (NB=1) — for n < 50 the blocked
// path gives identical numerical results.
//
// On entry, a holds the original symmetric matrix; on exit:
//   - d holds the diagonal of T
//   - e holds the off-diagonal of T (length n-1)
//   - tau holds the Householder scalars (length n-1)
//   - the upper or lower triangle of a holds the Householder reflectors
func dsytrd(uplo byte, n int, a []float64, lda int, d, e, tau []float64) {
	if n <= 0 {
		return
	}
	upper := uplo == 'U' || uplo == 'u'
	w := make([]float64, n)
	if upper {
		// Use unblocked code: reduce columns from N down to 2.
		for i := n; i >= 2; i-- {
			alpha := a[(i-1)*lda+(i-2)]
			tauI := dlarfg(i-1, &alpha, a[(i-1)*lda:], 1)
			a[(i-1)*lda+(i-2)] = alpha
			e[i-2] = a[(i-1)*lda+(i-2)]
			a[(i-1)*lda+(i-2)] = 1
			// Apply H from both sides: A := H * A * H
			// p := tau * A * v
			dsymv('U', i-1, tauI, a, lda, a[(i-1)*lda:], 1, 0, w, 1)
			alphaW := -0.5 * tauI * ddot(i-1, w, 1, a[(i-1)*lda:], 1)
			daxpy(i-1, alphaW, a[(i-1)*lda:], 1, w, 1)
			// A := A - v * w^T - w * v^T
			dsyr2('U', i-1, -1, a[(i-1)*lda:], 1, w, 1, a, lda)
			a[(i-1)*lda+(i-2)] = e[i-2]
			d[i-1] = a[(i-1)*lda+(i-1)]
			tau[i-2] = tauI
		}
		d[0] = a[0]
	} else {
		// Lower triangle: reduce columns from 1 to N-1.
		for i := 1; i <= n-1; i++ {
			ip2 := i + 2
			if ip2 > n {
				ip2 = n
			}
			alpha := a[(i-1)*lda+i]
			tauI := dlarfg(n-i, &alpha, a[(i-1)*lda+(ip2-1):], 1)
			a[(i-1)*lda+i] = alpha
			e[i-1] = a[(i-1)*lda+i]
			a[(i-1)*lda+i] = 1
			dsymv('L', n-i, tauI, a[i*lda+i:], lda,
				a[(i-1)*lda+i:], 1, 0, w[i:], 1)
			alphaW := -0.5 * tauI * ddot(n-i, w[i:], 1, a[(i-1)*lda+i:], 1)
			daxpy(n-i, alphaW, a[(i-1)*lda+i:], 1, w[i:], 1)
			dsyr2('L', n-i, -1, a[(i-1)*lda+i:], 1, w[i:], 1,
				a[i*lda+i:], lda)
			a[(i-1)*lda+i] = e[i-1]
			d[i-1] = a[(i-1)*lda+(i-1)]
			tau[i-1] = tauI
		}
		d[n-1] = a[(n-1)*lda+(n-1)]
	}
}

// dorgtr generates the orthogonal matrix Q from dsytrd's reflectors.
// Mirrors LAPACK dorgtr.f (using the unblocked dorg2l/dorg2r path).
func dorgtr(uplo byte, n int, a []float64, lda int, tau []float64) {
	if n == 0 {
		return
	}
	upper := uplo == 'U' || uplo == 'u'
	if upper {
		// Q was formed using H(1) ... H(n-1) on the upper triangle.
		// Shift the vectors: a[i, i+1] becomes a[i, i] etc.
		for j := 0; j < n-1; j++ {
			for i := 0; i < j; i++ {
				a[j*lda+i] = a[(j+1)*lda+i]
			}
			a[j*lda+(n-1)] = 0
		}
		for i := 0; i < n-1; i++ {
			a[(n-1)*lda+i] = 0
		}
		a[(n-1)*lda+(n-1)] = 1
		// Generate Q(1:n-1, 1:n-1) using dorg2l.
		dorg2l(n-1, n-1, n-1, a, lda, tau)
	} else {
		// Lower: shift Householder vectors RIGHT (column j-1 → column j).
		// This mirrors gonum's row-major Dorgtr (verified correct via tests
		// in TestDormtrAgainstExplicitQ etc.). The LAPACK 3.12 reference
		// shifts left when reading column-major; experimentation shows this
		// loses v_0's data. The correct shift for column-major preserving
		// gonum semantics is shift-right.
		for j := n - 1; j >= 1; j-- {
			a[j*lda+0] = 0
			for i := j + 1; i < n; i++ {
				a[j*lda+i] = a[(j-1)*lda+i]
			}
		}
		a[0] = 1
		for i := 1; i < n; i++ {
			a[0*lda+i] = 0
		}
		if n > 1 {
			// Generate Q(2:n, 2:n) using dorg2r.
			dorg2r(n-1, n-1, n-1, a[1*lda+1:], lda, tau)
		}
	}
}

// dorg2l generates an m×n real matrix Q with orthonormal columns from
// the elementary reflectors stored in the trailing n columns of A.
// Used by dorgtr UPLO='U'. Mirrors LAPACK dorg2l.f.
func dorg2l(m, n, k int, a []float64, lda int, tau []float64) {
	if n <= 0 {
		return
	}
	work := make([]float64, n)
	// Initialize columns k+1..n to columns of the unit matrix.
	for j := 0; j < n-k; j++ {
		for l := 0; l < m; l++ {
			a[j*lda+l] = 0
		}
		a[j*lda+(m-n+j)] = 1
	}
	for i := 1; i <= k; i++ {
		ii := n - k + i
		// Apply H(i) to A(1:m-k+i, 1:n-k+i-1) from the left.
		a[(ii-1)*lda+(m-n+ii-1)] = 1
		dlarf('L', m-n+ii, ii-1, a[(ii-1)*lda:], 1, tau[k-i],
			a, lda, work)
		dscal(m-n+ii-1, -tau[k-i], a[(ii-1)*lda:], 1)
		a[(ii-1)*lda+(m-n+ii-1)] = 1 - tau[k-i]
		// Set A(m-k+i+1:m, ii) = 0
		for l := m - n + ii; l < m; l++ {
			a[(ii-1)*lda+l] = 0
		}
	}
}

// dorg2r generates an m×n real matrix Q with orthonormal columns from
// elementary reflectors stored in the leading n columns of A. Used by
// dorgtr UPLO='L'. Mirrors LAPACK dorg2r.f.
func dorg2r(m, n, k int, a []float64, lda int, tau []float64) {
	if n <= 0 {
		return
	}
	work := make([]float64, n)
	// Initialize columns k+1..n.
	for j := k; j < n; j++ {
		for l := 0; l < m; l++ {
			a[j*lda+l] = 0
		}
		a[j*lda+j] = 1
	}
	for i := k - 1; i >= 0; i-- {
		// Apply H(i+1) from the left to A(i:m, i:n).
		if i+1 < n {
			a[i*lda+i] = 1
			dlarf('L', m-i, n-i-1, a[i*lda+i:], 1, tau[i],
				a[(i+1)*lda+i:], lda, work)
		}
		if i+1 < m {
			dscal(m-i-1, -tau[i], a[i*lda+i+1:], 1)
		}
		a[i*lda+i] = 1 - tau[i]
		// Set A(0:i-1, i) to zero.
		for l := 0; l < i; l++ {
			a[i*lda+l] = 0
		}
	}
}
