// fa/GPArotation_GPFoblq.go
package fa

import (
	"fmt"
	"math"
	"strings"

	"gonum.org/v1/gonum/mat"
)

// NormalizingWeight computes normalizing weights for GPA rotation.
// Simplified implementation based on GPArotation logic
func NormalizingWeight(A *mat.Dense, normalize bool) *mat.VecDense {
	p, q := A.Dims()
	W := mat.NewVecDense(p, nil)

	if normalize {
		// Kaiser normalization: sqrt of communalities
		for i := 0; i < p; i++ {
			sum := 0.0
			for j := 0; j < q; j++ {
				sum += A.At(i, j) * A.At(i, j)
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

// GPFoblq performs oblique rotation using GPA.
// Mirrors GPArotation::GPFoblq exactly
func GPFoblq(A *mat.Dense, Tmat *mat.Dense, normalize bool, eps float64, maxit int, method string, gamma float64) map[string]interface{} {
	rows, cols := A.Dims()
	if cols <= 1 {
		panic("rotation does not make sense for single factor models.")
	}

	Awork := mat.DenseCopyOf(A)
	var weights *mat.VecDense
	if normalize {
		weights = NormalizingWeight(A, true)
		for i := 0; i < rows; i++ {
			w := weights.AtVec(i)
			if w == 0 {
				continue
			}
			for j := 0; j < cols; j++ {
				Awork.Set(i, j, Awork.At(i, j)/w)
			}
		}
	}

	al := 1.0
	Tcurrent := mat.DenseCopyOf(Tmat)

	var Tinv mat.Dense
	if err := Tinv.Inverse(Tcurrent); err != nil {
		panic(fmt.Sprintf("Tmat inversion failed: %v", err))
	}
	var TinvT mat.Dense
	TinvT.CloneFrom(&Tinv)
	TinvT.T()

	Lcurrent := mat.NewDense(rows, cols, nil)
	Lcurrent.Mul(Awork, &TinvT)

	gqCurrent, fCurrent, methodName := obliqueCriterion(method, Lcurrent, gamma)
	fmt.Printf("GPFoblq initial f=%.6f, ||Gq||=%.6f\n", fCurrent, mat.Norm(gqCurrent, 2))
	G := computeObliqueGradient(Lcurrent, gqCurrent, Tcurrent)

	table := make([][]float64, 0, maxit+1)
	convergence := false
	var s float64

	for iter := 0; iter <= maxit; iter++ {
		currentAlpha := al
		Gp := computeGp(G, Tcurrent)

		var GpTGp mat.Dense
		GpTGp.Mul(Gp.T(), Gp)
		s = 0
		for i := 0; i < GpTGp.RawMatrix().Rows; i++ {
			s += GpTGp.At(i, i)
		}
		s = math.Sqrt(s)
		fmt.Printf("GPFoblq iter=%d, s=%.6e, f=%.6f, alpha=%.6f\n", iter, s, fCurrent, currentAlpha)
		table = append(table, []float64{float64(iter), fCurrent, math.Log10(s), currentAlpha})

		if s < eps {
			convergence = true
			break
		}

		al *= 2

		var Tnext *mat.Dense
		var Lnext *mat.Dense
		var gqNext *mat.Dense
		var fNext float64

		for inner := 0; inner <= 10; inner++ {
			X := mat.NewDense(Tcurrent.RawMatrix().Rows, Tcurrent.RawMatrix().Cols, nil)
			X.CloneFrom(Tcurrent)

			var scaledGp mat.Dense
			scaledGp.Scale(al, Gp)
			X.Sub(Tcurrent, &scaledGp)

			rowsT, colsT := X.Dims()
			v := make([]float64, colsT)
			for j := 0; j < colsT; j++ {
				sumsq := 0.0
				for i := 0; i < rowsT; i++ {
					val := X.At(i, j)
					sumsq += val * val
				}
				if sumsq <= 0 {
					v[j] = 1.0
				} else {
					v[j] = 1.0 / math.Sqrt(sumsq)
				}
			}

			Tmatt := mat.NewDense(rowsT, colsT, nil)
			for i := 0; i < rowsT; i++ {
				for j := 0; j < colsT; j++ {
					Tmatt.Set(i, j, X.At(i, j)*v[j])
				}
			}

			var TmattInv mat.Dense
			if err := TmattInv.Inverse(Tmatt); err != nil {
				panic(fmt.Sprintf("Tmatt inversion failed: %v", err))
			}
			var TmattInvT mat.Dense
			TmattInvT.CloneFrom(&TmattInv)
			TmattInvT.T()

			Lcandidate := mat.NewDense(rows, cols, nil)
			Lcandidate.Mul(Awork, &TmattInvT)

			gqCandidate, fCandidate, _ := obliqueCriterion(method, Lcandidate, gamma)
			improvement := fCurrent - fCandidate
			fmt.Printf("GPFoblq inner iter=%d improvement=%.6e threshold=%.6e alpha=%.6e\n", inner, improvement, 0.5*s*s*al, al)

			Tnext = Tmatt
			Lnext = Lcandidate
			gqNext = gqCandidate
			fNext = fCandidate
			if improvement > 0.5*s*s*al {
				break
			}
			al /= 2
		}

		if Tnext == nil || Lnext == nil {
			Tnext = Tcurrent
			Lnext = Lcurrent
			gqNext = gqCurrent
			fNext = fCurrent
		}

		Tcurrent = Tnext
		Lcurrent = Lnext
		gqCurrent = gqNext
		fCurrent = fNext
		G = computeObliqueGradient(Lcurrent, gqCurrent, Tcurrent)
		fmt.Printf("GPFoblq iter=%d gradient norm=%.6e\n", iter, mat.Norm(G, 2))
	}

	if normalize && weights != nil {
		for i := 0; i < rows; i++ {
			w := weights.AtVec(i)
			for j := 0; j < cols; j++ {
				Lcurrent.Set(i, j, Lcurrent.At(i, j)*w)
			}
		}
	}

	var Tt mat.Dense
	Tt.CloneFrom(Tcurrent)
	Tt.T()
	var Phi mat.Dense
	Phi.Mul(&Tt, Tcurrent)

	return map[string]interface{}{
		"loadings":    Lcurrent,
		"Phi":         &Phi,
		"Th":          Tcurrent,
		"Table":       table,
		"method":      methodName,
		"orthogonal":  false,
		"convergence": convergence,
		"Gq":          gqCurrent,
		"f":           fCurrent,
	}
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

func computeObliqueGradient(L *mat.Dense, Gq *mat.Dense, T *mat.Dense) *mat.Dense {
	var Tinv mat.Dense
	if err := Tinv.Inverse(T); err != nil {
		panic(fmt.Sprintf("rotation gradient inversion failed: %v", err))
	}
	var temp mat.Dense
	temp.Mul(L.T(), Gq)
	temp.Mul(&temp, &Tinv)
	grad := mat.NewDense(temp.RawMatrix().Cols, temp.RawMatrix().Rows, nil)
	grad.Scale(-1, temp.T())
	return grad
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
