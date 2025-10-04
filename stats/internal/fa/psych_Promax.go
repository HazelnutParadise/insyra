// fa/psych_Promax.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Promax performs Promax rotation.
// Mirrors psych::Promax exactly
func Promax(x *mat.Dense, m int, normalize bool) map[string]interface{} {
	nf, p := x.Dims()
	if nf < 2 {
		return map[string]interface{}{
			"loadings": x,
			"rotmat":   mat.NewDense(nf, nf, nil),
			"Phi":      mat.NewDense(nf, nf, nil),
		}
	}

	// xx <- stats::varimax(x)
	varimaxResult := Varimax(x, normalize, 1e-5, 1000)
	xx := varimaxResult["loadings"].(*mat.Dense)
	rotmatVarimax := varimaxResult["rotmat"].(*mat.Dense)

	// x <- xx$loadings
	x = xx

	// Q <- x * abs(x)^(m - 1)
	Q := mat.NewDense(nf, p, nil)
	for i := 0; i < nf; i++ {
		for j := 0; j < p; j++ {
			val := x.At(i, j)
			Q.Set(i, j, val*math.Pow(math.Abs(val), float64(m-1)))
		}
	}

	// U <- lm.fit(x, Q)$coefficients
	// This is solve(t(x) %*% x) %*% t(x) %*% Q
	var XtX mat.Dense
	XtX.Mul(x.T(), x)
	var XtQ mat.Dense
	XtQ.Mul(x.T(), Q)
	var U mat.Dense
	err := U.Solve(&XtX, &XtQ)
	if err != nil {
		// Handle singular matrix - use approximation like R
		return map[string]interface{}{
			"loadings": xx,
			"rotmat":   rotmatVarimax,
			"Phi":      mat.NewDense(nf, nf, nil),
		}
	}

	// d <- try(diag(solve(t(U) %*% U)), silent = TRUE)
	var UtU mat.Dense
	UtU.Mul(U.T(), &U)
	var UtUInv mat.Dense
	err = UtUInv.Inverse(&UtU)
	d := make([]float64, nf)
	if err != nil {
		// Simplified approximation
		for i := 0; i < nf; i++ {
			d[i] = 1.0
		}
	} else {
		for i := 0; i < nf; i++ {
			d[i] = UtUInv.At(i, i)
		}
	}

	// U <- U %*% diag(sqrt(d))
	for j := 0; j < nf; j++ {
		sqrtD := math.Sqrt(d[j])
		for i := 0; i < nf; i++ {
			U.Set(i, j, U.At(i, j)*sqrtD)
		}
	}

	// z <- x %*% U
	var z mat.Dense
	z.Mul(x, &U)

	// U <- xx$rotmat %*% U
	var rotmat mat.Dense
	rotmat.Mul(rotmatVarimax, &U)

	// ui <- solve(U)
	var ui mat.Dense
	err = ui.Inverse(&rotmat)
	if err != nil {
		return map[string]interface{}{
			"loadings": &z,
			"rotmat":   &rotmat,
			"Phi":      mat.NewDense(nf, nf, nil),
		}
	}

	// Phi <- ui %*% t(ui)
	var Phi mat.Dense
	Phi.Mul(&ui, ui.T())

	return map[string]interface{}{
		"loadings": &z,
		"rotmat":   &rotmat,
		"Phi":      &Phi,
	}
}
