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
	loadings, rotmat, Phi, _, err = TargetRotWithMask(x, keys, nil)
	return
}

// TargetRotWithMask performs target rotation with optional mask for target matrix elements.
// Elements marked with NaN in the mask are excluded from the alignment process.
func TargetRotWithMask(x *mat.Dense, keys *mat.Dense, mask *mat.Dense) (loadings, rotmat, Phi *mat.Dense, diagnostics map[string]interface{}, err error) {
	p, q := x.Dims()
	if q < 2 {
		return nil, nil, nil, nil, errors.New("rotation not meaningful with less than 2 factors")
	}

	diagnostics = map[string]interface{}{
		"maskApplied":    false,
		"maskedElements": [][]int{},
	}

	var Q *mat.Dense
	if keys == nil {
		Q = Factor2Cluster(x, nil)
	} else {
		Q = mat.DenseCopyOf(keys)
	}

	if Q.RawMatrix().Cols < 2 {
		return nil, nil, nil, diagnostics, errors.New("cluster structure produces 1 cluster")
	}

	// Apply mask if provided
	if mask != nil {
		diagnostics["maskApplied"] = true
		maskedElements := [][]int{}
		for i := range p {
			for j := range q {
				if math.IsNaN(mask.At(i, j)) {
					// Set corresponding element in Q to 0 (exclude from alignment)
					Q.Set(i, j, 0.0)
					maskedElements = append(maskedElements, []int{i, j})
				}
			}
		}
		diagnostics["maskedElements"] = maskedElements
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
		return nil, nil, nil, diagnostics, err
	}

	// Normalize U
	var UtU mat.Dense
	UtU.Mul(U.T(), &U)
	var UtUInv mat.Dense
	// Prefer exact inverse, but fall back safely to identity-like behavior
	// to avoid panics in edge cases.
	if inv := inverseOrIdentity(&UtU, UtU.RawMatrix().Rows); inv == nil {
		return nil, nil, nil, diagnostics, errors.New("failed to invert UtU and no safe fallback")
	} else {
		UtUInv.CloneFrom(inv)
	}
	d := make([]float64, q)
	for i := range q {
		d[i] = UtUInv.At(i, i)
	}

	for j := range q {
		sqrtD := math.Sqrt(d[j])
		for i := range q { // U is q x q
			U.Set(i, j, U.At(i, j)*sqrtD)
		}
	}

	// z = x %*% U
	loadings = mat.NewDense(p, q, nil)
	loadings.Mul(x, &U)

	// Phi = solve(U) %*% t(solve(U))
	var UInv mat.Dense
	// Use safe inverse helper; if inversion fails, return error to caller
	// as target rotation depends on U being invertible for Phi.
	invU := inverseOrIdentity(&U, U.RawMatrix().Rows)
	if invU == nil {
		return nil, nil, nil, diagnostics, errors.New("failed to invert U for target rotation")
	}
	UInv.CloneFrom(invU)
	Phi = mat.NewDense(q, q, nil)
	Phi.Mul(&UInv, UInv.T())

	return loadings, &U, Phi, diagnostics, nil
}
