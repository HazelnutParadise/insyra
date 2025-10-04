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
// Returns: Gq (gradient), f (objective), method
func vgQBentler(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// L2 = L^2
	L2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			L2.Set(i, j, l*l)
		}
	}

	// M = t(L2) %*% L2
	M := mat.NewDense(k, k, nil)
	M.Mul(L2.T(), L2)

	// D = diag(diag(M))
	D := mat.NewDense(k, k, nil)
	for i := 0; i < k; i++ {
		D.Set(i, i, M.At(i, i))
	}

	// solveM = solve(M)
	solveM := mat.NewDense(k, k, nil)
	err := solveM.Inverse(M)
	if err != nil {
		// Handle error, but for now assume invertible
		panic(err)
	}

	// solveD = solve(D)
	solveD := mat.NewDense(k, k, nil)
	err = solveD.Inverse(D)
	if err != nil {
		panic(err)
	}

	// diff = solveM - solveD
	diff := mat.NewDense(k, k, nil)
	diff.Sub(solveM, solveD)

	// L2Diff = L2 %*% diff
	L2Diff := mat.NewDense(p, k, nil)
	L2Diff.Mul(L2, diff)

	// Gq = -L * L2Diff
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			Gq.Set(i, j, -L.At(i, j)*L2Diff.At(i, j))
		}
	}

	// f = -(log(det(M)) - log(det(D)))/4
	detM := mat.Det(M)
	detD := mat.Det(D)
	f = -(math.Log(detM) - math.Log(detD)) / 4

	method = "Bentler's criterion"
	return
}
