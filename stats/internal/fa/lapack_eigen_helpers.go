// fa/lapack_eigen_helpers.go
//
// Shared LAPACK helper routines used by the dsyevr / dstemr eigenvalue
// path. Currently only dlae2 (2×2 symmetric eigenvalues, delegates to
// gonum) is needed by live code.
//
// This file previously held dsteqr (QL/QR tridiagonal eigensolver),
// dsyev (top-level QL/QR driver), dlaset (matrix initialization),
// dlasclG (general matrix scaling). All removed once the MRRR dsyevr
// path proved sufficient for R-parity. R itself uses dsyevr.
package fa

// dlae2 delegates to gonum's bit-equivalent Dlae2 (2×2 symmetric
// eigenvalues, used by dlarrf and friends in the MRRR pipeline).
func dlae2(a, b, c float64) (rt1, rt2 float64) {
	return gonumImpl().Dlae2(a, b, c)
}
