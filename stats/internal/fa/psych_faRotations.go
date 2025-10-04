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

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMat := mat.NewDense(nf, nf, nil)
	rotMat.Inverse(Th)
	// Transpose the inverse matrix
	rotMatT := rotMat.T()
	rotMatDense := mat.DenseCopyOf(rotMatT)

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
	}
}

// Quartimax performs quartimax rotation.
// Mirrors GPArotation::quartimax
func Quartimax(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]interface{} {
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

	// Use GPForth for proper quartimax rotation
	result := GPForth(loadings, Tmat, normalize, eps, maxIter, "quartimax")

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMat := mat.NewDense(nf, nf, nil)
	rotMat.Inverse(Th)
	// Transpose the inverse matrix
	rotMatT := rotMat.T()
	rotMatDense := mat.DenseCopyOf(rotMatT)

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
	}
}

// Quartimin performs quartimin rotation.
// Mirrors GPArotation::quartimin
func Quartimin(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]interface{} {
	_, nf := loadings.Dims() // loadings is p x nf (variables x factors)
	if nf <= 1 {
		// No rotation needed for single factor
		return map[string]interface{}{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   mat.NewDense(nf, nf, nil), // identity matrix
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		Tmat.Set(i, i, 1.0)
	}

	// Use GPFoblq for proper quartimin rotation
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "quartimin")

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMat := mat.NewDense(nf, nf, nil)
	rotMat.Inverse(Th)
	rotMatT := rotMat.T()
	rotMatDense := mat.DenseCopyOf(rotMatT)

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
	}
}

// Oblimin performs oblimin rotation.
// Mirrors GPArotation::oblimin
func Oblimin(loadings *mat.Dense, normalize bool, eps float64, maxIter int, gamma float64) map[string]interface{} {
	_, nf := loadings.Dims() // loadings is p x nf (variables x factors)
	if nf <= 1 {
		// No rotation needed for single factor
		return map[string]interface{}{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   mat.NewDense(nf, nf, nil), // identity matrix
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		Tmat.Set(i, i, 1.0)
	}

	// Use GPFoblq for proper oblimin rotation
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "oblimin")

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMat := mat.NewDense(nf, nf, nil)
	rotMat.Inverse(Th)
	rotMatT := rotMat.T()
	rotMatDense := mat.DenseCopyOf(rotMatT)

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
	}
}

// GeominT performs geomin rotation.
// Mirrors GPArotation::geominT
func GeominT(loadings *mat.Dense, normalize bool, eps float64, maxIter int, delta float64) map[string]interface{} {
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

	// Use GPForth for proper geominT rotation
	result := GPForth(loadings, Tmat, normalize, eps, maxIter, "geominT")

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMat := mat.NewDense(nf, nf, nil)
	rotMat.Inverse(Th)
	// Transpose the inverse matrix
	rotMatT := rotMat.T()
	rotMatDense := mat.DenseCopyOf(rotMatT)

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
	}
}

// BentlerT performs Bentler's criterion rotation.
// Mirrors GPArotation::bentlerT
func BentlerT(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]interface{} {
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

	// Use GPForth for proper bentlerT rotation
	result := GPForth(loadings, Tmat, normalize, eps, maxIter, "bentlerT")

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMat := mat.NewDense(nf, nf, nil)
	rotMat.Inverse(Th)
	// Transpose the inverse matrix
	rotMatT := rotMat.T()
	rotMatDense := mat.DenseCopyOf(rotMatT)

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
	}
}

// Simplimax performs simplimax rotation.
// Mirrors GPArotation::simplimax
func Simplimax(loadings *mat.Dense, normalize bool, eps float64, maxIter int, k int) map[string]interface{} {
	_, nf := loadings.Dims() // loadings is p x nf (variables x factors)
	if nf <= 1 {
		// No rotation needed for single factor
		return map[string]interface{}{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   mat.NewDense(nf, nf, nil), // identity matrix
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		Tmat.Set(i, i, 1.0)
	}

	// Use GPFoblq for proper simplimax rotation
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "simplimax")

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMat := mat.NewDense(nf, nf, nil)
	rotMat.Inverse(Th)
	rotMatT := rotMat.T()
	rotMatDense := mat.DenseCopyOf(rotMatT)

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
	}
}

// GeominQ performs geomin rotation (oblique).
// Mirrors GPArotation::geominQ
func GeominQ(loadings *mat.Dense, normalize bool, eps float64, maxIter int, delta float64) map[string]interface{} {
	_, nf := loadings.Dims() // loadings is p x nf (variables x factors)
	if nf <= 1 {
		// No rotation needed for single factor
		return map[string]interface{}{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   mat.NewDense(nf, nf, nil), // identity matrix
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		Tmat.Set(i, i, 1.0)
	}

	// Use GPFoblq for proper geominQ rotation
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "geominQ")

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMat := mat.NewDense(nf, nf, nil)
	rotMat.Inverse(Th)
	rotMatT := rotMat.T()
	rotMatDense := mat.DenseCopyOf(rotMatT)

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
	}
}

// BentlerQ performs Bentler's criterion rotation (oblique).

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
	case "quartimax":
		result := Quartimax(loadings, true, 1e-5, 1000)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		phi = nil
	case "quartimin":
		result := Quartimin(loadings, true, 1e-5, 1000)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		if result["phi"] != nil {
			phi = result["phi"].(*mat.Dense)
		} else {
			phi = nil
		}
	case "oblimin":
		result := Oblimin(loadings, true, 1e-5, 1000, hyper)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		if result["phi"] != nil {
			phi = result["phi"].(*mat.Dense)
		} else {
			phi = nil
		}
	case "geominT":
		result := GeominT(loadings, true, 1e-5, 1000, 0.01)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		phi = nil
	case "bentlerT":
		result := BentlerT(loadings, true, 1e-5, 1000)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		phi = nil
	case "simplimax":
		result := Simplimax(loadings, true, 1e-5, 1000, loadings.RawMatrix().Rows)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		if result["phi"] != nil {
			phi = result["phi"].(*mat.Dense)
		} else {
			phi = nil
		}
	case "geominQ":
		result := GeominQ(loadings, true, 1e-5, 1000, 0.01)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		if result["phi"] != nil {
			phi = result["phi"].(*mat.Dense)
		} else {
			phi = nil
		}
	case "bentlerQ":
		result := BentlerQ(loadings, true, 1e-5, 1000)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		if result["phi"] != nil {
			phi = result["phi"].(*mat.Dense)
		} else {
			phi = nil
		}
	case "promax", "Promax":
		result := Promax(loadings, 4, true)
		rotatedLoadings = result["loadings"].(*mat.Dense)
		rotMat = result["rotmat"].(*mat.Dense)
		if result["Phi"] != nil {
			phi = result["Phi"].(*mat.Dense)
		} else {
			phi = nil
		}
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

// BentlerQ performs Bentler's criterion rotation (oblique).
// Mirrors GPArotation::bentlerQ
func BentlerQ(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]interface{} {
	_, nf := loadings.Dims() // loadings is p x nf (variables x factors)
	if nf <= 1 {
		// No rotation needed for single factor
		return map[string]interface{}{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   mat.NewDense(nf, nf, nil), // identity matrix
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := mat.NewDense(nf, nf, nil)
	for i := 0; i < nf; i++ {
		Tmat.Set(i, i, 1.0)
	}

	// Use GPFoblq for proper bentlerQ rotation
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "bentlerQ")

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMat := mat.NewDense(nf, nf, nil)
	rotMat.Inverse(Th)
	rotMatT := rotMat.T()
	rotMatDense := mat.DenseCopyOf(rotMatT)

	// Return with correct key names expected by FaRotations
	return map[string]interface{}{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
	}
}
