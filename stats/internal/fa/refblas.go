// fa/refblas.go
//
// Pure-Go ports of Netlib reference BLAS routines used by the factor
// analysis pipeline (oblimin / GPA rotation, scoring, KMO). Their purpose
// is to match Fortran/R BLAS's floating-point accumulation order exactly
// so cross-language R parity tests stay bit-near-identical even after
// thousands of iterations.
//
// Gonum's optimized blocked dgemm is faster but uses different
// accumulation order, which manifests as 1-3 ULP per element after one
// dgemm call. After the ~1000 oblimin iterations or the dense covariance
// solves in regression scoring, these tiny rounding differences
// accumulate to ~1e-5 final-output divergence.
//
// Translation source: BLAS reference (Netlib), dgemm.f / dgemv.f /
// dsymv.f / dsyr2.f / dsyrk.f. We restrict to TRANSA/TRANSB = 'N' on
// dgemm and call sites pre-transpose where needed; this keeps the loop
// structure identical to BLAS reference's most common code path.
//
// All matrices are dense; matrix arguments may be `mat.Matrix` (we read
// via At/Dims) and outputs are `*mat.Dense`. We do NOT touch internal
// gonum storage; this means we pay an extra Set per write, but for the
// problem sizes here (n,p < 50) it's negligible and the order match is
// what we care about.
package fa

import "gonum.org/v1/gonum/mat"

// refDgemmNN computes C := alpha*A*B + beta*C using the Fortran reference
// dgemm's column-major J-L-I loop order (no transposes). Caller must
// ensure dimensions are compatible: A is m×k, B is k×n, C is m×n.
func refDgemmNN(alpha float64, a, b mat.Matrix, beta float64, c *mat.Dense) {
	m, k := a.Dims()
	kb, n := b.Dims()
	if k != kb {
		panic("refDgemmNN: inner dimension mismatch")
	}
	rc, cc := c.Dims()
	if rc != m || cc != n {
		panic("refDgemmNN: output dimension mismatch")
	}
	for j := 0; j < n; j++ {
		switch beta {
		case 0:
			for i := 0; i < m; i++ {
				c.Set(i, j, 0)
			}
		case 1:
			// no-op
		default:
			for i := 0; i < m; i++ {
				c.Set(i, j, beta*c.At(i, j))
			}
		}
		for l := 0; l < k; l++ {
			temp := alpha * b.At(l, j)
			if temp == 0 {
				continue
			}
			for i := 0; i < m; i++ {
				c.Set(i, j, c.At(i, j)+temp*a.At(i, l))
			}
		}
	}
}

// refMul is a drop-in replacement for c.Mul(a, b) using refDgemmNN.
func refMul(c *mat.Dense, a, b mat.Matrix) {
	refDgemmNN(1, a, b, 0, c)
}

// refMulNew allocates a fresh *mat.Dense and computes A*B into it,
// returning the result. Convenience wrapper for `var c mat.Dense; c.Mul(a, b)`
// pattern.
func refMulNew(a, b mat.Matrix) *mat.Dense {
	m, _ := a.Dims()
	_, n := b.Dims()
	c := mat.NewDense(m, n, nil)
	refDgemmNN(1, a, b, 0, c)
	return c
}

// refDgemvN computes y := alpha*A*x + beta*y using the Fortran reference
// dgemv's column-major J outer / I inner loop (matches BLAS dgemv with
// TRANS='N' and incx=incy=1). A is m×n, x has length n, y has length m.
func refDgemvN(alpha float64, a mat.Matrix, x []float64, beta float64, y []float64) {
	m, n := a.Dims()
	if len(x) != n || len(y) != m {
		panic("refDgemvN: dimension mismatch")
	}
	switch beta {
	case 0:
		for i := 0; i < m; i++ {
			y[i] = 0
		}
	case 1:
		// no-op
	default:
		for i := 0; i < m; i++ {
			y[i] = beta * y[i]
		}
	}
	for j := 0; j < n; j++ {
		temp := alpha * x[j]
		if temp == 0 {
			continue
		}
		for i := 0; i < m; i++ {
			y[i] += temp * a.At(i, j)
		}
	}
}

// refDsymvU computes y := alpha*A*x + beta*y where A is symmetric and
// only its UPPER triangle is referenced. Mirrors BLAS dsymv UPLO='U'.
func refDsymvU(alpha float64, a mat.Matrix, x []float64, beta float64, y []float64) {
	n, nc := a.Dims()
	if n != nc || len(x) != n || len(y) != n {
		panic("refDsymvU: dimension mismatch")
	}
	switch beta {
	case 0:
		for i := 0; i < n; i++ {
			y[i] = 0
		}
	case 1:
	default:
		for i := 0; i < n; i++ {
			y[i] = beta * y[i]
		}
	}
	// reference order: J outer; for I=0..J-1 use a(i,j); for diag and below use a(j,j) and below via symmetry
	for j := 0; j < n; j++ {
		temp1 := alpha * x[j]
		temp2 := 0.0
		for i := 0; i < j; i++ {
			y[i] += temp1 * a.At(i, j)
			temp2 += a.At(i, j) * x[i]
		}
		y[j] += temp1*a.At(j, j) + alpha*temp2
	}
}
