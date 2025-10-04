// fa/GPArotation_GPFoblq.go
package fa

import (
	"fmt"
	"math"

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
func GPFoblq(A *mat.Dense, Tmat *mat.Dense, normalize bool, eps float64, maxit int, method string) map[string]interface{} {
	nf, _ := A.Dims()
	if nf <= 1 {
		panic("rotation does not make sense for single factor models.")
	}

	var W *mat.VecDense
	if normalize {
		W = NormalizingWeight(A, normalize)
		normalize = true
		A = mat.DenseCopyOf(A)
		for i := 0; i < A.RawMatrix().Rows; i++ {
			for j := 0; j < A.RawMatrix().Cols; j++ {
				A.Set(i, j, A.At(i, j)/W.AtVec(i))
			}
		}
	}

	al := 1.0
	var TmatInv mat.Dense
	err := TmatInv.Inverse(Tmat)
	if err != nil {
		panic("Tmat is singular")
	}
	L := mat.NewDense(A.RawMatrix().Rows, A.RawMatrix().Cols, nil)
	L.Mul(A, TmatInv.T())

	// Call vgQ method
	var VgQ map[string]interface{}
	switch method {
	case "quartimin":
		Gq, f, _ := vgQQuartimin(L)
		VgQ = map[string]interface{}{
			"Gq":     Gq,
			"f":      f,
			"Method": "quartimin",
		}
	default:
		Gq, f, _ := vgQQuartimin(L)
		VgQ = map[string]interface{}{
			"Gq":     Gq,
			"f":      f,
			"Method": "quartimin",
		}
	}

	var TmatInv2 mat.Dense
	TmatInv2.Inverse(Tmat)
	var temp mat.Dense
	temp.Mul(L.T(), VgQ["Gq"].(*mat.Dense))
	temp.Mul(&temp, &TmatInv2)
	G := mat.NewDense(temp.RawMatrix().Cols, temp.RawMatrix().Rows, nil)
	G.Scale(-1, temp.T())

	f := VgQ["f"].(float64)
	var Table [][]float64

	var VgQt map[string]interface{}
	switch method {
	case "quartimin":
		Gq, f2, _ := vgQQuartimin(L)
		VgQt = map[string]interface{}{
			"Gq": Gq,
			"f":  f2,
		}
	default:
		Gq, f2, _ := vgQQuartimin(L)
		VgQt = map[string]interface{}{
			"Gq": Gq,
			"f":  f2,
		}
	}

	convergence := false
	s := eps + 1

	for iter := 0; iter <= maxit; iter++ {
		// Gp <- G - Tmat %*% diag(c(rep(1, nrow(G)) %*% (Tmat * G)))
		nrowG, _ := G.Dims()
		diagVec := make([]float64, nrowG)
		for i := 0; i < nrowG; i++ {
			sum := 0.0
			for j := 0; j < Tmat.RawMatrix().Cols; j++ {
				sum += Tmat.At(i, j) * G.At(j, i)
			}
			diagVec[i] = sum
		}

		Gp := mat.NewDense(G.RawMatrix().Rows, G.RawMatrix().Cols, nil)
		Gp.CloneFrom(G)
		for i := 0; i < nrowG; i++ {
			for j := 0; j < G.RawMatrix().Cols; j++ {
				Gp.Set(j, i, G.At(j, i)-Tmat.At(i, j)*diagVec[i])
			}
		}

		// s <- sqrt(sum(diag(crossprod(Gp))))
		var GpTGp mat.Dense
		GpTGp.Mul(Gp.T(), Gp)
		s = 0
		for i := 0; i < GpTGp.RawMatrix().Rows; i++ {
			s += GpTGp.At(i, i)
		}
		s = math.Sqrt(s)

		Table = append(Table, []float64{float64(iter), f, math.Log10(s), al})

		if s < eps {
			convergence = true
			break
		}

		al *= 2
		var Tmatt *mat.Dense
		for i := 0; i <= 10; i++ {
			X := mat.NewDense(Tmat.RawMatrix().Rows, Tmat.RawMatrix().Cols, nil)
			X.CloneFrom(Tmat)
			for k := 0; k < X.RawMatrix().Rows; k++ {
				for l := 0; l < X.RawMatrix().Cols; l++ {
					X.Set(k, l, Tmat.At(k, l)-al*Gp.At(l, k))
				}
			}

			// v <- 1/sqrt(c(rep(1, nrow(X)) %*% X^2))
			nrowX, _ := X.Dims()
			v := make([]float64, nrowX)
			for k := 0; k < nrowX; k++ {
				sum := 0.0
				for l := 0; l < X.RawMatrix().Cols; l++ {
					sum += X.At(k, l) * X.At(k, l)
				}
				v[k] = 1.0 / math.Sqrt(sum)
			}

			Tmatt = mat.NewDense(X.RawMatrix().Rows, X.RawMatrix().Cols, nil)
			for k := 0; k < nrowX; k++ {
				for l := 0; l < X.RawMatrix().Cols; l++ {
					Tmatt.Set(k, l, X.At(k, l)*v[k])
				}
			}

			var TmattInv mat.Dense
			TmattInv.Inverse(Tmatt)
			L.Mul(A, TmattInv.T())

			// Call vgQ again
			switch method {
			case "quartimin":
				GqNew, fNew, _ := vgQQuartimin(L)
				VgQt = map[string]interface{}{
					"Gq": GqNew,
					"f":  fNew,
				}
			default:
				GqNew, fNew, _ := vgQQuartimin(L)
				VgQt = map[string]interface{}{
					"Gq": GqNew,
					"f":  fNew,
				}
			}

			improvement := f - VgQt["f"].(float64)
			if improvement > 0.5*s*s*al {
				break
			}
			al /= 2
		}

		Tmat = Tmatt
		f = VgQt["f"].(float64)

		var TmattInv mat.Dense
		TmattInv.Inverse(Tmat)
		var temp2 mat.Dense
		temp2.Mul(L.T(), VgQt["Gq"].(*mat.Dense))
		temp2.Mul(&temp2, &TmattInv)
		G.Scale(-1, temp2.T())
	}

	if !convergence {
		fmt.Printf("convergence not obtained in GPFoblq. %d iterations used.\n", maxit)
	}

	if normalize {
		for i := 0; i < L.RawMatrix().Rows; i++ {
			for j := 0; j < L.RawMatrix().Cols; j++ {
				L.Set(i, j, L.At(i, j)*W.AtVec(i))
			}
		}
	}

	Phi := mat.NewDense(Tmat.RawMatrix().Cols, Tmat.RawMatrix().Cols, nil)
	var TmatT mat.Dense
	TmatT.CloneFrom(Tmat)
	TmatT.T()
	Phi.Mul(&TmatT, Tmat)

	return map[string]interface{}{
		"loadings":    L,
		"Phi":         Phi,
		"Th":          Tmat,
		"Table":       Table,
		"method":      VgQ["Method"],
		"orthogonal":  false,
		"convergence": convergence,
		"Gq":          VgQt["Gq"],
	}
}
