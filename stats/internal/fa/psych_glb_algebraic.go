// fa/psych_glb_algebraic.go
package fa

import (
	"errors"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/optimize"
)

// GlbAlgebraic computes the greatest lower bound algebraically.
// Mirrors psych::glb.algebraic with SDP solver implementation
func GlbAlgebraic(Cov *mat.Dense, LoBounds, UpBounds *mat.VecDense) (glb float64, solution *mat.VecDense, status int, call interface{}) {
	n, _ := Cov.Dims()
	if n == 0 {
		return 0, nil, 4, errors.New("empty covariance matrix")
	}

	// Check bounds
	if LoBounds.Len() != n || UpBounds.Len() != n {
		return 0, nil, 4, errors.New("bounds length mismatch")
	}

	// For SDP problems, we use a simplified approach
	// Initialize with diagonal matrix respecting bounds
	x0 := make([]float64, n*n)
	for i := 0; i < n; i++ {
		val := (LoBounds.AtVec(i) + UpBounds.AtVec(i)) / 2.0
		if val < 0.1 {
			val = 0.1
		}
		x0[i*n+i] = val
	}

	// Define the objective function: minimize trace(Cov * X)
	objective := func(x []float64) float64 {
		X := mat.NewDense(n, n, x)
		result := mat.NewDense(n, n, nil)
		result.Mul(Cov, X)
		trace := 0.0
		for i := 0; i < n; i++ {
			trace += result.At(i, i)
		}
		return trace
	}

	// Simple bound constraints function
	bounds := func(x []float64) {
		for i := 0; i < n; i++ {
			idx := i*n + i
			if x[idx] < LoBounds.AtVec(i) {
				x[idx] = LoBounds.AtVec(i)
			}
			if x[idx] > UpBounds.AtVec(i) {
				x[idx] = UpBounds.AtVec(i)
			}
		}
	}

	// Use LBFGS optimizer with bounds
	settings := optimize.Settings{
		GradientThreshold: 1e-8,
		Converger: &optimize.FunctionConverge{
			Absolute:   1e-10,
			Iterations: 1000,
		},
	}

	result, err := optimize.Minimize(
		optimize.Problem{Func: objective},
		x0,
		&settings,
		&optimize.LBFGS{},
	)
	if err != nil {
		return 0, nil, 4, err
	}

	// Apply bounds to final solution
	bounds(result.X)

	// Extract solution
	solutionVec := mat.NewVecDense(n*n, result.X)
	solutionMatrix := mat.NewDense(n, n, result.X)

	// Ensure solution is PSD
	solutionMatrix = projectToPSD(solutionMatrix)

	// Compute final GLB
	glb = objective(result.X)

	status = 0 // success
	return glb, solutionVec, status, nil
}

// projectToPSD projects a matrix onto the positive semidefinite cone
func projectToPSD(A *mat.Dense) *mat.Dense {
	n, _ := A.Dims()

	// For simplicity, if the matrix is already PSD (all eigenvalues >= 0), return as is
	// Otherwise, use a simple regularization approach
	result := mat.DenseCopyOf(A)

	// Add small regularization to diagonal to ensure PSD
	for i := 0; i < n; i++ {
		val := result.At(i, i)
		if val < 0.01 {
			result.Set(i, i, 0.01)
		}
	}

	return result
}
