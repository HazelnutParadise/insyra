package stats

import (
	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
)

// Diag creates a diagonal matrix or extracts the diagonal of a matrix.
//
// If x is a matrix (*mat.Dense), it extracts the diagonal elements as a slice of float64.
// If x is a slice of float64, it creates a diagonal matrix with those elements.
// If x is an int, it creates an identity matrix of that size.
//
// For creating diagonal matrices, nrow and ncol can be optionally specified to control the size.
// If not specified, it uses the length of the slice or the value of x.
//
// Usage:
//
//	Diag(x)              // Use default sizing
//	Diag(x, nrow)         // Specify nrow, ncol = nrow
//	Diag(x, nrow, ncol)   // Specify both nrow and ncol
//
// This function mimics the behavior of R's diag function.
func Diag(x any, dims ...int) any {
	// Parse optional dimensions
	var nrow, ncol int
	switch len(dims) {
	case 0:
		// No dimensions specified
	case 1:
		nrow = dims[0]
		ncol = dims[0]
	case 2:
		nrow = dims[0]
		ncol = dims[1]
	default:
		panic("too many dimensions specified")
	}

	switch v := x.(type) {
	case *mat.Dense:
		// Extract diagonal
		r, c := v.Dims()
		size := min(r, c)
		diag := make([]float64, size)
		for i := range size {
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
		insyra.LogWarning("stats", "Diag", "Unsupported type for Diag function, returning nil")
		return nil
	}
}
