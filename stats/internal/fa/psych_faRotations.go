// fa/psych_faRotations.go
package fa

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"gonum.org/v1/gonum/mat"
)

const debugOblimin = false

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
	// Transpose inverse
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

	// Use Th (T matrix) directly for oblique reporting
	Th := result["Th"].(*mat.Dense)
	rotMatDense := mat.DenseCopyOf(Th)

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

	// Calculate rotation matrix as t(solve(Th)) to match other oblique handlers
	Th := result["Th"].(*mat.Dense)
	rotSolve := mat.NewDense(nf, nf, nil)
	rotSolve.Inverse(Th)
	rotMatT := rotSolve.T()
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

	// Build starting rotation matrices
	// To emulate SPSS Direct Oblimin behavior deterministically, when
	// restarts <= 1, use only identity start. For larger restarts, include
	// additional heuristics (Varimax/Promax/Target) and random starts up to
	// the requested count.
	starts := make([]*mat.Dense, 0, max(1, restarts))
	starts = append(starts, identityMatrix(nf))
	if restarts > 1 && nf > 1 {
		// Heuristic starts
		vm := Varimax(baseLoadings, true, 1e-08, 5000)
		if rot, ok := vm["rotmat"].(*mat.Dense); ok && rot != nil {
			starts = append(starts, mat.DenseCopyOf(rot))
		}
		pm := Promax(baseLoadings, 4, true)
		if rot, ok := pm["rotmat"].(*mat.Dense); ok && rot != nil {
			starts = append(starts, mat.DenseCopyOf(rot))
		}
		if loadings != nil {
			if _, trgRot, _, err := TargetRot(baseLoadings, nil); err == nil {
				if trgRot != nil {
					starts = append(starts, mat.DenseCopyOf(trgRot))
				}
			}
		}
		// Add random orthonormal starts if budget remains
		if restarts > len(starts) {
			seed := seedFromMatrix(baseLoadings)
			rnd := rand.New(rand.NewSource(seed))
			for i := len(starts); i < restarts; i++ {
				starts = append(starts, randomOrthonormalMatrix(nf, rnd))
			}
		}
	}

	for idx, start := range starts {

		var result map[string]interface{}
		switch rotateLower {
		case "varimax":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = Varimax(pre, true, 1e-08, 5000)
		case "quartimax":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = Quartimax(pre, true, 1e-08, 5000)
		case "quartimin":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = Quartimin(pre, true, 1e-08, 5000)
		case "oblimin":
			// Use identity matrix as starting point, like R's psych package
			startIdentity := mat.NewDense(nf, nf, nil)
			for i := 0; i < nf; i++ {
				startIdentity.Set(i, i, 1.0)
			}
			var gpf map[string]interface{}
			var ok bool
			func() {
				defer func() {
					if r := recover(); r != nil {
						ok = false
					}
				}()
				gpf = GPFoblq(baseLoadings, startIdentity, true, 1e-08, 1000, "oblimin", hyper)
				ok = true
			}()
			if !ok {
				continue
			}
			result = finalizeGpfResult(gpf, nf)
		case "geomint":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = GeominT(pre, true, 1e-08, 5000, 0.01)
		case "geominq":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = GeominQ(pre, true, 1e-08, 5000, 0.01)
		case "bentlert":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = BentlerT(pre, true, 1e-08, 5000)
		case "bentlerq":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = BentlerQ(pre, true, 1e-08, 5000)
		case "simplimax":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = Simplimax(pre, true, 1e-08, 5000, pre.RawMatrix().Rows)
		case "promax":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			res := Promax(pre, 4, true)
			result = map[string]interface{}{
				"loadings": res["loadings"],
				"rotmat":   res["rotmat"],
				"Phi":      res["Phi"],
			}
		default:
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
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

		var finalRot *mat.Dense
		if rm, ok := result["rotmat"].(*mat.Dense); ok && rm != nil {
			if rotateLower == "oblimin" {
				finalRot = mat.DenseCopyOf(rm)
			} else {
				finalRot = mat.NewDense(start.RawMatrix().Rows, rm.RawMatrix().Cols, nil)
				finalRot.Mul(start, rm)
			}
		} else {
			if rotateLower == "oblimin" {
				finalRot = identityMatrix(nf)
			} else {
				finalRot = mat.DenseCopyOf(start)
			}
		}

		candidate := map[string]interface{}{
			"loadings": finalLoadings,
			"rotmat":   finalRot,
		}
		if phiVal, ok := result["phi"].(*mat.Dense); ok && phiVal != nil {
			candidate["Phi"] = phiVal
		} else if phiVal, ok := result["Phi"].(*mat.Dense); ok && phiVal != nil {
			candidate["Phi"] = phiVal
		}
		if debugOblimin && rotateLower == "oblimin" {
			fmt.Printf("oblimin start %d loadings:\n", idx)
			for i := 0; i < finalLoadings.RawMatrix().Rows; i++ {
				for j := 0; j < finalLoadings.RawMatrix().Cols; j++ {
					fmt.Printf(" % .6f", finalLoadings.At(i, j))
				}
				fmt.Printf("\n")
			}
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
		if debugOblimin && rotateLower == "oblimin" {
			fmt.Printf("oblimin start %d score=%.9f\n", idx, score)
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
	if debugOblimin && rotateLower == "oblimin" {
		fmt.Printf("oblimin best score=%.9f\n", bestScore)
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
	if debugOblimin {
		if rawPhi, ok := gpf["Phi"].(*mat.Dense); ok && rawPhi != nil {
			fmt.Printf("GPFoblq raw Phi:\n")
			for i := 0; i < rawPhi.RawMatrix().Rows; i++ {
				for j := 0; j < rawPhi.RawMatrix().Cols; j++ {
					fmt.Printf(" % .6f", rawPhi.At(i, j))
				}
				fmt.Printf("\n")
			}
		}
	}
	// rotmat = t(solve(Th)) to be consistent with composition rules
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
	if conv, ok := gpf["convergence"]; ok {
		res["convergence"] = conv
	}
	return res
}

// ParseRotationResult accepts the opaque result value returned by
// FaRotations (or individual rotation functions) and returns typed
// pointers to the rotated loadings, rotation matrix (rotmat), Phi
// (may be nil for orthogonal rotations), the objective f and a bool
// indicating success. This helper centralizes key extraction so callers
// can programmatically compare matrices (e.g. against SPSS reference).
func ParseRotationResult(res interface{}) (loadings, rotmat, phi *mat.Dense, f float64, ok bool) {
	ok = false
	if res == nil {
		return
	}
	// Many rotation functions return map[string]interface{}
	var m map[string]interface{}
	switch v := res.(type) {
	case map[string]interface{}:
		m = v
	default:
		return
	}

	if lv, found := m["loadings"]; found {
		if ld, cast := lv.(*mat.Dense); cast {
			loadings = mat.DenseCopyOf(ld)
		}
	}
	if rv, found := m["rotmat"]; found {
		if rd, cast := rv.(*mat.Dense); cast {
			rotmat = mat.DenseCopyOf(rd)
		}
	}
	// Accept either "Phi" or "phi" keys used in code
	if pv, found := m["Phi"]; found {
		if pd, cast := pv.(*mat.Dense); cast {
			phi = mat.DenseCopyOf(pd)
		}
	} else if pv, found := m["phi"]; found {
		if pd, cast := pv.(*mat.Dense); cast {
			phi = mat.DenseCopyOf(pd)
		}
	}
	if fv, found := m["f"]; found {
		if ff, cast := fv.(float64); cast {
			f = ff
		}
	}

	// success if at least loadings and rotmat present
	if loadings != nil && rotmat != nil {
		ok = true
	}
	return
}
