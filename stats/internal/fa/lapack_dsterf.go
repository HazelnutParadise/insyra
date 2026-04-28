// fa/lapack_dsterf.go
//
// dsterf — eigenvalues-only QL/QR with implicit shift on a symmetric
// tridiagonal matrix. Delegates to gonum's bit-equivalent Dsterf.
package fa

// dsterf computes all eigenvalues of a symmetric tridiagonal matrix
// (diagonal d, off-diagonal e) using gonum's faithful Pal-Walker-Kahan
// QL/QR. d is overwritten with eigenvalues ascending; e is destroyed.
// Returns 0 on convergence, 1 if not all eigenvalues converged.
func dsterf(n int, d, e []float64) (info int) {
	return gonumDsterf(n, d, e)
}
