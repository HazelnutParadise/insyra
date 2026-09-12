// fa/psych_faRotations.go
package fa

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/HazelnutParadise/insyra"

	"gonum.org/v1/gonum/mat"
)

const debugOblimin = false

// Varimax performs varimax rotation.
// Mirrors GPArotation::Varimax
func Varimax(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]any {
	_, cols := loadings.Dims()
	if cols <= 1 {
		// No rotation needed for single factor
		return map[string]any{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   identityMatrix(cols),
		}
	}

	// Initialize rotation matrix as identity
	Tmat := identityMatrix(cols)

	// Use GPForth for proper varimax rotation
	result, err := GPForth(loadings, Tmat, normalize, eps, maxIter, "varimax", 0)
	if err != nil {
		return map[string]any{
			"f":     0.0,
			"error": err.Error(),
		}
	}

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMatDense := rotMatFromTh(Th, cols)
	if rotMatDense == nil {
		return map[string]any{
			"f":     result["f"],
			"error": "failed to compute varimax rotation matrix",
		}
	}

	// Return with correct key names expected by FaRotations.
	return withConvergence(map[string]any{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"f":        result["f"],
	}, result)
}

// Quartimax performs quartimax rotation.
// Mirrors GPArotation::quartimax
func Quartimax(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]any {
	_, cols := loadings.Dims()
	if cols <= 1 {
		// No rotation needed for single factor
		return map[string]any{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   identityMatrix(cols),
		}
	}

	// Initialize rotation matrix as identity
	Tmat := identityMatrix(cols)

	// Use GPForth for proper quartimax rotation
	result, err := GPForth(loadings, Tmat, normalize, eps, maxIter, "quartimax", 0)
	if err != nil {
		return map[string]any{
			"f":     0.0,
			"error": err.Error(),
		}
	}

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMatDense := rotMatFromTh(Th, cols)
	if rotMatDense == nil {
		return map[string]any{
			"f":     result["f"],
			"error": "failed to compute quartimax rotation matrix",
		}
	}

	// Return with correct key names expected by FaRotations.
	return withConvergence(map[string]any{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"f":        result["f"],
	}, result)
}

// Quartimin performs quartimin rotation.
// Mirrors GPArotation::quartimin
func Quartimin(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]any {
	_, cols := loadings.Dims()
	if cols <= 1 {
		// No rotation needed for single factor
		return map[string]any{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   identityMatrix(cols),
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := identityMatrix(cols)

	// Use GPFoblq for proper quartimin rotation
	result, err := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "quartimin", 0.0)
	if err != nil {
		return map[string]any{
			"f":     0.0,
			"error": err.Error(),
		}
	}

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMatDense := rotMatFromTh(Th, cols)
	if rotMatDense == nil {
		return map[string]any{
			"f":     result["f"],
			"error": "failed to compute quartimin rotation matrix",
		}
	}

	// Return with correct key names expected by FaRotations
	return withConvergence(map[string]any{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
		"f":        result["f"],
	}, result)
}

// Oblimin performs oblimin rotation.
// Mirrors GPArotation::oblimin
func Oblimin(loadings *mat.Dense, normalize bool, eps float64, maxIter int, gamma float64) map[string]any {
	_, cols := loadings.Dims()
	if cols <= 1 {
		// No rotation needed for single factor
		return map[string]any{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   identityMatrix(cols),
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := identityMatrix(cols)

	// Use GPFoblq for proper oblimin rotation
	result, err := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "oblimin", gamma)
	if err != nil {
		return map[string]any{
			"f":     0.0,
			"error": err.Error(),
		}
	}

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMatDense := rotMatFromTh(Th, cols)
	if rotMatDense == nil {
		return map[string]any{
			"f":     result["f"],
			"error": "failed to compute oblimin rotation matrix",
		}
	}

	// Return with correct key names expected by FaRotations
	return withConvergence(map[string]any{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
		"f":        result["f"],
	}, result)
}

// GeominT performs geomin rotation.
// Mirrors GPArotation::geominT
func GeominT(loadings *mat.Dense, normalize bool, eps float64, maxIter int, delta float64) map[string]any {
	_, cols := loadings.Dims()
	if cols <= 1 {
		// No rotation needed for single factor
		return map[string]any{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   identityMatrix(cols),
		}
	}

	// Initialize rotation matrix as identity
	Tmat := identityMatrix(cols)

	// Use GPForth for proper geominT rotation
	result, err := GPForth(loadings, Tmat, normalize, eps, maxIter, "geomin", delta)
	if err != nil {
		return map[string]any{
			"f":     0.0,
			"error": err.Error(),
		}
	}

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMatDense := rotMatFromTh(Th, cols)
	if rotMatDense == nil {
		return map[string]any{
			"f":     result["f"],
			"error": "failed to compute geominT rotation matrix",
		}
	}

	// Return with correct key names expected by FaRotations.
	return withConvergence(map[string]any{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"f":        result["f"],
	}, result)
}

// BentlerT performs Bentler's criterion rotation.
// Mirrors GPArotation::bentlerT
func BentlerT(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]any {
	_, cols := loadings.Dims()
	if cols <= 1 {
		// No rotation needed for single factor
		return map[string]any{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   identityMatrix(cols),
		}
	}

	// Initialize rotation matrix as identity
	Tmat := identityMatrix(cols)

	// Use GPForth for proper bentlerT rotation
	result, err := GPForth(loadings, Tmat, normalize, eps, maxIter, "bentler", 0)
	if err != nil {
		return map[string]any{
			"f":     0.0,
			"error": err.Error(),
		}
	}

	// Use Th (T matrix) directly for orthogonal reporting
	Th := result["Th"].(*mat.Dense)
	rotMatDense := mat.DenseCopyOf(Th)

	// Return with correct key names expected by FaRotations.
	return withConvergence(map[string]any{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"f":        result["f"],
	}, result)
}

// Simplimax performs simplimax rotation.
// Mirrors GPArotation::simplimax
func Simplimax(loadings *mat.Dense, normalize bool, eps float64, maxIter int, k int) map[string]any {
	_, cols := loadings.Dims()
	if cols <= 1 {
		// No rotation needed for single factor
		return map[string]any{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   identityMatrix(cols),
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := identityMatrix(cols)

	// Use GPFoblq for proper simplimax rotation
	result, err := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "simplimax", 0.0)
	if err != nil {
		return map[string]any{
			"f":     0.0,
			"error": err.Error(),
		}
	}

	// Calculate rotation matrix as t(solve(Th)) to match other oblique handlers
	Th := result["Th"].(*mat.Dense)
	rotMatDense := rotMatFromTh(Th, cols)
	if rotMatDense == nil {
		return map[string]any{
			"f":     result["f"],
			"error": "failed to compute simplimax rotation matrix",
		}
	}

	// Return with correct key names expected by FaRotations
	return withConvergence(map[string]any{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
		"f":        result["f"],
	}, result)
}

// GeominQ performs geomin rotation (oblique).
// Mirrors GPArotation::geominQ
func GeominQ(loadings *mat.Dense, normalize bool, eps float64, maxIter int, delta float64) map[string]any {
	_, cols := loadings.Dims()
	if cols <= 1 {
		// No rotation needed for single factor
		return map[string]any{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   identityMatrix(cols),
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := identityMatrix(cols)

	// Use GPFoblq for proper geominQ rotation. The 7th param (named gamma in
	// GPFoblq) is repurposed as ε for the geomin criterion.
	result, err := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "geominQ", delta)
	if err != nil {
		return map[string]any{
			"f":     0.0,
			"error": err.Error(),
		}
	}

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMatDense := rotMatFromTh(Th, cols)
	if rotMatDense == nil {
		return map[string]any{
			"f":     result["f"],
			"error": "failed to compute geominQ rotation matrix",
		}
	}

	// Return with correct key names expected by FaRotations
	return withConvergence(map[string]any{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
		"f":        result["f"],
	}, result)
}

// BentlerQ performs Bentler's criterion rotation (oblique).
// Mirrors GPArotation::bentlerQ
func BentlerQ(loadings *mat.Dense, normalize bool, eps float64, maxIter int) map[string]any {
	_, cols := loadings.Dims()
	if cols <= 1 {
		// No rotation needed for single factor
		return map[string]any{
			"loadings": mat.DenseCopyOf(loadings),
			"rotmat":   identityMatrix(cols),
			"phi":      nil,
		}
	}

	// Initialize rotation matrix as identity
	Tmat := identityMatrix(cols)

	// Use GPFoblq for proper bentlerQ rotation
	result, err := GPFoblq(loadings, Tmat, normalize, eps, maxIter, "bentlerQ", 0.0)
	if err != nil {
		return map[string]any{
			"f":     0.0,
			"error": err.Error(),
		}
	}

	// Calculate rotation matrix as t(solve(Th)) like in R
	Th := result["Th"].(*mat.Dense)
	rotMatDense := rotMatFromTh(Th, cols)
	if rotMatDense == nil {
		return map[string]any{
			"f":     result["f"],
			"error": "failed to compute bentlerQ rotation matrix",
		}
	}

	// Return with correct key names expected by FaRotations
	return withConvergence(map[string]any{
		"loadings": result["loadings"],
		"rotmat":   rotMatDense,
		"phi":      result["Phi"],
		"f":        result["f"],
	}, result)
}

// FaRotations performs rotation selection with optional random restarts.
// FaRotations applies a factor rotation. promaxPower (R: m, default 4) is
// honored for the "promax" rotation; geominDelta (R: delta, default 0.01)
// is honored for "geomint" / "geominq". Pass <= 0 to use defaults.
func FaRotations(loadings *mat.Dense, r *mat.Dense, rotate string, hyper float64, nRotations int, promaxPower int, geominDelta float64, eps float64, maxIter int) any {
	if promaxPower <= 0 {
		promaxPower = 4
	}
	if geominDelta <= 0 {
		geominDelta = 0.01
	}
	if eps <= 0 {
		eps = 1e-05
	}
	if maxIter <= 0 {
		maxIter = 1000
	}
	_, nf := loadings.Dims()
	if nf == 0 {
		return map[string]any{}
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
	bestConverged := false
	var best map[string]any

	var baseLoadings *mat.Dense
	if useKaiser {
		baseLoadings = normalizedLoadings
	} else {
		baseLoadings = loadings
	}

	starts := buildStarts(baseLoadings, nf, restarts, eps, maxIter)

	for idx, start := range starts {

		var result map[string]any
		switch rotateLower {
		case "varimax":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			// R defaults: eps=1e-05, maxit=1000 (now overridable via opts)
			result = Varimax(pre, true, eps, maxIter)
		case "quartimax":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = Quartimax(pre, false, eps, maxIter)
		case "quartimin":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = Quartimin(pre, false, eps, maxIter)
		case "oblimin":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			gpf, err := GPFoblq(pre, identityMatrix(nf), false, eps, maxIter, "oblimin", hyper)
			if err != nil {
				continue
			}
			result = finalizeGpfResult(gpf, nf)
		case "geomint":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = GeominT(pre, false, eps, maxIter, geominDelta)
		case "geominq":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = GeominQ(pre, false, eps, maxIter, geominDelta)
		case "bentlert":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = BentlerT(pre, false, eps, maxIter)
		case "bentlerq":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = BentlerQ(pre, false, eps, maxIter)
		case "simplimax":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = Simplimax(pre, false, eps, maxIter, pre.RawMatrix().Rows)
		case "promax":
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			h2 := make([]float64, pre.RawMatrix().Rows)
			weighted := mat.DenseCopyOf(pre)
			for i := 0; i < pre.RawMatrix().Rows; i++ {
				sum := 0.0
				for j := 0; j < pre.RawMatrix().Cols; j++ {
					v := pre.At(i, j)
					sum += v * v
				}
				h2[i] = math.Sqrt(sum)
				if h2[i] != 0 {
					for j := 0; j < pre.RawMatrix().Cols; j++ {
						weighted.Set(i, j, pre.At(i, j)/h2[i])
					}
				}
			}
			res := Promax(weighted, promaxPower, false)
			if errMsg, ok := res["error"].(string); ok && errMsg != "" {
				result = map[string]any{"error": errMsg}
				break
			}
			if lm, ok := res["loadings"].(*mat.Dense); ok && lm != nil {
				normalized := mat.DenseCopyOf(lm)
				for i := 0; i < normalized.RawMatrix().Rows; i++ {
					for j := 0; j < normalized.RawMatrix().Cols; j++ {
						normalized.Set(i, j, normalized.At(i, j)*h2[i])
					}
				}
				res["loadings"] = normalized
			}
			result = map[string]any{
				"loadings": res["loadings"],
				"rotmat":   res["rotmat"],
				"Phi":      res["Phi"],
			}
		default:
			pre := mat.NewDense(baseLoadings.RawMatrix().Rows, baseLoadings.RawMatrix().Cols, nil)
			pre.Mul(baseLoadings, start)
			result = map[string]any{
				"error": fmt.Sprintf("unsupported rotation method: %s", rotate),
			}
		}

		if errMsg, ok := result["error"].(string); ok && errMsg != "" {
			continue
		}
		rotLoad, ok := result["loadings"].(*mat.Dense)
		if !ok {
			continue
		}

		finalLoadings := mat.DenseCopyOf(rotLoad)

		var finalRot *mat.Dense
		if rm, ok := result["rotmat"].(*mat.Dense); ok && rm != nil {
			finalRot = mat.NewDense(start.RawMatrix().Rows, rm.RawMatrix().Cols, nil)
			finalRot.Mul(start, rm)
		} else {
			continue
		}

		candidate := map[string]any{
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

		converged := true
		if conv, ok := result["convergence"].(bool); ok {
			converged = conv
		}
		candidate["convergence"] = converged

		if preferCandidate(best != nil, bestConverged, bestScore, converged, score) {
			best = candidate
			bestScore = score
			bestConverged = converged
		}
		if debugOblimin && rotateLower == "oblimin" {
			fmt.Printf("oblimin start %d score=%.9f\n", idx, score)
		}
	}

	if best == nil {
		best = map[string]any{
			"error": fmt.Sprintf("rotation %s failed for all starts", rotate),
		}
	} else if !bestConverged {
		// One warning for the search, not one per start: the per-start
		// runs report at debug level, and the caller reads the outcome
		// from the convergence flag this candidate carries.
		insyra.LogWarning("fa", "FaRotations",
			"%s rotation did not converge from any of %d starts within %d iterations; returning the best of them",
			rotateLower, len(starts), maxIter)
	}
	if debugOblimin && rotateLower == "oblimin" {
		fmt.Printf("oblimin best score=%.9f\n", bestScore)
	}

	return best
}

// startOrthogonalityTol bounds how far a starting matrix may sit from the
// orthogonal group. It is loose enough for a matrix assembled by QR or handed
// back by a rotation that converged, and far tighter than the departures #373
// produced: the Promax start was off by 0.42.
const startOrthogonalityTol = 1e-8

// buildStarts returns the starting rotation matrices for a multi-start search.
//
// Every start has to lie on the criterion's own manifold. The gradient
// projection algorithms project each step relative to where they began, so a
// start off the manifold yields a matrix that is not a rotation at all, and
// loadings that no longer describe the fitted model (#373). An orthogonal
// matrix satisfies both families' constraints, T'T = I for the orthogonal
// criteria and diag(T'T) = I for the oblique ones, which is why
// GPArotation::Random.Start hands the same QR-orthogonalised matrices to both.
//
// The list is the identity, Varimax's rotation matrix as one informed start,
// and random orthogonal matrices for the rest. len(starts) == restarts, so the
// parameter means what it says. eps and maxIter are the rotation's own, and
// bound the informed start.
func buildStarts(baseLoadings *mat.Dense, nf, restarts int, eps float64, maxIter int) []*mat.Dense {
	starts := make([]*mat.Dense, 0, max(1, restarts))
	starts = append(starts, identityMatrix(nf))
	if restarts <= 1 || nf <= 1 {
		return starts
	}

	// Varimax's rotation matrix is orthogonal by construction, but it is
	// checked like any other start: a rotation that ran out of iterations can
	// hand back a matrix that has drifted off the manifold. It runs at the
	// search's own tolerance: a start only has to be on the manifold, and at
	// 1e-8 / 5000 it hit the cap and cost 80 ms on tables where the whole
	// twenty-start search costs 2.
	vm := Varimax(baseLoadings, true, eps, maxIter)
	if rot, ok := vm["rotmat"].(*mat.Dense); ok && isOrthonormal(rot) {
		starts = append(starts, mat.DenseCopyOf(rot))
	}

	// A start that fails the check is skipped rather than used, and another is
	// drawn in its place, so the count does not silently shrink.
	rnd := rand.New(rand.NewSource(seedFromMatrix(baseLoadings)))
	for attempts := 0; len(starts) < restarts && attempts < 4*restarts; attempts++ {
		if q := randomOrthonormalMatrix(nf, rnd); isOrthonormal(q) {
			starts = append(starts, q)
		}
	}
	return starts
}

// isOrthonormal reports whether m'm is the identity within
// startOrthogonalityTol.
func isOrthonormal(m *mat.Dense) bool {
	if m == nil {
		return false
	}
	rows, cols := m.Dims()
	if rows != cols {
		return false
	}
	for i := 0; i < cols; i++ {
		for j := i; j < cols; j++ {
			dot := 0.0
			for k := 0; k < rows; k++ {
				dot += m.At(k, i) * m.At(k, j)
			}
			want := 0.0
			if i == j {
				want = 1.0
			}
			if math.Abs(dot-want) > startOrthogonalityTol {
				return false
			}
		}
	}
	return true
}

// preferCandidate decides whether a newly finished start beats the best so far.
//
// A converged solution always beats one that did not converge: comparing
// criterion values across starts only means something among runs that actually
// finished. When nothing converged the best of them is still returned, and
// reported as not converged, because an approximate rotation is more use to a
// caller than none.
func preferCandidate(haveBest, bestConverged bool, bestScore float64, converged bool, score float64) bool {
	if !haveBest {
		return true
	}
	if converged != bestConverged {
		return converged
	}
	if math.IsNaN(bestScore) {
		return !math.IsNaN(score)
	}
	return score < bestScore
}

// withConvergence copies the convergence flag out of a GPForth/GPFoblq result
// so the caller can report whether the rotation it chose actually converged.
func withConvergence(out, from map[string]any) map[string]any {
	if conv, ok := from["convergence"]; ok {
		out["convergence"] = conv
	}
	return out
}

func identityMatrix(n int) *mat.Dense {
	out := mat.NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		out.Set(i, i, 1)
	}
	return out
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
	var seed = uint64(len(data)) + 1
	for _, v := range data {
		bits := math.Float64bits(v)
		seed ^= bits + 0x9e3779b97f4a7c15 + (seed << 6) + (seed >> 2)
	}
	if seed == 0 {
		seed = 0x9e3779b97f4a7c15
	}
	return int64(seed)
}

func finalizeGpfResult(gpf map[string]any, nf int) map[string]any {
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
	rotMat := rotMatFromTh(Th, nf)
	if rotMat == nil {
		return map[string]any{"error": "failed to compute rotation matrix from transformation matrix"}
	}
	res := map[string]any{
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

// rotMatFromTh computes the rotation matrix from Th.
// In GPFoblq, Phi = Th^T * Th
// SPSS expects rotmat * rotmat^T = Phi
// Therefore rotmat * rotmat^T = Th^T * Th
// This means rotmat^T = Th, so rotmat = Th^T
func rotMatFromTh(Th *mat.Dense, nf int) *mat.Dense {
	if Th == nil {
		return nil
	}
	// R code: rot.mat <- t(solve(Th))
	// For orthogonal rotations (like Varimax), Th is orthogonal, so:
	// solve(Th) = t(Th), thus t(solve(Th)) = t(t(Th)) = Th
	// For non-orthogonal rotations, we need the full inverse

	// Check if Th is orthogonal
	var ThTTh mat.Dense
	ThTTh.Mul(Th.T(), Th)
	isOrthogonal := true
	for i := 0; i < nf && isOrthogonal; i++ {
		for j := 0; j < nf && isOrthogonal; j++ {
			expected := 0.0
			if i == j {
				expected = 1.0
			}
			if math.Abs(ThTTh.At(i, j)-expected) > 1e-6 {
				isOrthogonal = false
			}
		}
	}

	if isOrthogonal {
		// For orthogonal matrices: t(solve(Th)) = Th
		return mat.DenseCopyOf(Th)
	}

	// For non-orthogonal: compute inverse and transpose. invertDense
	// recovers from gonum's panic on truly-singular input.
	invTh, err := invertDense(Th)
	if err != nil {
		return nil
	}
	return mat.DenseCopyOf(invTh.T())
}
