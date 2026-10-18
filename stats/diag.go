package stats

import (
	"errors"
	"fmt"
	"reflect"

	"gonum.org/v1/gonum/mat"
)

// DiagOf returns the main diagonal of m, m[i][i] for i up to the smaller of
// its row and column counts. It returns an error for a nil m.
func DiagOf(m mat.Matrix) ([]float64, error) {
	if m == nil || (reflect.ValueOf(m).Kind() == reflect.Pointer && reflect.ValueOf(m).IsNil()) {
		return nil, errors.New("DiagOf: matrix is nil")
	}
	r, c := m.Dims()
	size := min(r, c)
	diag := make([]float64, size)
	for i := range size {
		diag[i] = m.At(i, i)
	}
	return diag, nil
}

// DiagMatrix returns the len(v) x len(v) matrix with v on its diagonal and
// zeros elsewhere. It returns an error for an empty v.
func DiagMatrix(v []float64) (*mat.Dense, error) {
	if len(v) == 0 {
		return nil, errors.New("DiagMatrix: v is empty")
	}
	return DiagMatrixSize(v, len(v), len(v))
}

// DiagMatrixSize returns the nrow x ncol matrix whose diagonal starts with
// v and is zero past the end of v; every other entry is zero. DiagMatrixSize
// with a v of ones gives a rectangular identity. It returns an error for a
// size below 1 and for a v longer than the diagonal, instead of cutting v.
func DiagMatrixSize(v []float64, nrow, ncol int) (*mat.Dense, error) {
	if nrow < 1 || ncol < 1 {
		return nil, fmt.Errorf("DiagMatrixSize: size %dx%d is below 1x1", nrow, ncol)
	}
	if len(v) > min(nrow, ncol) {
		return nil, fmt.Errorf("DiagMatrixSize: %d values do not fit a %dx%d diagonal", len(v), nrow, ncol)
	}
	m := mat.NewDense(nrow, ncol, nil)
	for i := 0; i < len(v); i++ {
		m.Set(i, i, v[i])
	}
	return m, nil
}

// IdentityMatrix returns the n x n identity matrix. It returns an error for
// an n below 1.
func IdentityMatrix(n int) (*mat.Dense, error) {
	if n < 1 {
		return nil, fmt.Errorf("IdentityMatrix: n is %d, below 1", n)
	}
	ones := make([]float64, n)
	for i := range ones {
		ones[i] = 1.0
	}
	return DiagMatrixSize(ones, n, n)
}
