package stats

import (
	"errors"
	"fmt"

	"gonum.org/v1/gonum/mat"
)

// Diag creates a diagonal matrix or extracts the diagonal of a matrix.
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
		matrix := mat.NewDense(nrow, ncol, nil)
		for i := 0; i < min(n, min(nrow, ncol)); i++ {
			matrix.Set(i, i, 1.0)
		}
		return matrix, nil
	default:
		return nil, fmt.Errorf("unsupported type for Diag: %T", x)
	}
}
