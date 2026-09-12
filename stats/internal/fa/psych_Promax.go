// fa/psych_Promax.go
package fa

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

// Promax performs Promax rotation.
//
// psych has two Promax paths and they differ in the varimax they start from.
// psych::Promax, which fa() reaches through kaiser(), pre-rotates with
// GPArotation::Varimax since 2.6.5; stats::promax, which principal() uses,
// pre-rotates with stats::varimax and its Kaiser row normalisation. normalize
// selects between them: false is psych::Promax, true is stats::promax. The
// target-matrix least-squares step after that is the same in both.
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

	// Step 1: the varimax pre-rotation. psych::Promax used stats::varimax
	// until 2.6.5, which replaced it with GPArotation::Varimax from the
	// identity ("replaced with GPArotation Varimax 5/9/26", psych/R/Promax.R):
	// no Kaiser normalisation, eps 1e-5. The two agree on data with a clear
	// varimax optimum, but on a nearly flat criterion they stop at different
	// angles, and the power step below raises that to the fourth power —
	// measured on the parity suite's ten-row table, loading[0,0] was 0.700
	// against psych's 0.625 before this followed psych.
	var xx, rotmatVarimax *mat.Dense
	if normalize {
		// stats::promax: xx <- varimax(x), Kaiser normalisation on.
		var err error
		xx, rotmatVarimax, err = KaiserVarimaxWithRotationMatrix(x, true, 1000, 1e-5)
		if err != nil {
			return map[string]any{
				"error":       fmt.Sprintf("varimax pre-rotation failed: %v", err),
				"diagnostics": diagnostics,
			}
		}
	} else {
		vx := Varimax(x, false, 1e-5, 1000)
		if msg, _ := vx["error"].(string); msg != "" {
			return map[string]any{
				"error":       fmt.Sprintf("varimax pre-rotation failed: %s", msg),
				"diagnostics": diagnostics,
			}
		}
		var ok, ok2 bool
		xx, ok = vx["loadings"].(*mat.Dense)
		rotmatVarimax, ok2 = vx["rotmat"].(*mat.Dense)
		if !ok || !ok2 || xx == nil || rotmatVarimax == nil {
			return map[string]any{
				"error":       "varimax pre-rotation returned no loadings or rotation matrix",
				"diagnostics": diagnostics,
			}
		}
	}

	// x <- xx$loadings
	x = xx
	var err error

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
	err = U.Solve(&XtX, &XtQ)
	if err != nil {
		diagnostics["converged"] = false
		diagnostics["matrixInversionErrors"] = append(
			diagnostics["matrixInversionErrors"].([]string),
			"failed to solve for U coefficients: "+err.Error())
		return map[string]any{
			"error":       fmt.Sprintf("failed to solve for promax U coefficients: %v", err),
			"diagnostics": diagnostics,
		}
	}

	// Step 4: Compute diagonal matrix d <- diag(solve(t(U) %*% U))
	var UtU mat.Dense
	UtU.Mul(U.T(), &U)
	var UtUInv mat.Dense
	err = UtUInv.Inverse(&UtU)
	d := make([]float64, cols)
	if err != nil {
		diagnostics["converged"] = false
		diagnostics["matrixInversionErrors"] = append(
			diagnostics["matrixInversionErrors"].([]string),
			"failed to invert UtU: "+err.Error())
		return map[string]any{
			"error":       fmt.Sprintf("failed to invert promax UtU: %v", err),
			"diagnostics": diagnostics,
		}
	}
	for i := 0; i < cols; i++ {
		d[i] = UtUInv.At(i, i)
		if d[i] <= 0 || math.IsNaN(d[i]) {
			diagnostics["converged"] = false
			return map[string]any{
				"error":       fmt.Sprintf("invalid promax normalization diagonal at %d: %g", i, d[i]),
				"diagnostics": diagnostics,
			}
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

	// Step 7: U <- Th %*% U, Th being Varimax's rotation matrix (orthogonal, so t(solve(Th)) = Th)
	var rotmat mat.Dense
	rotmat.Mul(rotmatVarimax, &U)

	// Step 8: ui <- solve(U)
	ui, err := invertDense(&rotmat)
	if err != nil {
		diagnostics["converged"] = false
		diagnostics["matrixInversionErrors"] = append(
			diagnostics["matrixInversionErrors"].([]string),
			"failed to invert final rotation matrix: "+err.Error())
		return map[string]any{
			"error":       fmt.Sprintf("failed to invert promax rotation matrix: %v", err),
			"diagnostics": diagnostics,
		}
	}

	// Step 9: Phi <- ui %*% t(ui)
	var Phi mat.Dense
	Phi.Mul(ui, ui.T())

	return map[string]any{
		"loadings":    &z,
		"rotmat":      &rotmat,
		"Phi":         &Phi,
		"diagnostics": diagnostics,
	}
}
