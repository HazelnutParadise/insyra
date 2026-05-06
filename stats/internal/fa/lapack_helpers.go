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

// lapackSafmin = DLAMCH('S'); used by various MRRR routines.
const lapackSafmin = 2.2250738585072014e-308


// dlassq delegates to gonum's bit-equivalent implementation.
func dlassq(n int, x []float64, incx int, scale, sumsq float64) (float64, float64) {
	return gonumImpl().Dlassq(n, x, incx, scale, sumsq)
}


// dlanst delegates to gonum (translates byte → lapack.MatrixNorm enum).
func dlanst(norm byte, n int, d, e []float64) float64 {
	return gonumDlanst(norm, n, d, e)
}

// dlaev2 delegates to gonum.
func dlaev2(a, b, c float64) (rt1, rt2, cs1, sn1 float64) {
	return gonumImpl().Dlaev2(a, b, c)
}

// dlasrt delegates to gonum (translates byte → lapack.Sort enum).
func dlasrt(id byte, n int, d []float64) (info int) {
	return gonumDlasrt(id, n, d)
}

// dlarfg delegates to gonum (adapts pointer-mutation convention).
func dlarfg(n int, alpha *float64, x []float64, incx int) (tau float64) {
	return gonumDlarfg(n, alpha, x, incx)
}
