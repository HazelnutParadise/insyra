// fa/psych_faRotations.go
package fa

import (
	"gonum.org/v1/gonum/mat"
)

// Varimax performs varimax rotation.
// Mirrors GPArotation::Varimax
func Varimax(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]interface{} {
	_, nf := loadings.Dims() // loadings is p x nf (variables x factors)
	if nf <= 1 {
		// No rotation needed for single factor
		return map[string]interface{}{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   mat.NewDense(nf, nf, nil), // identity matrix
		}
	}

	// Initialize rotation matrix as identity
	Tmat := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		Tmat.Set(i, i, 1.0)
	}

	// Use GPForth for proper varimax rotation
	result := GPForth(loadings, Tmat, normalize, eps, maxIter, "varimax")

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   result["Th"],
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
	// Remove zero-length vectors that cause issues
	// result["complexity"] = mat.NewVecDense(0, nil)   // placeholder
	// result["uniquenesses"] = mat.NewVecDense(0, nil) // placeholder

	return result
}
