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

	alpha := 1.0 // Start with larger alpha for better initial progress

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

	computeL := func(Tcur *mat.Dense) (*mat.Dense, error) {
		invT, err := invertDense(Tcur)
		if err != nil {
			return nil, fmt.Errorf("failed to invert oblique rotation matrix: %w", err)
		}
		L := mat.NewDense(rows, cols, nil)
		L.Mul(Aw, invT.T())
		return L, nil
	}

	L, err := computeL(T)
	if err != nil {
		return nil, err
	}
	Gq, f, methodName, err := obliqueCriterion(method, L, gamma)
	if err != nil {
		return nil, err
	}
	G, err := computeGMatrix(L, Gq, T)
	if err != nil {
		return nil, err
	}

	table := make([][]float64, 0, max(1, maxit+1))
	convergence := false

	iter := 0
	for iter <= maxit {
		Gp := computeGp(G, T)
		s := frobNorm(Gp)
		logTerm := math.Inf(-1)
		if s > 0 {
			logTerm = math.Log10(s)
		}
		table = append(table, []float64{float64(iter), f, logTerm, alpha})

		if s < eps {
			convergence = true
			break
		}
		// Match R's strategy: double alpha at start of each iteration
		// R code: al <- 2 * al
		alpha *= 2

		// Step size selection with backtracking. R updates Tmat to the last
		// trial Tmatt even when the sufficient-improvement condition does not
		// trigger before the inner loop limit.
		var lastT, lastL, lastGq *mat.Dense
		var lastF float64
		for i := 0; i <= 10; i++ {
			X := mat.DenseCopyOf(T)
			var scaledGp mat.Dense
			scaledGp.Scale(alpha, Gp)
			X.Sub(X, &scaledGp)

			// Normalize columns of X
			colsX := X.RawMatrix().Cols
			scaleVals := make([]float64, colsX)
			for j := 0; j < colsX; j++ {
				sumSq := 0.0
				for i := 0; i < X.RawMatrix().Rows; i++ {
					val := X.At(i, j)
					sumSq += val * val
				}
				if sumSq <= 0 {
					scaleVals[j] = 1.0
				} else {
					scaleVals[j] = 1.0 / math.Sqrt(sumSq)
				}
			}
			diagScale := mat.NewDiagDense(colsX, scaleVals)
			Tnew := mat.NewDense(X.RawMatrix().Rows, colsX, nil)
			Tnew.Mul(X, diagScale)

			Lnew, err := computeL(Tnew)
			if err != nil {
				continue
			}
			GqNew, fNew, _, err := obliqueCriterion(method, Lnew, gamma)
			if err != nil {
				// Skip this step if criterion fails
				continue
			}
			lastT = Tnew
			lastL = Lnew
			lastGq = GqNew
			lastF = fNew

			improvement := f - fNew
			// Match R's threshold: 0.5 * s^2 * al
			// R code: if (improvement > 0.5 * s^2 * al) break
			threshold := 0.5 * s * s * alpha

			if improvement > threshold {
				break
			} else {
				alpha /= 2
			}
		}

		if lastT != nil {
			T = lastT
			L = lastL
			Gq = lastGq
			f = lastF
			G, err = computeGMatrix(L, Gq, T)
			if err != nil {
				return nil, err
			}
		}

		iter++
	}

	// Warn on non-convergence — R's GPArotation::GPFoblq emits a warning
	// when the gradient norm hasn't dropped below eps within maxit iters.
	if !convergence {
		insyra.LogWarning("fa", "GPFoblq",
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

func computeGMatrix(L *mat.Dense, Gq *mat.Dense, T *mat.Dense) (*mat.Dense, error) {
	// R: G <- -t(t(L) %*% Gq %*% solve(Tmat))
	var Lt mat.Dense
	Lt.CloneFrom(L.T())

	var LtGq mat.Dense
	LtGq.Mul(&Lt, Gq)

	invT, err := invertDense(T)
	if err != nil {
		return nil, fmt.Errorf("failed to invert oblique rotation matrix for gradient: %w", err)
	}

	var temp mat.Dense
	temp.Mul(&LtGq, invT)

	var G mat.Dense
	G.CloneFrom(temp.T())
	G.Scale(-1, &G)
	return &G, nil
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
		// Use number of rows as k parameter for simplimax
		k := L.RawMatrix().Rows
		Gq, f, _ := vgQSimplimax(L, k)
		return Gq, f, "vgQ.simplimax", nil

	case "geominq":
		// Use default epsilon = 0.01 for geomin
		epsilon := 0.01
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
