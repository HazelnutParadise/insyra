// fa/GPArotation_vgQ_target.go
package fa

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

// vgQTarget computes the objective and gradient for target rotation.
// Mirrors GPArotation::vgQ.target(L, Target = NULL)
//
// Target rotation minimizes the squared differences between loadings and target values.
// Missing values (NaN) in the target matrix are ignored.
//
// Gq <- 2 * (L - Target)  [with NaN handling]
// Gq[is.na(Gq)] <- 0
// f <- sum((L - Target)^2, na.rm = TRUE)
//
// Returns: Gq (gradient), f (objective), method, error
func vgQTarget(L *mat.Dense, Target *mat.Dense) (Gq *mat.Dense, f float64, method string, err error) {
	if Target == nil {
		return nil, 0, "", fmt.Errorf("argument Target must be specified")
	}

	rows, cols := L.Dims()
	targetRows, targetCols := Target.Dims()
	if rows != targetRows || cols != targetCols {
		return nil, 0, "", fmt.Errorf("L and Target must have the same dimensions")
	}

	// Compute difference: L - Target
	diff := mat.NewDense(rows, cols, nil)
	diff.Sub(L, Target)

	// Compute gradient: Gq = 2 * (L - Target), with NaN handling
	Gq = mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := diff.At(i, j)
			if val != val { // NaN check (NaN != NaN is true)
				Gq.Set(i, j, 0.0)
			} else {
				Gq.Set(i, j, 2.0*val)
			}
		}
	}

	// Compute objective: f = sum((L - Target)^2), ignoring NaN values
	f = 0.0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := diff.At(i, j)
			if val == val { // not NaN (NaN == NaN is false)
				f += val * val
			}
		}
	}

	method = "Target rotation"
	return
}
