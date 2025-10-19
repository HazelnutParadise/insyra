// fa/psych_Promax.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Promax performs Promax rotation.
// Mirrors psych::Promax exactly
func Promax(x *mat.Dense, m int, normalize bool) map[string]any {
	rows, cols := x.Dims()

	diagnostics := map[string]interface{}{
		"converged":             true,
		"iterations":            0, // Promax doesn't iterate, but included for consistency
		"residualNorm":          0.0,
		"matrixInversionErrors": []string{},
	}

	if cols < 2 {
		return map[string]any{
			"loadings":    x,
			"rotmat":      identityMatrix(cols),
			"Phi":         identityMatrix(cols),
			"diagnostics": diagnostics,
		}
	}

	// Step 1: Perform varimax rotation
	varimaxResult := Varimax(x, normalize, 1e-5, 1000)
	xx := varimaxResult["loadings"].(*mat.Dense)
	rotmatVarimax := varimaxResult["rotmat"].(*mat.Dense)

	// x <- xx$loadings
	x = xx

	// Step 2: Compute Q matrix: Q <- x * abs(x)^(m - 1)
	Q := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := x.At(i, j)
			Q.Set(i, j, val*math.Pow(math.Abs(val), float64(m-1)))
		}
	}

	// Step 3: Solve for U: U <- lm.fit(x, Q)$coefficients
	// This is solve(t(x) %*% x) %*% t(x) %*% Q
	var XtX mat.Dense
	XtX.Mul(x.T(), x)
	var XtQ mat.Dense
	XtQ.Mul(x.T(), Q)
	var U mat.Dense
	err := U.Solve(&XtX, &XtQ)
	if err != nil {
		// Handle singular matrix - use approximation like R
		diagnostics["converged"] = false
		diagnostics["matrixInversionErrors"] = append(
			diagnostics["matrixInversionErrors"].([]string),
			"failed to solve for U coefficients: "+err.Error())
		return map[string]any{
			"loadings":    xx,
			"rotmat":      rotmatVarimax,
			"Phi":         identityMatrix(cols),
			"diagnostics": diagnostics,
		}
	}

	// Step 4: Compute diagonal matrix d <- diag(solve(t(U) %*% U))
	var UtU mat.Dense
	UtU.Mul(U.T(), &U)
	var UtUInv mat.Dense

	// Attempt regular inverse first; keep existing fallback logic below.
	err = UtUInv.Inverse(&UtU)
	d := make([]float64, cols)
	if err != nil {
		// Match R's eigenvalue approximation for singular matrices
		// Use simplified regularization approach
		regularization := 1e-8
		for i := 0; i < cols; i++ {
			UtU.Set(i, i, UtU.At(i, i)+regularization)
		}
		err = UtUInv.Inverse(&UtU)
		if err != nil {
			// Final fallback
			diagnostics["converged"] = false
			diagnostics["matrixInversionErrors"] = append(
				diagnostics["matrixInversionErrors"].([]string),
				"failed to invert UtU after regularization: "+err.Error())
			for i := 0; i < cols; i++ {
				d[i] = 1.0
			}
		} else {
			for i := 0; i < cols; i++ {
				d[i] = UtUInv.At(i, i)
			}
		}
	} else {
		for i := 0; i < cols; i++ {
			d[i] = UtUInv.At(i, i)
		}
	}

	// Step 5: U <- U %*% diag(sqrt(d))
	for j := 0; j < cols; j++ {
		sqrtD := math.Sqrt(d[j])
		for i := 0; i < cols; i++ {
			U.Set(i, j, U.At(i, j)*sqrtD)
		}
	}

	// Step 6: z <- x %*% U
	var z mat.Dense
	z.Mul(x, &U)

	// Step 7: U <- xx$rotmat %*% U
	var rotmat mat.Dense
	rotmat.Mul(rotmatVarimax, &U)

	// Step 8: ui <- solve(U) using a safe inverse helper
	uiDense := inverseOrIdentity(&rotmat, rotmat.RawMatrix().Rows)
	var ui mat.Dense
	ui.CloneFrom(uiDense)

	// Step 9: Phi <- ui %*% t(ui)
	var Phi mat.Dense
	Phi.Mul(&ui, ui.T())

	return map[string]any{
		"loadings":    &z,
		"rotmat":      &rotmat,
		"Phi":         &Phi,
		"diagnostics": diagnostics,
	}
}
