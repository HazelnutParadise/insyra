// fa/GPArotation_GPFoblq.go
package fa

import (
	"fmt"
	"math"
	"strings"

	"gonum.org/v1/gonum/mat"
)

const debugGPFoblq = true

// NormalizingWeight computes normalizing weights for GPA rotation.
// Mirrors GPArotation::NormalizingWeight for Kaiser normalization.
func NormalizingWeight(A *mat.Dense, normalize bool) *mat.VecDense {
	p, q := A.Dims()
	W := mat.NewVecDense(p, nil)

	if normalize {
		for i := 0; i < p; i++ {
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
func GPFoblq(A *mat.Dense, Tmat *mat.Dense, normalize bool, eps float64, maxit int, method string, gamma float64) map[string]interface{} {
	rows, cols := A.Dims()
	if cols <= 1 {
		panic("rotation does not make sense for single factor models")
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

	alpha := 1.0
	T := mat.DenseCopyOf(Tmat)

	computeL := func(Tcur *mat.Dense) *mat.Dense {
		var invT mat.Dense
		if err := invT.Inverse(Tcur); err != nil {
			panic(fmt.Sprintf("GPFoblq: T inversion failed: %v", err))
		}
		var invTT mat.Dense
		invTT.CloneFrom(&invT)
		invTT.T()
		L := mat.NewDense(rows, cols, nil)
		L.Mul(Aw, &invTT)
		return L
	}

	L := computeL(T)
	Gq, f, methodName := obliqueCriterion(method, L, gamma)
	G := computeGMatrix(L, Gq, T)

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
		if debugGPFoblq && (iter%500 == 0 || iter == maxit) {
			fmt.Printf("GPFoblq debug iter=%d f=%.6f s=%.6e alpha=%.6e\n", iter, f, s, alpha)
		}

		if s < eps {
			convergence = true
			break
		}

		// Step size selection - match R's logic
		alpha *= 2 // Double alpha each iteration like R's al <- 2 * al
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
			GqNew, fNew, _ := obliqueCriterion(method, Lnew, gamma)

			improvement := f - fNew
			threshold := 0.5 * s * s * alpha
			if debugGPFoblq && iter <= 1 && i <= 5 {
				fmt.Printf("GPFoblq inner=%d alpha=%.6e improve=%.6e threshold=%.6e f=%.9f fCand=%.9f\n", i, alpha, improvement, threshold, f, fNew)
			}

			if improvement > threshold {
				// Accept the step
				T = Tnew
				L = Lnew
				Gq = GqNew
				f = fNew
				G = computeGMatrix(L, Gq, T)
				break
			} else {
				alpha /= 2
				if alpha < 1e-10 {
					// No improvement found, use the last attempt anyway
					T = Tnew
					L = Lnew
					Gq = GqNew
					f = fNew
					G = computeGMatrix(L, Gq, T)
					break
				}
			}
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

	return map[string]interface{}{
		"loadings":    L,
		"Phi":         &Phi,
		"Th":          T,
		"Table":       table,
		"method":      methodName,
		"orthogonal":  false,
		"convergence": convergence,
		"Gq":          Gq,
		"f":           f,
	}
}

func computeGMatrix(L *mat.Dense, Gq *mat.Dense, T *mat.Dense) *mat.Dense {
	// G <- - solve(T) %*% (t(L) %*% Gq) - match R's GPArotation::GPFoblq
	var tL mat.Dense
	tL.CloneFrom(L.T()) // t(L) is q x p
	var tLGq mat.Dense
	tLGq.Mul(&tL, Gq) // t(L) %*% Gq is q x p * p x q = q x q
	var invT mat.Dense
	if err := invT.Inverse(T); err != nil {
		panic(fmt.Sprintf("GPFoblq: T inversion failed in computeGMatrix: %v", err))
	}
	var G mat.Dense
	G.Mul(&invT, &tLGq) // solve(T) %*% (t(L) %*% Gq) is q x q
	G.Scale(-1, &G)     // Negative sign
	return &G
}

func computeGp(G *mat.Dense, T *mat.Dense) *mat.Dense {
	rows, cols := G.Dims()
	// Compute rowSums(T * G) element-wise - match R's c(rep(1, nrow(G)) %*% (Tmat * G))
	rowSums := make([]float64, rows)
	for i := 0; i < rows; i++ {
		sum := 0.0
		for j := 0; j < cols; j++ {
			sum += T.At(i, j) * G.At(i, j)
		}
		rowSums[i] = sum
	}

	// Create diag(rowSums)
	diagMat := mat.NewDiagDense(rows, rowSums)

	// Compute T %*% diag(rowSums)
	var Tdiag mat.Dense
	Tdiag.Mul(T, diagMat)

	// Compute G - Tdiag
	Gp := mat.NewDense(rows, cols, nil)
	Gp.Sub(G, &Tdiag)
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

func obliqueCriterion(method string, L *mat.Dense, gamma float64) (*mat.Dense, float64, string) {
	switch strings.ToLower(method) {
	case "quartimin":
		return vgQQuartimin(L)
	case "oblimin":
		Gq, f, err := vgQOblimin(L, gamma)
		if err != nil {
			panic(fmt.Sprintf("vgQOblimin failed: %v", err))
		}
		return Gq, f, "vgQ.oblimin"
	case "simplimax":
		return vgQSimplimax(L, L.RawMatrix().Rows)
	case "geominq":
		return vgQGeomin(L, 0.01)
	case "bentlerq":
		return vgQBentler(L)
	default:
		return vgQQuartimin(L)
	}
}
