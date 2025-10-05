// fa/psych_faRotations.go
package fa

import (
	"math"
	"math/rand"
	"strings"

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
		"f":        result["f"],
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
		"f":        result["f"],
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
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "quartimin", 0.0)

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
		"f":        result["f"],
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
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "oblimin", gamma)

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
		"f":        result["f"],
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
		"f":        result["f"],
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
		"f":        result["f"],
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
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "simplimax", 0.0)

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
		"f":        result["f"],
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
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "geominQ", 0.0)

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
		"f":        result["f"],
	}
}

// BentlerQ performs Bentler's criterion rotation (oblique).

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
	result := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "bentlerQ", 0.0)

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
		"f":        result["f"],
	}
}

// FaRotations performs rotation selection with optional random restarts.
func FaRotations(loadings *mat.Dense, r *mat.Dense, rotate string, hyper float64, nRotations int) interface{} {
	_, nf := loadings.Dims()
	if nf == 0 {
		return map[string]interface{}{}
	}

	rotateLower := strings.ToLower(rotate)
	supportsRestarts := map[string]bool{
		"varimax":   true,
		"quartimax": true,
		"quartimin": true,
		"oblimin":   true,
		"geomint":   true,
		"geominq":   true,
		"bentlert":  true,
		"bentlerq":  true,
		"simplimax": true,
	}

	restarts := nRotations
	if restarts <= 0 {
		restarts = 1
	}
	if !supportsRestarts[rotateLower] {
		restarts = 1
	}

    // For oblimin, let GPFoblq handle Kaiser normalization internally
    useKaiser := false
    var normalizedLoadings *mat.Dense

    bestScore := math.Inf(1)
    var best map[string]interface{}

    var baseLoadings *mat.Dense
    if useKaiser {
        baseLoadings = normalizedLoadings
    } else {
        baseLoadings = loadings
    }

    // Build starting rotation matrices: identity, Varimax start, then random orthonormal
    starts := make([]*mat.Dense, 0, restarts)
    starts = append(starts, identityMatrix(nf))
    if nf > 1 {
        // Varimax start
        vm := Varimax(baseLoadings, true, 1e-05, 1000)
        if rot, ok := vm["rotmat"].(*mat.Dense); ok && rot != nil {
            starts = append(starts, mat.DenseCopyOf(rot))
        }
        // Promax start
        pm := Promax(baseLoadings, 4, true)
        if rot, ok := pm["rotmat"].(*mat.Dense); ok && rot != nil {
            starts = append(starts, mat.DenseCopyOf(rot))
        }
        // Target (cluster) start
        if loadings != nil {
            if trgLoad, trgRot, _, err := TargetRot(baseLoadings, nil); err == nil {
                _ = trgLoad
                if trgRot != nil {
                    starts = append(starts, mat.DenseCopyOf(trgRot))
                }
            }
        }
    }
    if restarts > len(starts) {
        seed := seedFromMatrix(baseLoadings)
        rnd := rand.New(rand.NewSource(seed))
        for i := len(starts); i < restarts; i++ {
            starts = append(starts, randomOrthonormalMatrix(nf, rnd))
        }
    }

	for idx, start := range starts {
		pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
		pre.Mul(baseLoadings, start)

		var result map[string]interface{}
		switch rotateLower {
		case "varimax":
			result = Varimax(pre, true, 1e-05, 1000)
		case "quartimax":
			result = Quartimax(pre, true, 1e-05, 1000)
		case "quartimin":
			result = Quartimin(pre, true, 1e-05, 1000)
        case "oblimin":
            // Let GPFoblq do Kaiser normalization (normalize=true) on original scale
            gpf := GPFoblq(pre, identityMatrix(nf), true, 1e-05, 1000, "oblimin", hyper)
            result = finalizeGpfResult(gpf, nf)
		case "geomint":
			result = GeominT(pre, true, 1e-05, 1000, 0.01)
		case "geominq":
			result = GeominQ(pre, true, 1e-05, 1000, 0.01)
		case "bentlert":
			result = BentlerT(pre, true, 1e-05, 1000)
		case "bentlerq":
			result = BentlerQ(pre, true, 1e-05, 1000)
		case "simplimax":
			result = Simplimax(pre, true, 1e-05, 1000, pre.RawMatrix().Rows)
		case "promax":
			res := Promax(pre, 4, true)
			result = map[string]interface{}{
				"loadings": res["loadings"],
				"rotmat":   res["rotmat"],
				"Phi":      res["Phi"],
			}
		default:
			result = map[string]interface{}{
				"loadings": mat.DenseCopyOf(pre),
				"rotmat":   identityMatrix(nf),
			}
		}

		rotLoad, ok := result["loadings"].(*mat.Dense)
		if !ok {
			continue
		}

        finalLoadings := mat.DenseCopyOf(rotLoad)

		var partialRot *mat.Dense
		if rm, ok := result["rotmat"].(*mat.Dense); ok && rm != nil {
			partialRot = rm
		} else {
			partialRot = identityMatrix(nf)
		}
		finalRot := mat.NewDense(start.RawMatrix().Rows, partialRot.RawMatrix().Cols, nil)
		finalRot.Mul(start, partialRot)

		candidate := map[string]interface{}{
			"loadings": finalLoadings,
			"rotmat":   finalRot,
		}
		if phiVal, ok := result["phi"].(*mat.Dense); ok && phiVal != nil {
			candidate["Phi"] = phiVal
		} else if phiVal, ok := result["Phi"].(*mat.Dense); ok && phiVal != nil {
			candidate["Phi"] = phiVal
		}

		score := math.Inf(1)
		if fVal, ok := result["f"].(float64); ok {
			score = fVal
			candidate["f"] = fVal
		} else if idx == 0 {
			score = 0
		}

		if best == nil || score < bestScore || (math.IsNaN(bestScore) && !math.IsNaN(score)) {
			best = candidate
			bestScore = score
		}
	}

	if best == nil {
		rotatedLoadings := mat.DenseCopyOf(loadings)
		rotMat := identityMatrix(nf)
		best = map[string]interface{}{
			"loadings": rotatedLoadings,
			"rotmat":   rotMat,
		}
	}

	return best
}

func identityMatrix(n int) *mat.Dense {
	matIdentity := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		matIdentity.Set(i, i, 1.0)
	}
	return matIdentity
}

func randomOrthonormalMatrix(n int, rnd *rand.Rand) *mat.Dense {
	data := make([]float64, n*n)
	for i := range data {
		data[i] = rnd.NormFloat64()
	}
	base := mat.NewDense(n, n, data)
	var qr mat.QR
	qr.Factorize(base)
	var q mat.Dense
	qr.QTo(&q)
	return mat.DenseCopyOf(&q)
}

func seedFromMatrix(m *mat.Dense) int64 {
	data := m.RawMatrix().Data
	var seed uint64 = uint64(len(data)) + 1
	for _, v := range data {
		bits := math.Float64bits(v)
		seed ^= bits + 0x9e3779b97f4a7c15 + (seed << 6) + (seed >> 2)
	}
	if seed == 0 {
		seed = 0x9e3779b97f4a7c15
	}
	return int64(seed)
}

func kaiserNormalize(loadings *mat.Dense) (*mat.Dense, []float64) {
	rows, cols := loadings.Dims()
	weights := make([]float64, rows)
	normalized := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		sum := 0.0
		for j := 0; j < cols; j++ {
			val := loadings.At(i, j)
			sum += val * val
		}
		if sum <= 0 {
			weights[i] = 1.0
		} else {
			weights[i] = 1.0 / math.Sqrt(sum)
		}
		for j := 0; j < cols; j++ {
			normalized.Set(i, j, loadings.At(i, j)*weights[i])
		}
	}
	return normalized, weights
}

func finalizeGpfResult(gpf map[string]interface{}, nf int) map[string]interface{} {
	Th, ok := gpf["Th"].(*mat.Dense)
	if !ok || Th == nil {
		return gpf
	}
	rotSolve := mat.NewDense(nf, nf, nil)
	rotSolve.Inverse(Th)
	rotMat := mat.DenseCopyOf(rotSolve.T())
	res := map[string]interface{}{
		"loadings": gpf["loadings"],
		"rotmat":   rotMat,
		"f":        gpf["f"],
	}
	if phi, ok := gpf["Phi"]; ok && phi != nil {
		res["Phi"] = phi
	}
	return res
}
