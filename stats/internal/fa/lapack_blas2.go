// fa/lapack_blas2.go
//
// BLAS-2 routines used by the dsyevr port: dgemv, dger, dswap. These
// supplement the BLAS-1 set in lbfgsb_blas.go (daxpy/dcopy/ddot/dnrm2/
// dscal). Strict Fortran-reference accumulation order: J outer / I inner.
//
// All matrices are column-major flat []float64 with explicit `lda`.
package fa

// dgemv computes y := alpha*op(A)*x + beta*y where op(A)=A if trans=='N',
// or A^T if trans=='T'. Reference BLAS dgemv.
//
// A is m×n with leading dim lda. For trans='N': x has length n, y length m.
// For trans='T': x has length m, y length n. Strides incx, incy honored.
func dgemv(trans byte, m, n int, alpha float64, a []float64, lda int,
	x []float64, incx int, beta float64, y []float64, incy int,
) {
	if m == 0 || n == 0 || (alpha == 0 && beta == 1) {
		return
	}
	var lenx, leny int
	switch trans {
	case 'N', 'n':
		lenx, leny = n, m
	case 'T', 't', 'C', 'c':
		lenx, leny = m, n
	default:
		return
	}
	kx := 0
	if incx < 0 {
		kx = -(lenx - 1) * incx
	}
	ky := 0
	if incy < 0 {
		ky = -(leny - 1) * incy
	}
	// y := beta * y
	if beta != 1 {
		if incy == 1 {
			if beta == 0 {
				for i := 0; i < leny; i++ {
					y[i] = 0
				}
			} else {
				for i := 0; i < leny; i++ {
					y[i] *= beta
				}
			}
		} else {
			iy := ky
			for i := 0; i < leny; i++ {
				if beta == 0 {
					y[iy] = 0
				} else {
					y[iy] *= beta
				}
				iy += incy
			}
		}
	}
	if alpha == 0 {
		return
	}
	if trans == 'N' || trans == 'n' {
		jx := kx
		if incy == 1 {
			for j := 0; j < n; j++ {
				temp := alpha * x[jx]
				if temp != 0 {
					for i := 0; i < m; i++ {
						y[i] += temp * a[j*lda+i]
					}
				}
				jx += incx
			}
		} else {
			for j := 0; j < n; j++ {
				temp := alpha * x[jx]
				if temp != 0 {
					iy := ky
					for i := 0; i < m; i++ {
						y[iy] += temp * a[j*lda+i]
						iy += incy
					}
				}
				jx += incx
			}
		}
	} else {
		jy := ky
		if incx == 1 {
			for j := 0; j < n; j++ {
				temp := 0.0
				for i := 0; i < m; i++ {
					temp += a[j*lda+i] * x[i]
				}
				y[jy] += alpha * temp
				jy += incy
			}
		} else {
			for j := 0; j < n; j++ {
				temp := 0.0
				ix := kx
				for i := 0; i < m; i++ {
					temp += a[j*lda+i] * x[ix]
					ix += incx
				}
				y[jy] += alpha * temp
				jy += incy
			}
		}
	}
}

// dger computes A := alpha*x*y^T + A. Reference BLAS dger.
// A is m×n column-major with leading dim lda. x length m, y length n,
// strides incx, incy honored.
func dger(m, n int, alpha float64, x []float64, incx int,
	y []float64, incy int, a []float64, lda int,
) {
	if m == 0 || n == 0 || alpha == 0 {
		return
	}
	jy := 0
	if incy < 0 {
		jy = -(n - 1) * incy
	}
	if incx == 1 {
		for j := 0; j < n; j++ {
			if y[jy] != 0 {
				temp := alpha * y[jy]
				for i := 0; i < m; i++ {
					a[j*lda+i] += x[i] * temp
				}
			}
			jy += incy
		}
	} else {
		kx := 0
		if incx < 0 {
			kx = -(m - 1) * incx
		}
		for j := 0; j < n; j++ {
			if y[jy] != 0 {
				temp := alpha * y[jy]
				ix := kx
				for i := 0; i < m; i++ {
					a[j*lda+i] += x[ix] * temp
					ix += incx
				}
			}
			jy += incy
		}
	}
}

// dswap swaps two vectors. Reference BLAS dswap.
func dswap(n int, x []float64, incx int, y []float64, incy int) {
	if n <= 0 {
		return
	}
	if incx == 1 && incy == 1 {
		for i := 0; i < n; i++ {
			x[i], y[i] = y[i], x[i]
		}
		return
	}
	ix := 0
	iy := 0
	if incx < 0 {
		ix = -(n - 1) * incx
	}
	if incy < 0 {
		iy = -(n - 1) * incy
	}
	for i := 0; i < n; i++ {
		x[ix], y[iy] = y[iy], x[ix]
		ix += incx
		iy += incy
	}
}
