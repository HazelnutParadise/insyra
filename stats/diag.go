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

// Diag creates a diagonal matrix or extracts the diagonal of a matrix, the
// way R's diag() does.
//
// Deprecated: Diag takes and returns any. Use the typed function for each
// use; Diag is removed in the next release.
//
//	Diag(m)                 DiagOf(m)
//	Diag(v)                 DiagMatrix(v)
//	Diag(v, n)              DiagMatrixSize(v, n, n)
//	Diag(v, nrow, ncol)     DiagMatrixSize(v, nrow, ncol)
//	Diag(n)                 IdentityMatrix(n)
//	Diag(nil)               IdentityMatrix(1)
//	Diag(nil, nrow, ncol)   DiagMatrixSize(ones, nrow, ncol), ones of length min(nrow, ncol)
//
// DiagMatrixSize refuses a v longer than the diagonal, which Diag cuts.
func Diag(x any, dims ...int) (any, error) {
	var nrow, ncol int
	switch len(dims) {
	case 0:
	case 1:
		nrow = dims[0]
		ncol = dims[0]
	case 2:
		nrow = dims[0]
		ncol = dims[1]
	default:
		return nil, errors.New("too many dimensions specified")
	}

	switch v := x.(type) {
	case *mat.Dense:
		r, c := v.Dims()
		size := min(r, c)
		diag := make([]float64, size)
		for i := range size {
			diag[i] = v.At(i, i)
		}
		return diag, nil
	case []float64:
		n := len(v)
		if nrow > 0 {
			n = nrow
		}
		if ncol <= 0 {
			ncol = n
		}
		if nrow <= 0 {
			nrow = n
		}
		if nrow < 1 || ncol < 1 {
			return nil, fmt.Errorf("Diag: size %dx%d is below 1x1", nrow, ncol)
		}
		matrix := mat.NewDense(nrow, ncol, nil)
		for i := 0; i < min(len(v), min(nrow, ncol)); i++ {
			matrix.Set(i, i, v[i])
		}
		return matrix, nil
	case int:
		n := v
		if nrow > 0 {
			n = nrow
		}
		if ncol <= 0 {
			ncol = n
		}
		if nrow <= 0 {
			nrow = n
		}
		if nrow < 1 || ncol < 1 {
			return nil, fmt.Errorf("Diag: size %dx%d is below 1x1", nrow, ncol)
		}
		matrix := mat.NewDense(nrow, ncol, nil)
		for i := 0; i < min(n, min(nrow, ncol)); i++ {
			matrix.Set(i, i, 1.0)
		}
		return matrix, nil
	case float64:
		n := int(v)
		if nrow > 0 {
			n = nrow
		}
		if ncol <= 0 {
			ncol = n
		}
		if nrow <= 0 {
			nrow = n
		}
		if nrow < 1 || ncol < 1 {
			return nil, fmt.Errorf("Diag: size %dx%d is below 1x1", nrow, ncol)
		}
		matrix := mat.NewDense(nrow, ncol, nil)
		for i := 0; i < min(n, min(nrow, ncol)); i++ {
			matrix.Set(i, i, 1.0)
		}
		return matrix, nil
	case nil:
		n := 1
		if nrow > 0 {
			n = nrow
		}
		if ncol <= 0 {
			ncol = n
		}
		if nrow <= 0 {
			nrow = n
		}
		if nrow < 1 || ncol < 1 {
			return nil, fmt.Errorf("Diag: size %dx%d is below 1x1", nrow, ncol)
		}
		matrix := mat.NewDense(nrow, ncol, nil)
		for i := 0; i < min(n, min(nrow, ncol)); i++ {
			matrix.Set(i, i, 1.0)
		}
		return matrix, nil
	default:
		return nil, fmt.Errorf("unsupported type for Diag: %T", x)
	}
}
