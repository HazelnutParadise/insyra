// fa/promax.go  （或貼到 rotate.go 同一個 package）
package fa

import (
	"errors"
	"math"

	"gonum.org/v1/gonum/mat"
)

// Promax performs Promax rotation on an unrotated loading matrix A (p×q).
// Steps:
//  1. Varimax (orthogonal) to get Z = A * T0
//  2. Target Q = Z ⊙ |Z|^(m-1)
//  3. U = (Zᵀ Z)^{-1} Zᵀ Q
//  4. Column-scale U so that diag(Uᵀ (Zᵀ Z) U) = 1
//  5. R = T0 * U; L = A * R; Phi = R^{-1} R^{-T}
//
// Returns: L (loadings), R (rotmat), Phi (factor correlations)
func Promax(A *mat.Dense, m int, normalize bool) (L, R, Phi *mat.Dense, err error) {
	if m <= 1 {
		m = 4 // psych 的預設（Promax power）
	}
	p, q := A.Dims()
	if q < 2 {
		return nil, nil, nil, errors.New("Promax: rotation does not make sense for single factor models")
	}

	// 1) Varimax 先做正交旋轉：Z = A * T0
	Lvar, T0, err := RotateOrth(A, "varimax", &RotOpts{
		Eps: 1e-5, MaxIter: 1000, Alpha0: 1.0,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	Z := mat.DenseCopyOf(Lvar) // p×q

	// 2) 目標 Q = Z ⊙ |Z|^(m-1)
	Q := mat.NewDense(p, q, nil)
	signedPow(Q, Z, float64(m-1)) // Q = sign(Z) * |Z|^(m-1)
	Q.MulElem(Q, Z)               // Q = Z * (sign(Z)*|Z|^(m-1))

	// 3) U = (Zᵀ Z)^{-1} Zᵀ Q
	var ZtZ mat.Dense
	ZtZ.Mul(Z.T(), Z) // q×q
	// Use pseudoinverse for numerical stability if needed:
	// ZtZInv := Pinv(&ZtZ, math.Sqrt(math.SmallestNonzeroFloat64))
	var ZtZInv mat.Dense
	ZtZInv.Inverse(&ZtZ)

	var ZtQ mat.Dense
	ZtQ.Mul(Z.T(), Q) // q×q

	var U mat.Dense
	U.Mul(&ZtZInv, &ZtQ) // q×q

	// 4) Normalize columns so that diag(Uᵀ (Zᵀ Z) U) == 1
	//    H = Uᵀ S U, S=ZᵀZ
	var H mat.Dense
	H.Mul(U.T(), &ZtZ)
	H.Mul(&H, &U)
	D := mat.NewDense(q, q, nil)
	for j := 0; j < q; j++ {
		hjj := H.At(j, j)
		sc := 1.0
		if hjj > 0 && !math.IsNaN(hjj) && !math.IsInf(hjj, 0) {
			sc = 1.0 / math.Sqrt(hjj)
		}
		D.Set(j, j, sc)
	}
	UScaled := mat.NewDense(q, q, nil)
	UScaled.Mul(&U, D)

	// 5) R = T0 * UScaled;  L = A * R;  Phi = R^{-1} R^{-T}
	R = mat.NewDense(q, q, nil)
	R.Mul(T0, UScaled)

	L = mat.NewDense(p, q, nil)
	L.Mul(A, R)

	var Rinv mat.Dense
	Rinv.Inverse(R)
	Phi = mat.NewDense(q, q, nil)
	Phi.Mul(Rinv.T(), &Rinv) // R^{-T} * R^{-1} == (R^{-1} R^{-T})^T, symmetric

	return L, R, Phi, nil
}

// signedPow computes dst = sign(src) * |src|^k  (elementwise).
func signedPow(dst, src *mat.Dense, k float64) {
	r, c := src.Dims()
	dst.ReuseAs(r, c)
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			v := src.At(i, j)
			sign := 1.0
			if v < 0 {
				sign = -1.0
				v = -v
			}
			dst.Set(i, j, sign*math.Pow(v, k))
		}
	}
}
