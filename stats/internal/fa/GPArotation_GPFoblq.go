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
// debugGPFoblq controls verbose outputs for this algorithm; set to true during development.
var debugGPFoblq = false

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
		for i := range rows {
			w := weights.AtVec(i)
			if w == 0 {
				continue
			}
			for j := 0; j < cols; j++ {
				Aw.Set(i, j, Aw.At(i, j)/w)
			}
		}
	}

	alpha := 0.01
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
		if debugGPFoblq && (iter%10 == 0 || iter <= 5 || iter == maxit) {
			fmt.Printf("GPFoblq debug iter=%d f=%.6f s=%.6e alpha=%.6e\n", iter, f, s, alpha)
		}

		if s < eps {
			convergence = true
			break
		}

		// Step size selection - match R's logic
		// R code: al <- 2*al. Here we start with current alpha and double it *after* a successful step.
		stepAccepted := false
		for i := 0; i <= 10; i++ {
			X := mat.DenseCopyOf(T)
			var scaledGp mat.Dense
			scaledGp.Scale(alpha, Gp)
			X.Sub(X, &scaledGp)

			// Normalize columns of X
			colsX := X.RawMatrix().Cols
			scaleVals := make([]float64, colsX)
			for j := range colsX {
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
			threshold := 0.01 * s * s * alpha // Use smaller coefficient for more lenient acceptance
			if debugGPFoblq && (iter <= 1 || iter >= 495) && i <= 5 {
				fmt.Printf("GPFoblq iter=%d inner=%d alpha=%.6e improve=%.6e threshold=%.6e f=%.9f fCand=%.9f\n", iter, i, alpha, improvement, threshold, f, fNew)
			}

			if improvement > threshold {
				// Accept the step
				T = Tnew
				L = Lnew
				Gq = GqNew
				f = fNew
				G = computeGMatrix(L, Gq, T)
				alpha *= 2 // Double alpha for the *next* iteration, matching R's `al <- 2*al`
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

		// If no step was accepted and alpha became too small, reset alpha to a small positive value
		if !stepAccepted && alpha < 1e-10 {
			alpha = 0.01 // Reset to initial value
		}

		iter++
	}

	if normalize && weights != nil {
		for i := range rows {
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
	if debugGPFoblq {
		fmt.Printf("GPFoblq final Phi:\n")
		for i := 0; i < Phi.RawMatrix().Rows; i++ {
			for j := 0; j < Phi.RawMatrix().Cols; j++ {
				fmt.Printf(" % .6f", Phi.At(i, j))
			}
			fmt.Printf("\n")
		}
		fmt.Printf("GPFoblq final grad=%.6e, iterations=%d\n", frobNorm(computeGp(G, T)), len(table)-1)
	}

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
	var tL mat.Dense
	tL.CloneFrom(L.T()) // t(L) is q x p
	var tLGq mat.Dense
	tLGq.Mul(&tL, Gq) // t(L) %*% Gq is q x p * p x q = q x q
	invT := inverseOrIdentity(T, T.RawMatrix().Rows)
	var G mat.Dense
	G.Mul(invT, &tLGq) // solve(T) %*% (t(L) %*% Gq) is q x q
	G.Scale(-1, &G)    // Negative sign
	return &G
}

// computeGp computes the projected gradient Gp.
// For oblique rotation, empirically the identity projection works best for SPSS compatibility
// Gp <- G
func computeGp(G, T *mat.Dense) *mat.Dense {
	Gp := mat.DenseCopyOf(G)
	return Gp
}

func frobNorm(M *mat.Dense) float64 {
	sum := 0.0
	for i := 0; i < M.RawMatrix().Rows; i++ {
		for j := 0; j < M.RawMatrix().Cols; j++ {
			val := M.At(i, j)
			sum += val * val
		}
	}
	return math.Sqrt(sum)
}

func obliqueCriterion(method string, L *mat.Dense, gamma float64) (*mat.Dense, float64, string, error) {
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
		Gq, f, _ := vgQSimplimax(L, L.RawMatrix().Rows)
		return Gq, f, "vgQ.simplimax", nil
	case "geominq":
		Gq, f, _ := vgQGeomin(L, 0.01)
		return Gq, f, "vgQ.geomin", nil
	case "bentlerq":
		Gq, f, _, err := vgQBentler(L)
		if err != nil {
			return nil, 0, "", err
		}
		return Gq, f, "vgQ.bentler", nil
	default:
		Gq, f, _ := vgQQuartimin(L)
		return Gq, f, "vgQ.quartimin", nil
	}
}
