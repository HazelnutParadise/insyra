// fa/GPArotation_vgQ_target.go
package fa

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

// vgQTarget computes the objective and gradient for target rotation.
// Mirrors GPArotation::vgQ.target(L, Target = NULL)
//
// if (is.null(Target)) stop("argument Target must be specified.")
// Gq <- 2 * (L - Target)
// Gq[is.na(Gq)] <- 0
// f <- sum((L - Target)^2, na.rm = TRUE)
//
// Returns: Gq (gradient), f (objective), method, error
func vgQTarget(L *mat.Dense, Target *mat.Dense) (Gq *mat.Dense, f float64, method string, err error) {
	if Target == nil {
		return nil, 0, "", fmt.Errorf("argument Target must be specified")
	}

	p, q := L.Dims()
	tp, tq := Target.Dims()
	if p != tp || q != tq {
		return nil, 0, "", fmt.Errorf("L and Target must have the same dimensions")
	}

	// L_minus_Target = L - Target
	L_minus_Target := mat.NewDense(p, q, nil)
	L_minus_Target.Sub(L, Target)

	// Gq = 2 * (L - Target)
	Gq = mat.NewDense(p, q, nil)
	for i := range p {
		for j := range q {
			diff := L_minus_Target.At(i, j)
			if diff != diff { // NaN check
				Gq.Set(i, j, 0.0)
			} else {
				Gq.Set(i, j, 2.0*diff)
			}
		}
	}

	// f = sum((L - Target)^2, na.rm = TRUE)
	f = 0.0
	for i := range p {
		for j := range q {
			diff := L_minus_Target.At(i, j)
			if diff == diff { // not NaN
				f += diff * diff
			}
		}
	}

	method = "Target rotation"
	return
}
