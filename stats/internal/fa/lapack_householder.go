// fa/lapack_householder.go
//
// Householder reflector application (dlarf) and plane-rotation sequence
// application (dlasr). Used by dsytrd, dorgtr, dormtr, dsteqr in the
// dsyevr port. Faithful translations of LAPACK 3.12.1 reference Fortran.
package fa

// dlarf applies a Householder reflector H = I - tau*v*v^T to matrix C.
//
// side = 'L' : C := H * C  (m×n applied from left, v has length m)
// side = 'R' : C := C * H  (m×n applied from right, v has length n)
//
// v has stride incv. tau is the reflector scalar. work is a scratch
// buffer of length max(m, n). Mirrors LAPACK dlarf.f (with the lastv /
// lastc trailing-zero scan that newer LAPACK includes).
func dlarf(side byte, m, n int, v []float64, incv int, tau float64,
	c []float64, ldc int, work []float64,
) {
	applyLeft := side == 'L' || side == 'l'
	lastv := 0
	lastc := 0
	if tau != 0 {
		if applyLeft {
			lastv = m
		} else {
			lastv = n
		}
		var i int
		if incv > 0 {
			i = 1 + (lastv-1)*incv
		} else {
			i = 1
		}
		// Trim trailing zeros from v.
		for lastv > 0 && v[i-1] == 0 {
			lastv--
			i -= incv
		}
		if applyLeft {
			lastc = iladlc(lastv, n, c, ldc)
		} else {
			lastc = iladlr(m, lastv, c, ldc)
		}
	}
	if lastv == 0 || lastc == 0 {
		return
	}
	if applyLeft {
		// w(1:lastc) := C(1:lastv, 1:lastc)^T * v(1:lastv)
		dgemv('T', lastv, lastc, 1, c, ldc, v, incv, 0, work, 1)
		// C(1:lastv, 1:lastc) -= v * w^T  (rank-1 update)
		dger(lastv, lastc, -tau, v, incv, work, 1, c, ldc)
	} else {
		// w(1:lastc) := C(1:lastc, 1:lastv) * v(1:lastv)
		dgemv('N', lastc, lastv, 1, c, ldc, v, incv, 0, work, 1)
		dger(lastc, lastv, -tau, work, 1, v, incv, c, ldc)
	}
}

// iladlr returns the index of the last non-zero ROW of an m×n matrix.
// (Helper used by dlarf to find the trailing range; LAPACK reference
// version is iladlr.f.)
func iladlr(m, n int, a []float64, lda int) int {
	if m == 0 {
		return 0
	}
	// Quick check first/last column for structure.
	if a[(n-1)*lda+(m-1)] != 0 || a[0*lda+(m-1)] != 0 {
		return m
	}
	last := 0
	for j := 0; j < n; j++ {
		i := m - 1
		for i >= 0 && a[j*lda+i] == 0 {
			i--
		}
		if i+1 > last {
			last = i + 1
		}
	}
	return last
}

// iladlc returns the index of the last non-zero COLUMN of an m×n matrix.
// LAPACK iladlc.f.
func iladlc(m, n int, a []float64, lda int) int {
	if n == 0 {
		return 0
	}
	if a[(n-1)*lda+0] != 0 || a[(n-1)*lda+(m-1)] != 0 {
		return n
	}
	for j := n - 1; j >= 0; j-- {
		for i := 0; i < m; i++ {
			if a[j*lda+i] != 0 {
				return j + 1
			}
		}
	}
	return 0
}
