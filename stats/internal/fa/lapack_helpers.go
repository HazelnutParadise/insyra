// fa/lapack_helpers.go
//
// Foundational LAPACK leaf routines used by the dsyevr port:
//
//   dlartg  — generate a plane rotation (Givens) — dlartg.f90
//   dlanst  — norm of a real symmetric tridiagonal — dlanst.f
//   dlaev2  — eigenvalues / eigenvectors of a 2x2 symmetric matrix — dlaev2.f
//   dlasrt  — quicksort sort doubles ascending or descending — dlasrt.f
//   dlarfg  — generate an elementary Householder reflector — dlarfg.f
//   dlapy2  — sqrt(x^2 + y^2) avoiding overflow — dlapy2.f
//   dlassq  — sum-of-squares with overflow scaling — dlassq.f
//
// These are pure-Go faithful translations of the Fortran reference for
// LAPACK 3.12.1 (the version that ships with R 4.5.x). Conventions:
//
//   * Slice arguments use 1-based indexing semantics shifted to 0-based
//     by `idx-1` adjustments at access sites; the rest of the code reads
//     identically to the Fortran reference for line-by-line review.
//   * Matrix arguments are column-major flat []float64 with explicit
//     leading dimension `lda` so BLAS-style strides translate verbatim.
//   * Stride / `incx` is honored exactly matching reference BLAS.
package fa

import "math"

// safmin / safmax used by dlartg's safeguarded path.
const (
	lapackSafmin   = 2.2250738585072014e-308 // DLAMCH('S')
	lapackSafmax   = 1.7976931348623158e+308 / 2
	lapackSafminEx = lapackSafmin / 2.220446049250313e-16 // for dlarfg
)

// dlapy2 returns sqrt(x^2 + y^2) computed without overflow / underflow.
// Mirrors LAPACK dlapy2.f.
func dlapy2(x, y float64) float64 {
	xa := math.Abs(x)
	ya := math.Abs(y)
	w := math.Max(xa, ya)
	z := math.Min(xa, ya)
	if z == 0 {
		return w
	}
	r := z / w
	return w * math.Sqrt(1+r*r)
}

// dlassq updates sum-of-squares: returns (newScale, newSumsq) such that
// (newScale^2 * newSumsq) == (scale^2 * sumsq) + sum(x[i*incx]^2 for i in
// [0, n)). Mirrors LAPACK dlassq.f (simplified for the cases we need).
func dlassq(n int, x []float64, incx int, scale, sumsq float64) (float64, float64) {
	if n == 0 {
		return scale, sumsq
	}
	for ix := 0; ix < n; ix++ {
		v := x[ix*incx]
		if v == 0 {
			continue
		}
		ax := math.Abs(v)
		if scale < ax {
			ratio := scale / ax
			sumsq = 1 + sumsq*ratio*ratio
			scale = ax
		} else {
			ratio := ax / scale
			sumsq += ratio * ratio
		}
	}
	return scale, sumsq
}

// dlartg generates a Givens plane rotation [[c, s], [-s, c]] · [f; g] = [r; 0].
// Mirrors LAPACK dlartg.f90 (LAPACK 3.7+ accurate version).
func dlartg(f, g float64) (c, s, r float64) {
	const zero = 0.0
	const one = 1.0
	rtmin := math.Sqrt(lapackSafmin)
	rtmax := math.Sqrt(lapackSafmax / 2)

	f1 := math.Abs(f)
	g1 := math.Abs(g)
	switch {
	case g == zero:
		c = one
		s = zero
		r = f
	case f == zero:
		c = zero
		s = math.Copysign(one, g)
		r = g1
	case f1 > rtmin && f1 < rtmax && g1 > rtmin && g1 < rtmax:
		d := math.Sqrt(f*f + g*g)
		c = f1 / d
		r = math.Copysign(d, f)
		s = g / r
	default:
		u := math.Min(lapackSafmax, math.Max(lapackSafmin, math.Max(f1, g1)))
		fs := f / u
		gs := g / u
		d := math.Sqrt(fs*fs + gs*gs)
		c = math.Abs(fs) / d
		r = math.Copysign(d, f)
		s = gs / r
		r = r * u
	}
	return
}

// dlanst returns the value of the requested norm of a real symmetric
// tridiagonal matrix A with diagonal d (length n) and off-diagonal e
// (length n-1). norm in {'M','1','I','F','E'}. Mirrors LAPACK dlanst.f.
func dlanst(norm byte, n int, d, e []float64) float64 {
	if n <= 0 {
		return 0
	}
	switch norm {
	case 'M', 'm':
		anorm := math.Abs(d[n-1])
		for i := 0; i < n-1; i++ {
			if v := math.Abs(d[i]); v > anorm || math.IsNaN(v) {
				anorm = v
			}
			if v := math.Abs(e[i]); v > anorm || math.IsNaN(v) {
				anorm = v
			}
		}
		return anorm
	case 'O', 'o', '1', 'I', 'i':
		// 1-norm = inf-norm for symmetric matrix; column j sum is
		// |d[j]| + |e[j-1]| + |e[j]| (clamped at boundaries).
		if n == 1 {
			return math.Abs(d[0])
		}
		anorm := math.Abs(d[0]) + math.Abs(e[0])
		if v := math.Abs(e[n-2]) + math.Abs(d[n-1]); v > anorm || math.IsNaN(v) {
			anorm = v
		}
		for i := 1; i < n-1; i++ {
			v := math.Abs(d[i]) + math.Abs(e[i]) + math.Abs(e[i-1])
			if v > anorm || math.IsNaN(v) {
				anorm = v
			}
		}
		return anorm
	case 'F', 'f', 'E', 'e':
		scale := 0.0
		sumsq := 1.0
		if n > 1 {
			scale, sumsq = dlassq(n-1, e, 1, scale, sumsq)
			sumsq = 2 * sumsq
		}
		scale, sumsq = dlassq(n, d, 1, scale, sumsq)
		return scale * math.Sqrt(sumsq)
	}
	return 0
}

// dlaev2 computes the eigenvalues (rt1 >= rt2 in absolute value) and
// the eigenvector of the 2x2 symmetric matrix [[a, b], [b, c]]:
//
//	[ cs1  sn1 ] [a b ] [ cs1 -sn1 ] = [ rt1   0  ]
//	[-sn1  cs1 ] [b c ] [ sn1  cs1 ]   [  0   rt2 ]
//
// Mirrors LAPACK dlaev2.f.
func dlaev2(a, b, c float64) (rt1, rt2, cs1, sn1 float64) {
	sm := a + c
	df := a - c
	adf := math.Abs(df)
	tb := b + b
	ab := math.Abs(tb)

	var acmx, acmn float64
	if math.Abs(a) > math.Abs(c) {
		acmx, acmn = a, c
	} else {
		acmx, acmn = c, a
	}

	var rt float64
	switch {
	case adf > ab:
		rt = adf * math.Sqrt(1+(ab/adf)*(ab/adf))
	case adf < ab:
		rt = ab * math.Sqrt(1+(adf/ab)*(adf/ab))
	default:
		rt = ab * math.Sqrt(2)
	}

	var sgn1 int
	switch {
	case sm < 0:
		rt1 = 0.5 * (sm - rt)
		sgn1 = -1
		rt2 = (acmx/rt1)*acmn - (b/rt1)*b
	case sm > 0:
		rt1 = 0.5 * (sm + rt)
		sgn1 = 1
		rt2 = (acmx/rt1)*acmn - (b/rt1)*b
	default:
		rt1 = 0.5 * rt
		rt2 = -0.5 * rt
		sgn1 = 1
	}

	var cs float64
	var sgn2 int
	if df >= 0 {
		cs = df + rt
		sgn2 = 1
	} else {
		cs = df - rt
		sgn2 = -1
	}
	acs := math.Abs(cs)
	if acs > ab {
		ct := -tb / cs
		sn1 = 1 / math.Sqrt(1+ct*ct)
		cs1 = ct * sn1
	} else if ab == 0 {
		cs1 = 1
		sn1 = 0
	} else {
		tn := -cs / tb
		cs1 = 1 / math.Sqrt(1+tn*tn)
		sn1 = tn * cs1
	}
	if sgn1 == sgn2 {
		tn := cs1
		cs1 = -sn1
		sn1 = tn
	}
	return
}

// dlasrt sorts d[0:n] in increasing ('I') or decreasing ('D') order.
// Mirrors LAPACK dlasrt.f using insertion-sort + quicksort hybrid
// (insertion on segments of length <= 20).
func dlasrt(id byte, n int, d []float64) (info int) {
	dir := -1
	switch id {
	case 'D', 'd':
		dir = 0 // decreasing
	case 'I', 'i':
		dir = 1 // increasing
	}
	if dir == -1 {
		return -1
	}
	if n < 0 {
		return -2
	}
	if n <= 1 {
		return 0
	}

	const selectN = 20
	type segment struct{ lo, hi int }
	stack := make([]segment, 0, 64)
	stack = append(stack, segment{0, n - 1})

	for len(stack) > 0 {
		seg := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		start, endd := seg.lo, seg.hi
		if endd-start <= selectN && endd-start > 0 {
			// Insertion sort.
			if dir == 0 {
				for i := start + 1; i <= endd; i++ {
					for j := i; j > start; j-- {
						if d[j] > d[j-1] {
							d[j], d[j-1] = d[j-1], d[j]
						} else {
							break
						}
					}
				}
			} else {
				for i := start + 1; i <= endd; i++ {
					for j := i; j > start; j-- {
						if d[j] < d[j-1] {
							d[j], d[j-1] = d[j-1], d[j]
						} else {
							break
						}
					}
				}
			}
		} else if endd-start > selectN {
			// Median-of-three pivot.
			d1 := d[start]
			d2 := d[endd]
			i := (start + endd) / 2
			d3 := d[i]
			var dmnmx float64
			if d1 < d2 {
				if d3 < d1 {
					dmnmx = d1
				} else if d3 < d2 {
					dmnmx = d3
				} else {
					dmnmx = d2
				}
			} else {
				if d3 < d2 {
					dmnmx = d2
				} else if d3 < d1 {
					dmnmx = d3
				} else {
					dmnmx = d1
				}
			}
			if dir == 0 {
				// Decreasing: partition with elements > pivot to left.
				i = start - 1
				j := endd + 1
				for {
					for {
						j--
						if d[j] >= dmnmx {
							break
						}
					}
					for {
						i++
						if d[i] <= dmnmx {
							break
						}
					}
					if i < j {
						d[i], d[j] = d[j], d[i]
					} else {
						break
					}
				}
				if j-start > endd-j-1 {
					stack = append(stack, segment{start, j})
					stack = append(stack, segment{j + 1, endd})
				} else {
					stack = append(stack, segment{j + 1, endd})
					stack = append(stack, segment{start, j})
				}
			} else {
				// Increasing: partition with elements < pivot to left.
				i = start - 1
				j := endd + 1
				for {
					for {
						j--
						if d[j] <= dmnmx {
							break
						}
					}
					for {
						i++
						if d[i] >= dmnmx {
							break
						}
					}
					if i < j {
						d[i], d[j] = d[j], d[i]
					} else {
						break
					}
				}
				if j-start > endd-j-1 {
					stack = append(stack, segment{start, j})
					stack = append(stack, segment{j + 1, endd})
				} else {
					stack = append(stack, segment{j + 1, endd})
					stack = append(stack, segment{start, j})
				}
			}
		}
	}
	return 0
}

// dlarfg generates an elementary Householder reflector H of order n with
//
//	H * (alpha, x)' = (beta, 0)'
//
// where H = I - tau * v * v', v = (1, x_scaled). Mirrors LAPACK dlarfg.f.
//
// On entry: alpha is the leading scalar; x is the rest of the vector
// (length n-1) with stride incx.
// On exit: alpha is overwritten with beta; x is overwritten with the
// non-trivial part of v (the leading 1 is implicit). tau is the scalar.
func dlarfg(n int, alpha *float64, x []float64, incx int) (tau float64) {
	if n <= 1 {
		return 0
	}
	xnorm := dnrm2(n-1, x, incx)
	if xnorm == 0 {
		return 0
	}
	beta := -math.Copysign(dlapy2(*alpha, xnorm), *alpha)
	knt := 0
	if math.Abs(beta) < lapackSafminEx {
		rsafmn := 1 / lapackSafminEx
		for math.Abs(beta) < lapackSafminEx && knt < 20 {
			knt++
			dscal(n-1, rsafmn, x, incx)
			beta *= rsafmn
			*alpha *= rsafmn
		}
		xnorm = dnrm2(n-1, x, incx)
		beta = -math.Copysign(dlapy2(*alpha, xnorm), *alpha)
	}
	tau = (beta - *alpha) / beta
	dscal(n-1, 1/(*alpha-beta), x, incx)
	for j := 0; j < knt; j++ {
		beta *= lapackSafminEx
	}
	*alpha = beta
	return
}
