// fa/GPArotation_vgQ_mccammon.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQMcCammon computes the objective and gradient for McCammon entropy rotation.
// Mirrors GPArotation::vgQ.mccammon(L)
//
// Returns: Gq (gradient), f (objective), method
func vgQMcCammon(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// S = L^2
	S := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			S.Set(i, j, l*l)
		}
	}

	// s2 = colSums(S)
	s2 := make([]float64, k)
	for j := 0; j < k; j++ {
		for i := 0; i < p; i++ {
			s2[j] += S.At(i, j)
		}
	}

	// P = S / rep(s2, p)
	P := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			P.Set(i, j, S.At(i, j)/s2[j])
		}
	}

	// Q1 = -sum(P * log(P))
	Q1 := 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			pij := P.At(i, j)
			if pij > 0 {
				Q1 -= pij * math.Log(pij)
			}
		}
	}

	// H = -(log(P) + 1)
	H := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			pij := P.At(i, j)
			if pij > 0 {
				H.Set(i, j, -(math.Log(pij) + 1))
			} else {
				H.Set(i, j, 0)
			}
		}
	}

	// R = matrix(rep(s2, p), ncol = k, byrow = T) = each row is s2
	R := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			R.Set(i, j, s2[j])
		}
	}

	// S_H_R2 = S * H / R^2
	S_H_R2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			r := R.At(i, j)
			S_H_R2.Set(i, j, S.At(i, j)*H.At(i, j)/(r*r))
		}
	}

	// M %*% S_H_R2, where M is ones(p,p), so each row sum of columns
	M_S_H_R2 := mat.NewDense(p, k, nil)
	for j := 0; j < k; j++ {
		colSum := 0.0
		for i := 0; i < p; i++ {
			colSum += S_H_R2.At(i, j)
		}
		for i := 0; i < p; i++ {
			M_S_H_R2.Set(i, j, colSum)
		}
	}

	// G1 = H / R - M_S_H_R2
	G1 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			G1.Set(i, j, H.At(i, j)/R.At(i, j)-M_S_H_R2.At(i, j))
		}
	}

	// s = sum(S)
	s := 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			s += S.At(i, j)
		}
	}

	// p2 = s2 / s
	p2 := make([]float64, k)
	for j := 0; j < k; j++ {
		p2[j] = s2[j] / s
	}

	// Q2 = -sum(p2 * log(p2))
	Q2 := 0.0
	for j := 0; j < k; j++ {
		if p2[j] > 0 {
			Q2 -= p2[j] * math.Log(p2[j])
		}
	}

	// h = -(log(p2) + 1)
	h := make([]float64, k)
	for j := 0; j < k; j++ {
		if p2[j] > 0 {
			h[j] = -(math.Log(p2[j]) + 1)
		}
	}

	// alpha = dot(h, p2)
	alpha := 0.0
	for j := 0; j < k; j++ {
		alpha += h[j] * p2[j]
	}

	// G2 = rep(1,p) %*% t(h) / s - alpha * ones
	G2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			G2.Set(i, j, h[j]/s-alpha)
		}
	}

	// Gq = 2 * L * (G1 / Q1 - G2 / Q2)
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			diff := G1.At(i, j)/Q1 - G2.At(i, j)/Q2
			Gq.Set(i, j, 2*L.At(i, j)*diff)
		}
	}

	f = math.Log(Q1) - math.Log(Q2)

	method = "McCammon entropy"
	return
}
