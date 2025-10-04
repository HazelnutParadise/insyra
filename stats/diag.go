package stats

import (
	"gonum.org/v1/gonum/mat"
)

// Diag creates a diagonal matrix or extracts the diagonal of a matrix.
//
// If x is a matrix (*mat.Dense), it extracts the diagonal elements as a slice of float64.
// If x is a slice of float64, it creates a diagonal matrix with those elements.
// If x is an int, it creates an identity matrix of that size.
//
// For creating diagonal matrices, nrow and ncol can be specified to control the size.
// If not specified, it uses the length of the slice or the value of x.
// names is currently not used, reserved for future use.
//
// This function mimics the behavior of R's diag function.
func Diag(x interface{}, nrow, ncol int, names bool) interface{} {
	switch v := x.(type) {
	case *mat.Dense:
		// Extract diagonal
		r, c := v.Dims()
		size := min(r, c)
		diag := make([]float64, size)
		for i := 0; i < size; i++ {
			diag[i] = v.At(i, i)
		}
		return diag
	case []float64:
		// Create diagonal matrix
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
		return matrix
	case int:
		// Create identity matrix
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
		return matrix
	case float64:
		// Create identity matrix from float (like R diag(5.0))
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
		return matrix
	case nil:
		// Create identity matrix, default size 1 or from nrow
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
		return matrix
	default:
		panic("unsupported type for Diag")
	}
}
