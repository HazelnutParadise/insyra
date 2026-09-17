// fa/GPArotation_GPFoblq.go
package fa

import (
	"fmt"
	"math"
	"strings"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/mat"
)

// NormalizingWeight computes normalizing weights for GPA rotation.
// Mirrors GPArotation::NormalizingWeight for Kaiser normalization.
func NormalizingWeight(A *mat.Dense, normalize bool) *mat.VecDense {
	p, q := A.Dims()
	W := mat.NewVecDense(p, nil)

	if normalize {
		for i := range p {
			sum := 0.0
			for j := 0; j < q; j++ {
				val := A.At(i, j)
				sum += val * val
			}
			W.SetVec(i, math.Sqrt(sum))
		}
	} else {
		for i := 0; i < p; i++ {
			W.SetVec(i, 1.0)
		}
	}

	return W
}

// GPFoblq performs oblique GPA rotation.
// Transliteration of GPArotation::GPFoblq from R.
func GPFoblq(A *mat.Dense, Tmat *mat.Dense, normalize bool, eps float64, maxit int, method string, gamma float64) (map[string]any, error) {
	rows, cols := A.Dims()
	if cols <= 1 {
		return nil, fmt.Errorf("rotation does not make sense for single factor models")
	}

	// Work on a copy so the original loadings stay untouched.
	Aw := mat.DenseCopyOf(A)
	var weights *mat.VecDense
	if normalize {
		weights = NormalizingWeight(A, true)
		for i := 0; i < rows; i++ {
			w := weights.AtVec(i)
			if w == 0 {
				continue
			}
			for j := 0; j < cols; j++ {
				Aw.Set(i, j, Aw.At(i, j)/w)
			}
		}
	}

	// Re-normalize columns of T so each has unit norm. Oblique GPA expects
	// columns of T on the unit sphere; an arbitrary start (e.g. an oblique
	// matrix from Promax) violates this and corrupts the first iteration.
	T := mat.DenseCopyOf(Tmat)
	for j := 0; j < cols; j++ {
		s := 0.0
		for i := 0; i < cols; i++ {
			v := T.At(i, j)
			s += v * v
		}
		s = math.Sqrt(s)
		if s > 0 && s != 1 {
			for i := 0; i < cols; i++ {
				T.Set(i, j, T.At(i, j)/s)
			}
		}
	}

	// computeL returns L = A·(T')⁻¹ together with the inverse, which the
	// gradient needs as well.
	computeL := func(Tcur *mat.Dense) (*mat.Dense, *mat.Dense) {
		invT := safeInverse(Tcur)
		L := mat.NewDense(rows, cols, nil)
		L.Mul(Aw, invT.T())
		return L, invT
	}

	L, invT := computeL(T)
	Gq, f, methodName, err := obliqueCriterion(method, L, gamma)
	if err != nil {
		return nil, err
	}
	G := computeGMatrix(L, Gq, invT)

	table := make([][]float64, 0, max(1, maxit+1))
	convergence := false

	// The step follows GPArotation 2026.8.2's default, algorithm = "bb" with
	// fwindow = 10: see nextStepSize and windowMax. A trial is accepted when it
	// improves enough on the largest criterion value of the recent iterations,
	// not only on the current one, so the search does not have to crawl along
	// a nearly flat criterion. The old step, which GPArotation keeps as
	// GPFoblq.legacy, stopped at its cap from most starts on such loadings.
	alpha := 1.0
	var prevT, prevGp *mat.Dense
	iter := 0
	for ; iter <= maxit; iter++ {
		Gp := computeGp(G, T)
		s := frobNorm(Gp)
		table = append(table, []float64{float64(iter), f, math.Log10(s), alpha})
		if s < eps {
			convergence = true
			break
		}
		alpha = nextStepSize(alpha, T, prevT, Gp, prevGp)
		target := windowMax(table)

		// Up to eleven trials, halving the step after each one that does not
		// improve enough. R moves to the last trial even when none did.
		var Tt, Lt, invTt, Gqt *mat.Dense
		var ft float64
		for i := 0; i <= 10; i++ {
			X := mat.DenseCopyOf(T)
			var scaledGp mat.Dense
			scaledGp.Scale(alpha, Gp)
			X.Sub(X, &scaledGp)
			Tt = normalizeColumns(X)
			Lt, invTt = computeL(Tt)
			Gqt, ft, _, err = obliqueCriterion(method, Lt, gamma)
			if err != nil {
				return nil, err
			}
			if target-ft > 0.5*s*s*alpha {
				break
			}
			alpha /= 2
		}

		prevT, prevGp = T, Gp
		T, L, Gq, f = Tt, Lt, Gqt, ft
		G = computeGMatrix(L, Gq, invTt)
	}

	// A run that hit the cap is reported here at debug level only. Under a
	// multi-start search this is one start of many, and the caller decides
	// whether the chosen solution converged: FaRotations warns once, and
	// only then. R's GPArotation::GPFoblq warns per run because it is the
	// whole computation there.
	if !convergence {
		insyra.LogDebug("fa", "GPFoblq",
			"oblique rotation did not converge after %d iterations (max %d)",
			iter, maxit)
	}

	if normalize && weights != nil {
		for i := 0; i < rows; i++ {
			w := weights.AtVec(i)
			if w == 0 {
				continue
			}
			for j := 0; j < cols; j++ {
				L.Set(i, j, L.At(i, j)*w)
			}
		}
	}

	var Phi mat.Dense
	Phi.Mul(T.T(), T)

	return map[string]any{
		"loadings":    L,
		"Phi":         &Phi,
		"Th":          T,
		"Table":       table,
		"method":      methodName,
		"orthogonal":  false,
		"convergence": convergence,
		"Gq":          Gq,
		"f":           f,
		"iterations":  iter,
		"penalty":     f,
	}, nil
}

func computeGMatrix(L, Gq, invT *mat.Dense) *mat.Dense {
	// R: G <- -t(t(L) %*% Gq %*% Tmat_inv)
	var LtGq mat.Dense
	LtGq.Mul(L.T(), Gq)

	var temp mat.Dense
	temp.Mul(&LtGq, invT)

	var G mat.Dense
	G.CloneFrom(temp.T())
	G.Scale(-1, &G)
	return &G
}

// gparotationWindow is GPArotation's fwindow under algorithm = "bb": the
// number of recent criterion values a trial step has to improve on.
const gparotationWindow = 10

// nextStepSize is the step size GPArotation 2026.8.2 uses at the start of an
// iteration under algorithm = "bb". The first iteration doubles it. After
// that it is the Barzilai-Borwein estimate sum(dT^2) / |sum(dT * dGp)| from
// the previous iteration, held in [1e-10, 20], and it is left as it was when
// the projected gradient did not change. The sums run in column-major order,
// as R's do.
func nextStepSize(alpha float64, T, prevT, Gp, prevGp *mat.Dense) float64 {
	if prevT == nil {
		return 2 * alpha
	}
	rows, cols := T.Dims()
	var sq, cross, denom float64
	for j := 0; j < cols; j++ {
		for i := 0; i < rows; i++ {
			dT := T.At(i, j) - prevT.At(i, j)
			dG := Gp.At(i, j) - prevGp.At(i, j)
			sq += dT * dT
			cross += dT * dG
			denom += dG * dG
		}
	}
	if !(denom > 0) {
		return alpha
	}
	return math.Max(1e-10, math.Min(sq/math.Abs(cross), 20))
}

// windowMax is the largest criterion value among the last gparotationWindow
// rows of the iteration table, the target a trial step has to improve on.
// A NaN value is skipped, as R's max(..., na.rm = TRUE) skips it.
func windowMax(table [][]float64) float64 {
	best := math.Inf(-1)
	for _, row := range table[max(0, len(table)-gparotationWindow):] {
		if row[1] > best {
			best = row[1]
		}
	}
	return best
}

// normalizeColumns scales every column of X to unit length, leaving a zero
// column as it is.
func normalizeColumns(X *mat.Dense) *mat.Dense {
	rows, cols := X.Dims()
	out := mat.DenseCopyOf(X)
	for j := 0; j < cols; j++ {
		sumSq := 0.0
		for i := 0; i < rows; i++ {
			sumSq += X.At(i, j) * X.At(i, j)
		}
		if sumSq <= 0 {
			continue
		}
		scale := 1 / math.Sqrt(sumSq)
		for i := 0; i < rows; i++ {
			out.Set(i, j, X.At(i, j)*scale)
		}
	}
	return out
}

// safeInverse is GPArotation's safe_inverse: the inverse when the matrix has
// one, and otherwise the pseudo-inverse from its singular value decomposition,
// with singular values at or below sqrt(machine epsilon) treated as zero. A
// decomposition that fails yields NaN, so the trial's criterion is NaN and
// the step is not accepted.
func safeInverse(x *mat.Dense) *mat.Dense {
	if inv, err := invertDense(x); err == nil {
		return inv
	}
	n, _ := x.Dims()
	var svd mat.SVD
	if !svd.Factorize(x, mat.SVDFull) {
		nan := mat.NewDense(n, n, nil)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				nan.Set(i, j, math.NaN())
			}
		}
		return nan
	}
	var U, V mat.Dense
	svd.UTo(&U)
	svd.VTo(&V)
	tol := math.Sqrt(2.220446049250313e-16)
	dInv := mat.NewDense(n, n, nil)
	for i, d := range svd.Values(nil) {
		if d > tol {
			dInv.Set(i, i, 1/d)
		}
	}
	var tmp, out mat.Dense
	tmp.Mul(&V, dInv)
	out.Mul(&tmp, U.T())
	return &out
}

// computeGp computes the projected gradient Gp.
// For oblique rotation, project G onto the tangent space of the manifold
// Gp <- G - T %*% diag(diag(t(T) %*% G))
func computeGp(G, T *mat.Dense) *mat.Dense {
	// R's GPFoblq uses: Gp <- G - T %*% diag(diag(t(T) %*% G))
	// This projects G onto the tangent space at T
	var TtG mat.Dense
	TtG.Mul(T.T(), G)

	// Extract diagonal elements: diag(t(T) %*% G)
	rows, cols := TtG.Dims()
	minDim := rows
	if cols < minDim {
		minDim = cols
	}
	diagVals := make([]float64, minDim)
	for i := 0; i < minDim; i++ {
		diagVals[i] = TtG.At(i, i)
	}

	// Create diagonal matrix
	diagMat := mat.NewDiagDense(minDim, diagVals)

	// Compute T %*% diag(diag(t(T) %*% G))
	var TDiag mat.Dense
	TDiag.Mul(T, diagMat)

	// Gp = G - T %*% diag(diag(t(T) %*% G))
	Gp := mat.DenseCopyOf(G)
	Gp.Sub(Gp, &TDiag)

	return Gp
}

func frobNorm(M *mat.Dense) float64 {
	// Compute Frobenius norm: sqrt(sum of squares of all elements)
	sumSq := 0.0
	rows, cols := M.Dims()
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			val := M.At(i, j)
			sumSq += val * val
		}
	}
	return math.Sqrt(sumSq)
}

func obliqueCriterion(method string, L *mat.Dense, gamma float64) (*mat.Dense, float64, string, error) {
	// Select appropriate criterion function based on method
	switch strings.ToLower(method) {
	case "quartimin":
		Gq, f, _ := vgQQuartimin(L)
		return Gq, f, "vgQ.quartimin", nil

	case "oblimin":
		Gq, f, err := vgQOblimin(L, gamma)
		if err != nil {
			return nil, 0, "", fmt.Errorf("vgQOblimin failed: %v", err)
		}
		return Gq, f, "vgQ.oblimin", nil

	case "simplimax":
		Gq, f, _ := vgQSimplimax(L, simplimaxK(L))
		return Gq, f, "vgQ.simplimax", nil

	case "geominq":
		// gamma is repurposed as ε for geomin (unused by the geomin criterion
		// otherwise). Caller GeominQ() passes the user's GeominEpsilon.
		epsilon := gamma
		if epsilon <= 0 {
			epsilon = 0.01
		}
		Gq, f, _ := vgQGeomin(L, epsilon)
		return Gq, f, "vgQ.geomin", nil

	case "bentlerq":
		Gq, f, _, err := vgQBentler(L)
		if err != nil {
			return nil, 0, "", fmt.Errorf("vgQBentler failed: %v", err)
		}
		return Gq, f, "vgQ.bentler", nil

	default:
		return nil, 0, "", fmt.Errorf("unsupported oblique rotation criterion: %s", method)
	}
}
