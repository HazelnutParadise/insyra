// fa/kaiser_varimax.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// KaiserVarimax performs Varimax rotation using the classic Jacobi rotation method
// as described in Kaiser (1958) and implemented in SPSS.
//
// This method differs from GPArotation's gradient-based approach:
// - Uses pairwise Givens rotations instead of gradient projection
// - Iterates through all factor pairs until convergence
// - More closely matches SPSS output
//
// References:
// Kaiser, H. F. (1958). The varimax criterion for analytic rotation in factor analysis.
// Psychometrika, 23(3), 187-200.
//
// Algorithm:
// 1. Apply Kaiser normalization (row normalize by sqrt of communality)
// 2. For each pair of factors (p, q):
//   - Compute optimal rotation angle θ that maximizes variance of squared loadings
//   - Apply Givens rotation to factors p and q
//
// 3. Repeat until convergence
// 4. Denormalize by multiplying back by sqrt of communality
func KaiserVarimax(A *mat.Dense, normalize bool, maxIter int, epsilon float64) (*mat.Dense, error) {
	p, q := A.Dims()

	// Default parameters
	if maxIter == 0 {
		maxIter = 1000
	}
	if epsilon == 0 {
		epsilon = 1e-6
	}

	// Copy input matrix
	L := mat.DenseCopyOf(A)

	// Step 1: Kaiser normalization (if requested)
	var h []float64 // communalities (sqrt)
	if normalize {
		h = make([]float64, p)
		for i := 0; i < p; i++ {
			sum := 0.0
			for j := 0; j < q; j++ {
				val := L.At(i, j)
				sum += val * val
			}
			h[i] = math.Sqrt(sum)

			// Normalize row i
			if h[i] > 1e-10 {
				for j := 0; j < q; j++ {
					L.Set(i, j, L.At(i, j)/h[i])
				}
			}
		}
	}

	// Step 2: Jacobi rotation
	for iter := 0; iter < maxIter; iter++ {
		maxChange := 0.0

		// Iterate through all pairs of factors
		for j := 0; j < q-1; j++ {
			for k := j + 1; k < q; k++ {
				// Extract columns j and k
				A_j := make([]float64, p)
				A_k := make([]float64, p)
				for i := 0; i < p; i++ {
					A_j[i] = L.At(i, j)
					A_k[i] = L.At(i, k)
				}

				// Compute u and v for the rotation angle
				// u = A_j^2 - A_k^2
				// v = 2 * A_j * A_k
				var sumU, sumV, sumU2, sumV2 float64
				for i := 0; i < p; i++ {
					u := A_j[i]*A_j[i] - A_k[i]*A_k[i]
					v := 2.0 * A_j[i] * A_k[i]

					sumU += u
					sumV += v
					sumU2 += u * u
					sumV2 += v * v
				}

				// Compute rotation angle
				// tan(4θ) = (Σv - Σu/p) / (Σ(u^2 - v^2) / (2p))
				// Simplified SPSS formula:
				// tan(4θ) = (2 * Σv) / Σ(u^2 - v^2)
				numerator := 2.0 * sumV
				denominator := sumU2 - sumV2

				var theta float64
				if math.Abs(denominator) < 1e-10 {
					theta = 0
				} else {
					// θ = atan(numerator / denominator) / 4
					theta = math.Atan2(numerator, denominator) / 4.0
				}

				// Track maximum change
				if math.Abs(theta) > maxChange {
					maxChange = math.Abs(theta)
				}

				// Apply Givens rotation if angle is significant
				if math.Abs(theta) > epsilon {
					cosTheta := math.Cos(theta)
					sinTheta := math.Sin(theta)

					for i := 0; i < p; i++ {
						a_j := L.At(i, j)
						a_k := L.At(i, k)

						// Givens rotation:
						// [a_j'] = [cos θ  -sin θ] [a_j]
						// [a_k']   [sin θ   cos θ] [a_k]
						L.Set(i, j, cosTheta*a_j-sinTheta*a_k)
						L.Set(i, k, sinTheta*a_j+cosTheta*a_k)
					}
				}
			}
		}

		// Check convergence
		if maxChange < epsilon {
			break
		}
	}

	// Step 3: Denormalize (if we normalized)
	if normalize {
		for i := 0; i < p; i++ {
			if h[i] > 1e-10 {
				for j := 0; j < q; j++ {
					L.Set(i, j, L.At(i, j)*h[i])
				}
			}
		}
	}

	return L, nil
}

// KaiserVarimaxWithRotationMatrix returns both the rotated loadings and the rotation matrix
func KaiserVarimaxWithRotationMatrix(A *mat.Dense, normalize bool, maxIter int, epsilon float64) (*mat.Dense, *mat.Dense, error) {
	p, q := A.Dims()

	// Default parameters
	if maxIter == 0 {
		maxIter = 1000
	}
	if epsilon == 0 {
		epsilon = 1e-6
	}

	// Copy input matrix
	L := mat.DenseCopyOf(A)

	// Initialize rotation matrix as identity
	T := mat.NewDense(q, q, nil)
	for i := 0; i < q; i++ {
		T.Set(i, i, 1.0)
	}

	// Step 1: Kaiser normalization (if requested)
	var h []float64 // communalities (sqrt)
	if normalize {
		h = make([]float64, p)
		for i := 0; i < p; i++ {
			sum := 0.0
			for j := 0; j < q; j++ {
				val := L.At(i, j)
				sum += val * val
			}
			h[i] = math.Sqrt(sum)

			// Normalize row i
			if h[i] > 1e-10 {
				for j := 0; j < q; j++ {
					L.Set(i, j, L.At(i, j)/h[i])
				}
			}
		}
	}

	// Step 2: Jacobi rotation
	for iter := 0; iter < maxIter; iter++ {
		maxChange := 0.0

		// Iterate through all pairs of factors
		for j := 0; j < q-1; j++ {
			for k := j + 1; k < q; k++ {
				// Extract columns j and k
				A_j := make([]float64, p)
				A_k := make([]float64, p)
				for i := 0; i < p; i++ {
					A_j[i] = L.At(i, j)
					A_k[i] = L.At(i, k)
				}

				// Compute u and v for the rotation angle
				var sumU, sumV, sumU2, sumV2 float64
				for i := 0; i < p; i++ {
					u := A_j[i]*A_j[i] - A_k[i]*A_k[i]
					v := 2.0 * A_j[i] * A_k[i]

					sumU += u
					sumV += v
					sumU2 += u * u
					sumV2 += v * v
				}

				// Compute rotation angle
				numerator := 2.0 * sumV
				denominator := sumU2 - sumV2

				var theta float64
				if math.Abs(denominator) < 1e-10 {
					theta = 0
				} else {
					theta = math.Atan2(numerator, denominator) / 4.0
				}

				// Track maximum change
				if math.Abs(theta) > maxChange {
					maxChange = math.Abs(theta)
				}

				// Apply Givens rotation if angle is significant
				if math.Abs(theta) > epsilon {
					cosTheta := math.Cos(theta)
					sinTheta := math.Sin(theta)

					// Rotate loadings
					for i := 0; i < p; i++ {
						a_j := L.At(i, j)
						a_k := L.At(i, k)
						L.Set(i, j, cosTheta*a_j-sinTheta*a_k)
						L.Set(i, k, sinTheta*a_j+cosTheta*a_k)
					}

					// Update rotation matrix
					for i := 0; i < q; i++ {
						t_j := T.At(i, j)
						t_k := T.At(i, k)
						T.Set(i, j, cosTheta*t_j-sinTheta*t_k)
						T.Set(i, k, sinTheta*t_j+cosTheta*t_k)
					}
				}
			}
		}

		// Check convergence
		if maxChange < epsilon {
			break
		}
	}

	// Step 3: Denormalize (if we normalized)
	if normalize {
		for i := 0; i < p; i++ {
			if h[i] > 1e-10 {
				for j := 0; j < q; j++ {
					L.Set(i, j, L.At(i, j)*h[i])
				}
			}
		}
	}

	return L, T, nil
}
