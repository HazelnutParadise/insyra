// fa/psych_target_rot.go
package fa

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
)

// TargetRot performs target rotation on an unrotated loading matrix x.
// Mirrors psych::target.rot exactly
func TargetRot(x *mat.Dense, keys *mat.Dense) (loadings, rotmat, Phi *mat.Dense, err error) {
	p, q := x.Dims()
	if q < 2 {
		return nil, nil, nil, errors.New("rotation not meaningful with less than 2 factors")
	}

	var Q *mat.Dense
	if keys == nil {
		Q = Factor2Cluster(x)
	} else {
		Q = mat.DenseCopyOf(keys)
	}

	if Q.RawMatrix().Cols < 2 {
		return nil, nil, nil, errors.New("cluster structure produces 1 cluster")
	}

	// U = coefficients from lm.fit(x, Q)
	// Simplified: U = solve(t(x) %*% x) %*% t(x) %*% Q
	var XtX mat.Dense
	XtX.Mul(x.T(), x)
	var XtQ mat.Dense
	XtQ.Mul(x.T(), Q)
	var U mat.Dense
	err = U.Solve(&XtX, &XtQ)
	if err != nil {
		return nil, nil, nil, err
	}

	// Normalize U
	var UtU mat.Dense
	UtU.Mul(U.T(), &U)
	var UtUInv mat.Dense
	err = UtUInv.Inverse(&UtU)
	if err != nil {
		return nil, nil, nil, err
	}
	d := make([]float64, q)
	for i := 0; i < q; i++ {
		d[i] = UtUInv.At(i, i)
	}

    for j := 0; j < q; j++ {
        sqrtD := math.Sqrt(d[j])
        for i := 0; i < q; i++ { // U is q x q
            U.Set(i, j, U.At(i, j)*sqrtD)
        }
    }

	// z = x %*% U
	loadings = mat.NewDense(p, q, nil)
	loadings.Mul(x, &U)

	// Phi = solve(U) %*% t(solve(U))
	var UInv mat.Dense
	err = UInv.Inverse(&U)
	if err != nil {
		return nil, nil, nil, err
	}
	Phi = mat.NewDense(q, q, nil)
	Phi.Mul(&UInv, UInv.T())

	return loadings, &U, Phi, nil
}
