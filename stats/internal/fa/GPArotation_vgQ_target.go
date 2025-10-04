// fa/GPArotation_vgQ_target.go
package fa

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQTarget computes the objective and gradient for target rotation.
// Mirrors GPArotation::vgQ.target(L, Target = NULL)
//
// Returns: Gq (gradient), f (objective), method
func vgQTarget(L, Target *mat.Dense) (Gq *mat.Dense, f float64, method string, err error) {
	if Target == nil {
		err = errors.New("argument Target must be specified")
		return
	}

	p, k := L.Dims()

	// diff = L - Target
	diff := mat.NewDense(p, k, nil)
	diff.Sub(L, Target)

	// Gq = 2 * diff, with NaN -> 0
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			d := diff.At(i, j)
			if math.IsNaN(d) {
				Gq.Set(i, j, 0)
			} else {
				Gq.Set(i, j, 2*d)
			}
		}
	}

	// f = sum(diff^2, na.rm = TRUE)
	f = 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			d := diff.At(i, j)
			if !math.IsNaN(d) {
				f += d * d
			}
		}
	}

	method = "Target rotation"
	return
}
