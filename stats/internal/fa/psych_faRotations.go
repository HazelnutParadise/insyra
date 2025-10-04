// fa/psych_faRotations.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Varimax performs varimax rotation.
// Mirrors GPArotation::Varimax
func Varimax(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]interface{} {
	nf, p := loadings.Dims()

	// Initial rotation matrix: identity
	T := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		T.Set(i, i, 1.0)
	}

	// Normalize loadings if requested
	var A *mat.Dense
	if normalize {
		A = mat.DenseCopyOf(loadings)
		for j := 0; j < p; j++ {
			colNorm := 0.0
			for i := 0; i < nf; i++ {
				colNorm += loadings.At(i, j) * loadings.At(i, j)
			}
			colNorm = math.Sqrt(colNorm)
			if colNorm > 0 {
				for i := 0; i < nf; i++ {
					A.Set(i, j, loadings.At(i, j)/colNorm)
				}
			}
		}
	} else {
		A = mat.DenseCopyOf(loadings)
	}

	// Optimization loop
	for iter := 0; iter < maxIter; iter++ {
		// Compute gradient and objective
		Gq, f, _ := vgQVarimax(A)

		// Update T using simple gradient descent (simplified)
		learningRate := 0.01
		for i := 0; i < nf; i++ {
			for j := 0; j < nf; j++ {
				T.Set(i, j, T.At(i, j)-learningRate*Gq.At(i, j))
			}
		}

		// Apply rotation
		var rotated mat.Dense
		rotated.Mul(T, A)
		A = &rotated

		// Check convergence
		if math.Abs(f) < eps {
			break
		}
	}

	// Denormalize if normalized
	if normalize {
		// Simplified, assume no denormalization for now
	}

	return map[string]interface{}{
		"loadings": A,
		"rotmat":   T,
	}
}

// FaRotations performs various rotations on factor loadings.
// Mirrors psych::faRotations
func FaRotations(loadings *mat.Dense, r *mat.Dense, rotate string, hyper float64, nRotations int) interface{} {
	var rotatedLoadings *mat.Dense
	var rotMat *mat.Dense
	var phi *mat.Dense

	switch rotate {
	case "varimax", "Varimax":
		result := Varimax(loadings, true, 1e-5, 1000)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		phi = nil
	default:
		// No rotation
		rotatedLoadings = mat.DenseCopyOf(loadings)
		nf, _ := loadings.Dims()
		rotMat = mat.NewDense(nf, nf, nil)
		for i := 0; i < nf; i++ {
			rotMat.Set(i, i, 1.0)
		}
		phi = nil
	}

	result := make(map[string]interface{})
	result["loadings"] = rotatedLoadings
	result["rotmat"] = rotMat
	if phi != nil {
		result["Phi"] = phi
	}
	result["complexity"] = mat.NewVecDense(0, nil)   // placeholder
	result["uniquenesses"] = mat.NewVecDense(0, nil) // placeholder

	return result
}
