// fa/lbfgsb_blas.go
//
// Faithful port of the BLAS-1 and LINPACK routines used by Byrd-Lu-
// Nocedal-Zhu L-BFGS-B (lbfgsb.f v3.0, Netlib). The port preserves
// Fortran's 1-based indexing through explicit `-1` adjustments at the
// call sites; index arguments inside these primitives stay Fortran-style
// for line-by-line review against the reference implementation.
//
// All matrix arguments are Fortran column-major: element (i, j) of a
// matrix `a` declared `a(lda, *)` lives at `a[(j-1)*lda + (i-1)]` from
// the start of the slice. Stride/`incx` arguments of BLAS routines are
// honored exactly to match the original behavior.
//
// Source: lbfgsb 3.0 by Ciyou Zhu, Richard Byrd, Jorge Nocedal and Jose
// Luis Morales (3-clause BSD).
package fa

import "math"

// -----------------------------------------------------------------------------
// BLAS-1 (Dongarra Linpack subset used by lbfgsb)
// -----------------------------------------------------------------------------

// daxpy: y[iy] = y[iy] + da*x[ix], constant times a vector plus a vector.
//
// Fortran: subroutine daxpy(n,da,dx,incx,dy,incy)
// Indices ix, iy advance by incx, incy starting from the first element of
// the slice (Fortran's element 1).
func daxpy(n int, da float64, dx []float64, incx int, dy []float64, incy int) {
	if n <= 0 || da == 0 {
		return
	}
	if incx == 1 && incy == 1 {
		// Unrolled by 4 to match Fortran's clean-up behavior bit-for-bit.
		m := n % 4
		for i := 0; i < m; i++ {
			dy[i] += da * dx[i]
		}
		if n < 4 {
			return
		}
		for i := m; i < n; i += 4 {
			dy[i] += da * dx[i]
			dy[i+1] += da * dx[i+1]
			dy[i+2] += da * dx[i+2]
			dy[i+3] += da * dx[i+3]
		}
		return
	}
	ix := 0
	iy := 0
	if incx < 0 {
		ix = (-n + 1) * incx
	}
	if incy < 0 {
		iy = (-n + 1) * incy
	}
	for i := 0; i < n; i++ {
		dy[iy] += da * dx[ix]
		ix += incx
		iy += incy
	}
}

// dcopy: y = x, copies a vector.
//
// Fortran: subroutine dcopy(n,dx,incx,dy,incy)
func dcopy(n int, dx []float64, incx int, dy []float64, incy int) {
	if n <= 0 {
		return
	}
	if incx == 1 && incy == 1 {
		// Unrolled by 7 to match Fortran's clean-up behavior bit-for-bit.
		m := n % 7
		for i := 0; i < m; i++ {
			dy[i] = dx[i]
		}
		if n < 7 {
			return
		}
		for i := m; i < n; i += 7 {
			dy[i] = dx[i]
			dy[i+1] = dx[i+1]
			dy[i+2] = dx[i+2]
			dy[i+3] = dx[i+3]
			dy[i+4] = dx[i+4]
			dy[i+5] = dx[i+5]
			dy[i+6] = dx[i+6]
		}
		return
	}
	ix := 0
	iy := 0
	if incx < 0 {
		ix = (-n + 1) * incx
	}
	if incy < 0 {
		iy = (-n + 1) * incy
	}
	for i := 0; i < n; i++ {
		dy[iy] = dx[ix]
		ix += incx
		iy += incy
	}
}

// ddot: returns sum_{k=1..n} dx[ix(k)]*dy[iy(k)].
//
// Fortran: double precision function ddot(n,dx,incx,dy,incy)
func ddot(n int, dx []float64, incx int, dy []float64, incy int) float64 {
	if n <= 0 {
		return 0
	}
	dtemp := 0.0
	if incx == 1 && incy == 1 {
		// Unrolled by 5 (matches Fortran's clean-up).
		m := n % 5
		for i := 0; i < m; i++ {
			dtemp += dx[i] * dy[i]
		}
		if n < 5 {
			return dtemp
		}
		for i := m; i < n; i += 5 {
			dtemp += dx[i]*dy[i] +
				dx[i+1]*dy[i+1] +
				dx[i+2]*dy[i+2] +
				dx[i+3]*dy[i+3] +
				dx[i+4]*dy[i+4]
		}
		return dtemp
	}
	ix := 0
	iy := 0
	if incx < 0 {
		ix = (-n + 1) * incx
	}
	if incy < 0 {
		iy = (-n + 1) * incy
	}
	for i := 0; i < n; i++ {
		dtemp += dx[ix] * dy[iy]
		ix += incx
		iy += incy
	}
	return dtemp
}

// dscal: x = da * x.
//
// Fortran: subroutine dscal(n,da,dx,incx)  -- returns if incx <= 0.
func dscal(n int, da float64, dx []float64, incx int) {
	if n <= 0 || incx <= 0 {
		return
	}
	if incx == 1 {
		// Unrolled by 5 (matches Fortran).
		m := n % 5
		for i := 0; i < m; i++ {
			dx[i] = da * dx[i]
		}
		if n < 5 {
			return
		}
		for i := m; i < n; i += 5 {
			dx[i] = da * dx[i]
			dx[i+1] = da * dx[i+1]
			dx[i+2] = da * dx[i+2]
			dx[i+3] = da * dx[i+3]
			dx[i+4] = da * dx[i+4]
		}
		return
	}
	nincx := n * incx
	for i := 0; i < nincx; i += incx {
		dx[i] = da * dx[i]
	}
}

// dnrm2: Euclidean norm of x[1..n] strided by incx, using Fortran's
// scale-then-square trick to avoid intermediate overflow / underflow.
//
// Fortran: double precision function dnrm2(n,x,incx)
// Note: the Netlib lbfgsb dnrm2 is the BLAS textbook version (not the
// Hammarling/dnrm2 one shipped in modern reference BLAS). The loop bound
// is `do 10 i = 1, n, incx` — matching the Fortran exactly.
func dnrm2(n int, x []float64, incx int) float64 {
	scale := 0.0
	for i := 0; i < n; i += incx {
		if a := math.Abs(x[i]); a > scale {
			scale = a
		}
	}
	if scale == 0 {
		return 0
	}
	sum := 0.0
	for i := 0; i < n; i += incx {
		v := x[i] / scale
		sum += v * v
	}
	return scale * math.Sqrt(sum)
}

// -----------------------------------------------------------------------------
// LINPACK helpers used by lbfgsb (dpofa, dtrsl).
// -----------------------------------------------------------------------------

// dpofa factors a symmetric positive definite matrix (Cholesky), in-place
// in the upper triangle. `a` is column-major, `lda` is its leading dim.
// Returns 0 on success, k (1-based) if the leading minor of order k is
// not positive definite. Mirrors LINPACK dpofa.
func dpofa(a []float64, lda, n int) int {
	for j := 1; j <= n; j++ {
		s := 0.0
		jm1 := j - 1
		if jm1 >= 1 {
			for k := 1; k <= jm1; k++ {
				// t = a(k,j) - ddot(k-1, a(1,k), 1, a(1,j), 1)
				t := a[(j-1)*lda+(k-1)] - ddot(k-1, a[(k-1)*lda:], 1, a[(j-1)*lda:], 1)
				t /= a[(k-1)*lda+(k-1)]
				a[(j-1)*lda+(k-1)] = t
				s += t * t
			}
		}
		s = a[(j-1)*lda+(j-1)] - s
		if s <= 0 {
			return j
		}
		a[(j-1)*lda+(j-1)] = math.Sqrt(s)
	}
	return 0
}

// dtrsl solves t*x = b or t'*x = b for triangular t. Operates in-place
// on b. job encoding (Fortran): 00 / 01 / 10 / 11 — see comments below.
// Returns info: 0 on success, otherwise the 1-based index of the first
// zero diagonal element of t.
func dtrsl(t []float64, ldt, n int, b []float64, job int) int {
	// check for zero diagonal
	for i := 1; i <= n; i++ {
		if t[(i-1)*ldt+(i-1)] == 0 {
			return i
		}
	}
	cs := 1
	if job%10 != 0 {
		cs = 2
	}
	if (job%100)/10 != 0 {
		cs += 2
	}
	switch cs {
	case 1: // solve t*x = b, t lower triangular
		b[0] /= t[0]
		if n >= 2 {
			for j := 2; j <= n; j++ {
				temp := -b[j-2]
				// daxpy(n-j+1, temp, t(j,j-1), 1, b(j), 1)
				daxpy(n-j+1, temp, t[(j-2)*ldt+(j-1):], 1, b[j-1:], 1)
				b[j-1] /= t[(j-1)*ldt+(j-1)]
			}
		}
	case 2: // solve t*x = b, t upper triangular
		b[n-1] /= t[(n-1)*ldt+(n-1)]
		if n >= 2 {
			for jj := 2; jj <= n; jj++ {
				j := n - jj + 1
				temp := -b[j]
				// daxpy(j, temp, t(1,j+1), 1, b(1), 1)
				daxpy(j, temp, t[j*ldt:], 1, b, 1)
				b[j-1] /= t[(j-1)*ldt+(j-1)]
			}
		}
	case 3: // solve trans(t)*x = b, t lower triangular
		b[n-1] /= t[(n-1)*ldt+(n-1)]
		if n >= 2 {
			for jj := 2; jj <= n; jj++ {
				j := n - jj + 1
				// b(j) = b(j) - ddot(jj-1, t(j+1,j), 1, b(j+1), 1)
				b[j-1] -= ddot(jj-1, t[(j-1)*ldt+j:], 1, b[j:], 1)
				b[j-1] /= t[(j-1)*ldt+(j-1)]
			}
		}
	case 4: // solve trans(t)*x = b, t upper triangular
		b[0] /= t[0]
		if n >= 2 {
			for j := 2; j <= n; j++ {
				// b(j) = b(j) - ddot(j-1, t(1,j), 1, b(1), 1)
				b[j-1] -= ddot(j-1, t[(j-1)*ldt:], 1, b, 1)
				b[j-1] /= t[(j-1)*ldt+(j-1)]
			}
		}
	}
	return 0
}
