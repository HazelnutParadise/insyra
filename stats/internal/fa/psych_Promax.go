// fa/psych_Promax.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// Promax performs Promax rotation.
// Mirrors psych::Promax
func Promax(x *mat.Dense, m int, normalize bool) map[string]interface{} {
	nf, p := x.Dims()
	if nf < 2 {
		return map[string]interface{}{
			"loadings": x,
			"rotmat":   mat.NewDense(nf, nf, nil),
			"Phi":      mat.NewDense(nf, nf, nil),
		}
	}

	// First, varimax rotation
	varimaxResult := Varimax(x, normalize, 1e-5, 1000)
	xx := varimaxResult["loadings"].(*mat.Dense)
	rotmatVarimax := varimaxResult["rotmat"].(*mat.Dense)

	// Q = x * abs(x)^(m-1)
	Q := mat.NewDense(nf, p, nil)
	for i := 0; i < nf; i++ {
		for j := 0; j < p; j++ {
			val := xx.At(i, j)
			Q.Set(i, j, val*math.Pow(math.Abs(val), float64(m-1)))
		}
	}

	// U = coefficients from lm.fit(x, Q)
	// Simplified: U = solve(t(x) %*% x) %*% t(x) %*% Q
	var XtX mat.Dense
	XtX.Mul(xx.T(), xx)
	var XtQ mat.Dense
	XtQ.Mul(xx.T(), Q)
	var U mat.Dense
	err := U.Solve(&XtX, &XtQ)
	if err != nil {
		// Handle singular matrix
		return map[string]interface{}{
			"loadings": xx,
			"rotmat":   rotmatVarimax,
			"Phi":      mat.NewDense(nf, nf, nil),
		}
	}

	// d = diag(solve(t(U) %*% U))
	var UtU mat.Dense
	UtU.Mul(U.T(), &U)
	var UtUInv mat.Dense
	err = UtUInv.Inverse(&UtU)
	if err != nil {
		// Approximation
		d := make([]float64, nf)
		for i := 0; i < nf; i++ {
			d[i] = 1.0
		}
		for i := 0; i < nf; i++ {
			U.Set(i, i, U.At(i, i)*math.Sqrt(d[i]))
		}
	} else {
		d := make([]float64, nf)
		for i := 0; i < nf; i++ {
			d[i] = UtUInv.At(i, i)
		}
		for i := 0; i < nf; i++ {
			U.Set(i, i, U.At(i, i)*math.Sqrt(d[i]))
		}
	}

	// z = x %*% U
	var z mat.Dense
	z.Mul(xx, &U)

	// U = xx$rotmat %*% U
	var rotmat mat.Dense
	rotmat.Mul(rotmatVarimax, &U)

	// ui = solve(U)
	var ui mat.Dense
	err = ui.Inverse(&rotmat)
	if err != nil {
		return map[string]interface{}{
			"loadings": &z,
			"rotmat":   &rotmat,
			"Phi":      mat.NewDense(nf, nf, nil),
		}
	}

	// Phi = ui %*% t(ui)
	var Phi mat.Dense
	Phi.Mul(&ui, ui.T())

	return map[string]interface{}{
		"loadings": &z,
		"rotmat":   &rotmat,
		"Phi":      &Phi,
	}
}
