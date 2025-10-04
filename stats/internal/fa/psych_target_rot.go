// fa/psych_target_rot.go
package fa

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
)

// TargetRot performs target rotation on an unrotated loading matrix x.
// It mirrors psych::target.rot behavior:
// - If keys is nil, use factor2cluster to generate Q
// - Else use keys as Q
// - U = lm.fit(x, Q)$coefficients
// - d = diag(solve(t(U) %*% U))
// - U = U %*% diag(sqrt(d))
// - z = x %*% U
// - Phi = solve(U) %*% t(solve(U))
//
// Returns: loadings, rotmat, Phi
func TargetRot(x *mat.Dense, keys *mat.Dense) (loadings, rotmat, Phi *mat.Dense, err error) {
	p, q := x.Dims()
	if q < 2 {
		return nil, nil, nil, errors.New("rotation not meaningful with less than 2 factors")
	}

	var Q *mat.Dense
	if keys == nil {
		// Use factor2cluster(x) - need to implement or assume
		// For now, placeholder
		return nil, nil, nil, errors.New("keys is nil, factor2cluster not implemented")
	} else {
		Q = mat.DenseCopyOf(keys)
	}

	if Q.RawMatrix().Cols < 2 {
		return nil, nil, nil, errors.New("cluster structure produces 1 cluster")
	}

	// U = lm.fit(x, Q)$coefficients = solve(t(x) %*% x) %*% t(x) %*% Q
	var XtX mat.Dense
	XtX.Mul(x.T(), x)
	var XtQ mat.Dense
	XtQ.Mul(x.T(), Q)
	var U mat.Dense
	err = U.Solve(&XtX, &XtQ)
	if err != nil {
		return nil, nil, nil, err
	}

	// d = diag(solve(t(U) %*% U))
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

	// U = U %*% diag(sqrt(d))
	for j := 0; j < q; j++ {
		sqrtD := math.Sqrt(d[j])
		for i := 0; i < p; i++ {
			U.Set(i, j, U.At(i, j)*sqrtD)
		}
	}

	// z = x %*% U
	loadings = mat.NewDense(p, q, nil)
	loadings.Mul(x, &U)

	// ui = solve(U)
	var ui mat.Dense
	err = ui.Inverse(&U)
	if err != nil {
		return nil, nil, nil, err
	}

	// Phi = ui %*% t(ui)
	Phi = mat.NewDense(q, q, nil)
	Phi.Mul(&ui, ui.T())

	return loadings, &U, Phi, nil
}
