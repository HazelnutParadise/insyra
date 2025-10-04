// fa/GPArotation_vgQ_pst.go
package fa

import (
	"errors"

	"gonum.org/v1/gonum/mat"
)

// vgQPst computes the objective and gradient for partially specified target rotation.
// Mirrors GPArotation::vgQ.pst(L, W = NULL, Target = NULL)
//
// Returns: Gq (gradient), f (objective), method
func vgQPst(L, W, Target *mat.Dense) (Gq *mat.Dense, f float64, method string, err error) {
	if W == nil || Target == nil {
		err = errors.New("arguments W and Target must be specified")
		return
	}

	p, k := L.Dims()

	// Btilde = W * Target
	Btilde := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			Btilde.Set(i, j, W.At(i, j)*Target.At(i, j))
		}
	}

	// WL = W * L
	WL := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			WL.Set(i, j, W.At(i, j)*L.At(i, j))
		}
	}

	// diff = WL - Btilde
	diff := mat.NewDense(p, k, nil)
	diff.Sub(WL, Btilde)

	// Gq = 2 * diff
	Gq = mat.NewDense(p, k, nil)
	Gq.Scale(2, diff)

	// f = sum(diff^2)
	f = 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			d := diff.At(i, j)
			f += d * d
		}
	}

	method = "Partially specified target"
	return
}
