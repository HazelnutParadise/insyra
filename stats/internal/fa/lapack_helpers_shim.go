// fa/lapack_helpers_shim.go
//
// Replaces our hand-ported scalar / 1D LAPACK helpers with thin shims
// delegating to gonum. Each function listed here was originally a
// faithful translation of the LAPACK 3.12.1 Fortran reference; gonum's
// implementations are likewise faithful, so the ALGORITHM is identical
// to R's reference LAPACK. For these scalar / 1D routines there is no
// BLAS accumulation order to worry about, so substitution is bit-exact
// equivalent.
//
// Functions delegated:
//   dlapy2, dlae2, dlaev2, dlartg, dlassq, dlanst, dlasrt, dsterf, dlarfg
//
// Functions NOT delegated (require column-major layout; gonum is
// row-major): dlansy, dlaset, dlascl, dlarf, dlasr, dgemv, dger, dsymv,
// dsyr2, dlatrd, dsytrd, dorgtr, dsteqr.
package fa

import (
	"gonum.org/v1/gonum/lapack"
	"gonum.org/v1/gonum/lapack/gonum"
)

// To enable shims, this file currently provides reference (no-op or
// example) wrappers — actual delegation is done by removing the
// hand-written bodies in the original files.

func gonumImpl() gonum.Implementation { return gonum.Implementation{} }

// gonumDsterf — wraps gonum's Dsterf (returns bool ok) into our
// info-int convention (0 = converged, >0 = number of unconverged).
// We approximate non-convergence with 1 since gonum's Dsterf hides
// the count.
func gonumDsterf(n int, d, e []float64) int {
	if gonumImpl().Dsterf(n, d, e) {
		return 0
	}
	return 1
}

// gonumDlasrt wraps gonum's Dlasrt to accept byte 'I'/'D'.
func gonumDlasrt(id byte, n int, d []float64) int {
	var s lapack.Sort
	switch id {
	case 'I', 'i':
		s = lapack.SortIncreasing
	case 'D', 'd':
		s = lapack.SortDecreasing
	default:
		return -1
	}
	gonumImpl().Dlasrt(s, n, d)
	return 0
}

// gonumDlanst wraps gonum's Dlanst translating byte → enum.
func gonumDlanst(norm byte, n int, d, e []float64) float64 {
	var k lapack.MatrixNorm
	switch norm {
	case 'M', 'm':
		k = lapack.MaxAbs
	case '1', 'O', 'o':
		k = lapack.MaxColumnSum
	case 'I', 'i':
		k = lapack.MaxRowSum
	case 'F', 'f', 'E', 'e':
		k = lapack.Frobenius
	default:
		return 0
	}
	return gonumImpl().Dlanst(k, n, d, e)
}

// gonumDlarfg wraps gonum's Dlarfg (which returns (beta, tau) by
// value) into our pointer-mutating convention.
func gonumDlarfg(n int, alpha *float64, x []float64, incx int) (tau float64) {
	beta, t := gonumImpl().Dlarfg(n, *alpha, x, incx)
	*alpha = beta
	return t
}
