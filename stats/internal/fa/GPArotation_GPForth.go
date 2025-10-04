// fa/GPArotation_GPForth.go
package fa

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

// GPForth performs orthogonal rotation using GPA.
// Mirrors GPArotation::GPForth
// Simplified implementation
func GPForth(A *mat.Dense, Tmat *mat.Dense, normalize bool, eps float64, maxit int, method string) map[string]interface{} {
	nf, _ := A.Dims()
	if nf <= 1 {
		panic("rotation does not make sense for single factor models.")
	}

	// Simplified: assume no normalization
	if normalize {
		// Normalize by communalities (simplified)
		for i := 0; i < A.RawMatrix().Rows; i++ {
			sum := 0.0
			for j := 0; j < A.RawMatrix().Cols; j++ {
				sum += A.At(i, j) * A.At(i, j)
			}
			if sum > 0 {
				for j := 0; j < A.RawMatrix().Cols; j++ {
					A.Set(i, j, A.At(i, j)/math.Sqrt(sum))
				}
			}
		}
	}

	al := 1.0
	L := mat.NewDense(A.RawMatrix().Rows, A.RawMatrix().Cols, nil)
	L.Mul(A, Tmat)

	// Call vgQ method
	var Gq *mat.Dense
	var f float64
	switch method {
	case "varimax":
		Gq, f, _ = vgQVarimax(L)
	default:
		Gq, f, _ = vgQVarimax(L)
	}

	G := mat.NewDense(A.RawMatrix().Cols, Gq.RawMatrix().Cols, nil)
	G.Mul(A.T(), Gq)

	var Table [][]float64

	convergence := false
	s := eps + 1 // Initialize s > eps

	for iter := 0; iter <= maxit; iter++ {
		M := mat.NewDense(Tmat.RawMatrix().Cols, G.RawMatrix().Cols, nil)
		M.Mul(Tmat.T(), G)

		S := mat.NewDense(M.RawMatrix().Rows, M.RawMatrix().Cols, nil)
		S.Add(M, M.T())
		S.Scale(0.5, S)

		Gp := mat.NewDense(G.RawMatrix().Rows, G.RawMatrix().Cols, nil)
		temp := mat.NewDense(Tmat.RawMatrix().Rows, S.RawMatrix().Cols, nil)
		temp.Mul(Tmat, S)
		Gp.Sub(G, temp)

		// Calculate s = sqrt(sum(diag(Gp' * Gp)))
		GpTGp := mat.NewDense(Gp.RawMatrix().Cols, Gp.RawMatrix().Cols, nil)
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
		Tmatt := mat.NewDense(Tmat.RawMatrix().Rows, Tmat.RawMatrix().Cols, nil)
		Tmatt.CloneFrom(Tmat)
		var GqNew *mat.Dense
		var fNew float64

		for i := 0; i <= 10; i++ {
			X := mat.NewDense(Tmat.RawMatrix().Rows, Tmat.RawMatrix().Cols, nil)
			tempGp := mat.NewDense(Gp.RawMatrix().Rows, Gp.RawMatrix().Cols, nil)
			tempGp.Scale(al, Gp)
			X.Sub(Tmat, tempGp)

			// SVD
			var svd mat.SVD
			ok := svd.Factorize(X, mat.SVDThin)
			if !ok {
				panic("SVD failed")
			}
			var U mat.Dense
			var V mat.Dense
			svd.UTo(&U)
			svd.VTo(&V)

			Tmatt.Mul(&U, V.T())

			L.Mul(A, Tmatt)

			// Call vgQ again
			GqNew, fNew, _ = vgQVarimax(L)
			if fNew < (f - 0.5*s*s*al) {
				break
			}
			al /= 2
		}

		Tmat = Tmatt
		f = fNew
		G.Mul(A.T(), GqNew)
	}

	if !convergence {
		fmt.Printf("convergence not obtained in GPForth. %d iterations used.\n", maxit)
	}

	// Simplified: no denormalization

	return map[string]interface{}{
		"loadings":    L,
		"Th":          Tmat,
		"Table":       Table,
		"method":      method,
		"orthogonal":  true,
		"convergence": convergence,
		"Gq":          Gq,
	}
}
