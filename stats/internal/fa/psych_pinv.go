// fa/psych_pinv.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// Pinv computes the Moore-Penrose pseudo-inverse of a matrix.
// Mirrors psych::Pinv
func Pinv(X *mat.Dense) *mat.Dense {
	var inv mat.Dense
	err := inv.Inverse(X)
	if err != nil {
		return nil
	}
	return &inv
}
