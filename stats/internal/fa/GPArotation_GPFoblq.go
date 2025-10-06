// fa/GPArotation_GPFoblq.go
package fa

import (
	"fmt"
	"math"
	"strings"

	"gonum.org/v1/gonum/mat"
)

const debugGPFoblq = false

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

	for iter := 0; iter <= maxit; iter++ {
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

		alpha *= 2

		var Tnext *mat.Dense
		var Lnext *mat.Dense
		var GqNext *mat.Dense
		var fNext float64

		var lastT *mat.Dense
		var lastL *mat.Dense
		var lastGq *mat.Dense
		var lastF float64

		for inner := 0; inner <= 10; inner++ {
			X := mat.DenseCopyOf(T)
			var scaledGp mat.Dense
			scaledGp.Scale(alpha, Gp)
			X.Sub(X, &scaledGp)

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
			Tmatt := mat.NewDense(X.RawMatrix().Rows, colsX, nil)
			Tmatt.Mul(X, diagScale)

			Lcandidate := computeL(Tmatt)
			GqCandidate, fCandidate, _ := obliqueCriterion(method, Lcandidate, gamma)

			improvement := f - fCandidate
			if debugGPFoblq && iter <= 1 && inner <= 5 {
				fmt.Printf("GPFoblq inner=%d alpha=%.6e improve=%.6e threshold=%.6e f=%.9f fCand=%.9f\n", inner, alpha, improvement, 0.5*s*s*alpha, f, fCandidate)
			}

			lastT = Tmatt
			lastL = Lcandidate
			lastGq = GqCandidate
			lastF = fCandidate

			if improvement > 0.5*s*s*alpha {
				Tnext = Tmatt
				Lnext = Lcandidate
				GqNext = GqCandidate
				fNext = fCandidate
				break
			}

			alpha /= 2
		}

		if Tnext == nil {
			// Fall back to the last candidate to mimic R behaviour.
			Tnext = lastT
			Lnext = lastL
			GqNext = lastGq
			fNext = lastF
		}

		if Tnext == nil || Lnext == nil {
			// Numerical failure: abort to avoid nil dereference.
			break
		}

		T = Tnext
		L = Lnext
		Gq = GqNext
		f = fNext
		G = computeGMatrix(L, Gq, T)
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
	var ltGq mat.Dense
	ltGq.Mul(L.T(), Gq)
	var solved mat.Dense
	var temp mat.Dense
	temp.CloneFrom(ltGq.T())
	if err := solved.Solve(T.T(), &temp); err != nil {
		panic(fmt.Sprintf("GPFoblq: rotation gradient solve failed: %v", err))
	}
	G := mat.DenseCopyOf(solved.T())
	G.Scale(-1, G)
	return G
}

func computeGp(G *mat.Dense, T *mat.Dense) *mat.Dense {
	rows, cols := G.Dims()
	Gp := mat.NewDense(rows, cols, nil)
	for j := 0; j < cols; j++ {
		sum := 0.0
		for i := 0; i < rows; i++ {
			sum += T.At(i, j) * G.At(i, j)
		}
		for i := 0; i < rows; i++ {
			Gp.Set(i, j, G.At(i, j)-T.At(i, j)*sum)
		}
	}
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
