// fa/GPArotation_vgQ_bentler.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQBentler computes the objective and gradient for Bentler's criterion rotation.
// Mirrors GPArotation::vgQ.bentler(L)
//
// L2 <- L^2
// M <- crossprod(L2)
// D <- diag(diag(M))
// Gq <- -L * (L2 %*% (solve(M) - solve(D)))
// f <- -(log(det(M)) - log(det(D)))/4
//
// Returns: Gq (gradient), f (objective), method, error
func vgQBentler(L *mat.Dense) (Gq *mat.Dense, f float64, method string, err error) {
	p, q := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, q, nil)
	for i := range p {
		for j := range q {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// M = crossprod(L2) = t(L2) %*% L2
	var M mat.Dense
	M.Mul(L2.T(), L2)

	// D = diag(diag(M))
	D := mat.NewDense(q, q, nil)
	for i := range q {
		D.Set(i, i, M.At(i, i))
	}

	// solve(M)
	var solveM mat.Dense
	if err = solveM.Inverse(&M); err != nil {
		return nil, 0, "", err
	}

	// solve(D)
	var solveD mat.Dense
	if err = solveD.Inverse(D); err != nil {
		return nil, 0, "", err
	}

	// solve(M) - solve(D)
	var diff mat.Dense
	diff.Sub(&solveM, &solveD)

	// L2 %*% diff
	var L2_diff mat.Dense
	L2_diff.Mul(L2, &diff)

	// Gq = -L * L2_diff
	Gq = mat.NewDense(p, q, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < q; j++ {
			Gq.Set(i, j, -L.At(i, j)*L2_diff.At(i, j))
		}
	}

	// det(M)
	detM := mat.Det(&M)
	// det(D)
	detD := mat.Det(D)

	// f = -(log(det(M)) - log(det(D)))/4
	f = -(math.Log(detM) - math.Log(detD)) / 4

	method = "Bentler's criterion"
	return
}
