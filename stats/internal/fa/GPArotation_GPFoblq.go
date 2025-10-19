// fa/GPArotation_GPFoblq.go
package fa

import (
	"fmt"
	"math"
	"strings"

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
	T := mat.DenseCopyOf(Tmat)

	computeL := func(Tcur *mat.Dense) *mat.Dense {
		// Use safe inverse helper to avoid panics when Tcur is singular.
		invT := inverseOrIdentity(Tcur, Tcur.RawMatrix().Rows)
		var invTT mat.Dense
		invTT.CloneFrom(invT)
		invTT.T()
		L := mat.NewDense(rows, cols, nil)
		L.Mul(Aw, &invTT)
		return L
	}

	L := computeL(T)
	Gq, f, methodName, err := obliqueCriterion(method, L, gamma)
	if err != nil {
		return nil, err
	}
	G := computeGMatrix(L, Gq, T)

	table := make([][]float64, 0, max(1, maxit+1))
	convergence := false

	iter := 0
	for iter <= maxit {
		// Add M matrix calculation to match R's implementation, for closer comparison.
		// M <- t(Tmat) %*% Tmat
		var M mat.Dense
		M.Mul(T.T(), T)

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

		// Step size selection with backtracking
		stepAccepted := false
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

			Lnew := computeL(Tnew)
			GqNew, fNew, _, err := obliqueCriterion(method, Lnew, gamma)
			if err != nil {
				// Skip this step if criterion fails
				continue
			}

			improvement := f - fNew
			// Match R's threshold: 0.5 * s^2 * al
			// R code: if (improvement > 0.5 * s^2 * al) break
			threshold := 0.5 * s * s * alpha

			if improvement > threshold {
				// Accept the step
				T = Tnew
				L = Lnew
				Gq = GqNew
				f = fNew
				G = computeGMatrix(L, Gq, T)
				// Match R: double alpha after successful step
				// R code: al <- 2 * al (at the start of iteration)
				// We apply it here after accepting the step
				stepAccepted = true
				break
			} else {
				alpha /= 2
				// If alpha becomes too small, break inner loop and proceed to next main iteration
				if alpha < 1e-10 {
					break
				}
			}
		}

		// If no step was accepted and alpha became too small, reset alpha to a larger value
		if !stepAccepted && alpha < 1e-10 {
			alpha = 1.0 // Reset to initial value for fresh start
		}

		iter++
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

func computeGMatrix(L *mat.Dense, Gq *mat.Dense, T *mat.Dense) *mat.Dense {
	// G <- - solve(T) %*% (t(L) %*% Gq) - match R's GPArotation::GPFoblq
	// Compute transpose of L: t(L) is q x p
	var Lt mat.Dense
	Lt.CloneFrom(L.T())

	// Compute t(L) %*% Gq: q x p * p x q = q x q
	var LtGq mat.Dense
	LtGq.Mul(&Lt, Gq)

	// Compute inverse of T or identity if singular
	invT := inverseOrIdentity(T, T.RawMatrix().Rows)

	// Compute solve(T) %*% (t(L) %*% Gq): q x q
	var G mat.Dense
	G.Mul(invT, &LtGq)

	// Apply negative sign: G <- -G
	G.Scale(-1, &G)

	return &G
}

// computeGp computes the projected gradient Gp.
// For oblique rotation, project G onto the tangent space of the manifold
// Gp <- G - T %*% diag(diag(t(T) %*% G))
func computeGp(G, T *mat.Dense) *mat.Dense {
	// R's GPFoblq uses: Gp <- G - T %*% diag(diag(t(T) %*% G))
	// This projects G onto the tangent space at T

	// Compute t(T) %*% G
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
		// Default to quartimin if method not recognized
		Gq, f, _ := vgQQuartimin(L)
		return Gq, f, "vgQ.quartimin", nil
	}
}
