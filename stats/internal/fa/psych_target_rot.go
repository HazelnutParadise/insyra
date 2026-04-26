// fa/psych_target_rot.go
package fa

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
)

// TargetRot performs target rotation on an unrotated loading matrix x against
// an automatically derived cluster target. Mirrors psych::target.rot.
func TargetRot(x *mat.Dense) (loadings, rotmat, Phi *mat.Dense, err error) {
	p, q := x.Dims()
	if q < 2 {
		return nil, nil, nil, errors.New("rotation not meaningful with less than 2 factors")
	}

	Q := Factor2Cluster(x)
	if Q.RawMatrix().Cols < 2 {
		return nil, nil, nil, errors.New("cluster structure produces 1 cluster")
	}

	// U = coefficients from lm.fit(x, Q): U = solve(t(x) %*% x) %*% t(x) %*% Q
	var XtX mat.Dense
	XtX.Mul(x.T(), x)
	var XtQ mat.Dense
	XtQ.Mul(x.T(), Q)
	var U mat.Dense
	if err = U.Solve(&XtX, &XtQ); err != nil {
		return nil, nil, nil, err
	}

	// Normalize U columns by sqrt(diag(solve(U'U)))
	var UtU mat.Dense
	UtU.Mul(U.T(), &U)
	UtUInv, err := invertDense(&UtU)
	if err != nil {
		return nil, nil, nil, err
	}
	for j := 0; j < q; j++ {
		sqrtD := math.Sqrt(UtUInv.At(j, j))
		for i := 0; i < q; i++ {
			U.Set(i, j, U.At(i, j)*sqrtD)
		}
	}

	loadings = mat.NewDense(p, q, nil)
	loadings.Mul(x, &U)

	UInv, err := invertDense(&U)
	if err != nil {
		return nil, nil, nil, err
	}
	Phi = mat.NewDense(q, q, nil)
	Phi.Mul(UInv, UInv.T())

	return loadings, &U, Phi, nil
}
