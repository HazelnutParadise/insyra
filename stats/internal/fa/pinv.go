// fa/pinv.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Pinv computes the Moore–Penrose pseudoinverse of X using SVD.
// It mirrors psych::Pinv(X, tol = sqrt(.Machine$double.eps)) behavior:
// - threshold = max(tol * d[0], 0)
// - keep singular values strictly greater than threshold
//
// R reference:
//
//	svdX <- svd(X)
//	p <- svdX$d > max(tol * svdX$d[1], 0)
//	if (all(p)) V %*% (1/d * t(U)) else V[,p] %*% (1/d[p] * t(U[,p]))
func Pinv(X mat.Matrix, tol float64) *mat.Dense {
	var svd mat.SVD
	// Full SVD to get U, V and singular values.
	ok := svd.Factorize(X, mat.SVDThin)
	if !ok {
		// Fallback to full if thin fails for some reason.
		_ = svd.Factorize(X, mat.SVDNone)
	}

	// Extract U, V and singular values.
	var U, V mat.Dense
	svd.UTo(&U)
	svd.VTo(&V)
	d := svd.Values(nil)
	if len(d) == 0 {
		return mat.NewDense(0, 0, nil)
	}

	if tol <= 0 {
		tol = math.Sqrt(math.SmallestNonzeroFloat64) // ~ sqrt(.Machine$double.eps)
	}
	threshold := math.Max(tol*d[0], 0)

	// Build diag(1/d[p]) with mask p = d_i > threshold
	r := 0
	for _, s := range d {
		if s > threshold {
			r++
		}
	}
	if r == 0 {
		// All singular values are ~0: return zero matrix with flipped shape.
		rX, cX := X.Dims()
		return mat.NewDense(cX, rX, nil)
	}

	// Slice U and V by kept columns
	var Ukeep, Vkeep mat.Dense
	Ukeep.CloneFrom(&U)
	Vkeep.CloneFrom(&V)

	// Build D^+ as diagonal with 1/d for kept entries.
	Dp := mat.NewDense(r, r, nil)
	k := 0
	for i := range d {
		if d[i] > threshold {
			Dp.Set(k, k, 1.0/d[i])
			k++
			if k == r {
				break
			}
		}
	}

	// We need V[,p] * Dp * U[,p]^T
	// To avoid copying columns, form selection by constructing
	// column index lists.
	idx := make([]int, 0, r)
	for i := range d {
		if d[i] > threshold {
			idx = append(idx, i)
		}
	}

	Vsub := pickColumns(&Vkeep, idx)
	Usub := pickColumns(&Ukeep, idx)

	// temp = Vsub * Dp
	var temp mat.Dense
	temp.Mul(Vsub, Dp)
	// Pinv = temp * Usub^T
	var pinv mat.Dense
	pinv.Mul(&temp, Usub.T())
	return &pinv
}

// pickColumns returns a view matrix with the selected columns.
func pickColumns(m *mat.Dense, cols []int) mat.Matrix {
	r, _ := m.Dims()
	out := mat.NewDense(r, len(cols), nil)
	for j, cj := range cols {
		out.ColView(j).(*mat.VecDense).CopyVec(m.ColView(cj))
	}
	return out
}
