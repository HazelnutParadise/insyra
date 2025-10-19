// fa/GPArotation_GPForth.go
package fa

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

// GPForth performs orthogonal rotation using GPA.
// Mirrors GPArotation::GPForth exactly
func GPForth(A *mat.Dense, Tmat *mat.Dense, normalize bool, eps float64, maxit int, method string) (map[string]any, error) {
	rows, cols := A.Dims() // A is p x nf (variables x factors), nf is number of factors
	if cols <= 1 {
		return nil, fmt.Errorf("rotation does not make sense for single factor models")
	}

	var W *mat.VecDense
	// Match R logic: if ((!is.logical(normalize)) || normalize)
	// In Go, since normalize is bool, this simplifies to: if normalize
	if normalize {
		W = NormalizingWeight(A, normalize)
		normalize = true
		A = mat.DenseCopyOf(A)
		for i := 0; i < rows; i++ {
			for j := 0; j < cols; j++ {
				A.Set(i, j, A.At(i, j)/W.AtVec(i))
			}
		}
	}

	al := 1.0
	L := mat.NewDense(rows, cols, nil)
	L.Mul(A, Tmat)

	// Method <- paste("vgQ", method, sep = ".")
	// VgQ <- do.call(Method, append(list(L), methodArgs))
	var VgQ map[string]any
	switch method {
	case "varimax":
		Gq, f, _ := vgQVarimax(L)
		VgQ = map[string]any{
			"Gq":     Gq,
			"f":      f,
			"Method": "vgQ." + method,
		}
	case "quartimax":
		Gq, f, _ := vgQQuartimax(L)
		VgQ = map[string]any{
			"Gq":     Gq,
			"f":      f,
			"Method": "vgQ." + method,
		}
	case "geominT":
		Gq, f, _ := vgQGeomin(L, 0.01)
		VgQ = map[string]any{
			"Gq":     Gq,
			"f":      f,
			"Method": "vgQ." + method,
		}
	case "bentlerT":
		Gq, f, _, err := vgQBentler(L)
		if err != nil {
			return nil, err
		}
		VgQ = map[string]any{
			"Gq":     Gq,
			"f":      f,
			"Method": "vgQ." + method,
		}
	case "targetT":
		// For target rotation, we need a Target matrix, but for now use nil
		// This would need to be passed as parameter
		return nil, fmt.Errorf("targetT requires Target matrix parameter")
	default:
		Gq, f, _ := vgQVarimax(L)
		VgQ = map[string]any{
			"Gq":     Gq,
			"f":      f,
			"Method": "vgQ." + method,
		}
	}

	// G <- crossprod(A, VgQ$Gq)
	G := mat.NewDense(cols, VgQ["Gq"].(*mat.Dense).RawMatrix().Cols, nil)
	G.Mul(A.T(), VgQ["Gq"].(*mat.Dense))

	f := VgQ["f"].(float64)
	var Table [][]float64

	var convergence bool
	var s float64
	var iter int
	var VgQt map[string]any

	for iter = 0; iter <= maxit; iter++ {
		// M <- crossprod(Tmat, G)
		M := mat.NewDense(Tmat.RawMatrix().Cols, G.RawMatrix().Cols, nil)
		M.Mul(Tmat.T(), G)

		// S <- (M + t(M))/2
		S := mat.NewDense(M.RawMatrix().Rows, M.RawMatrix().Cols, nil)
		Mt := M.T() // Get transpose view
		S.Add(M, Mt)
		S.Scale(0.5, S)

		// Gp <- G - Tmat %*% S
		Gp := mat.NewDense(G.RawMatrix().Rows, G.RawMatrix().Cols, nil)
		var TS mat.Dense
		TS.Mul(Tmat, S)
		Gp.Sub(G, &TS)

		// s <- sqrt(sum(diag(crossprod(Gp))))
		var GpTGp mat.Dense
		GpTGp.Mul(Gp.T(), Gp)
		sLocal := 0.0
		for i := 0; i < GpTGp.RawMatrix().Rows; i++ {
			sLocal += GpTGp.At(i, i)
		}
		sLocal = math.Sqrt(sLocal)
		s = sLocal // Update outer s

		Table = append(Table, []float64{float64(iter), f, math.Log10(sLocal), al})

		if sLocal < eps {
			break
		}

		al *= 2
		var Tmatt *mat.Dense
		for i := 0; i <= 10; i++ {
			// X <- Tmat - al * Gp
			X := mat.NewDense(Tmat.RawMatrix().Rows, Tmat.RawMatrix().Cols, nil)
			X.CloneFrom(Tmat)
			var alGp mat.Dense
			alGp.Scale(al, Gp)
			X.Sub(Tmat, &alGp)

			// UDV <- svd(X)
			var svd mat.SVD
			ok := svd.Factorize(X, mat.SVDThin)
			if !ok {
				return nil, fmt.Errorf("SVD failed during rotation")
			}
			var U mat.Dense
			var V mat.Dense
			svd.UTo(&U)
			svd.VTo(&V)

			// Tmatt <- UDV$u %*% t(UDV$v)
			Tmatt = mat.NewDense(U.RawMatrix().Rows, V.RawMatrix().Cols, nil)
			var Vt mat.Dense
			Vt.CloneFrom(&V)
			Vt.T()
			Tmatt.Mul(&U, &Vt)

			// L <- A %*% Tmatt
			L.Mul(A, Tmatt)

			// VgQt <- do.call(Method, append(list(L), methodArgs))
			switch method {
			case "varimax":
				GqNew, fNew, _ := vgQVarimax(L)
				VgQt = map[string]any{
					"Gq": GqNew,
					"f":  fNew,
				}
			case "quartimax":
				GqNew, fNew, _ := vgQQuartimax(L)
				VgQt = map[string]any{
					"Gq": GqNew,
					"f":  fNew,
				}
			default:
				GqNew, fNew, _ := vgQVarimax(L)
				VgQt = map[string]any{
					"Gq": GqNew,
					"f":  fNew,
				}
			}

			// if (VgQt$f < (f - 0.5 * s^2 * al)) break
			if VgQt["f"].(float64) < (f - 0.5*s*s*al) {
				break
			}
			al /= 2
		}

		Tmat = Tmatt
		f = VgQt["f"].(float64)
		// G <- crossprod(A, VgQt$Gq)
		G.Mul(A.T(), VgQt["Gq"].(*mat.Dense))
	}

	convergence = (s < eps)
	if iter == maxit && !convergence {
		fmt.Printf("convergence not obtained in GPForth. %d iterations used.\n", maxit)
	}

	if normalize {
		for i := 0; i < L.RawMatrix().Rows; i++ {
			for j := 0; j < L.RawMatrix().Cols; j++ {
				L.Set(i, j, L.At(i, j)*W.AtVec(i))
			}
		}
	}

	return map[string]any{
		"loadings":    L,
		"Th":          Tmat,
		"Table":       Table,
		"method":      VgQ["Method"],
		"orthogonal":  true,
		"convergence": convergence,
		"Gq":          VgQt["Gq"],
		"f":           f,
		"iterations":  iter,
	}, nil
}
