// fa/GPArotation_vgQ_infomax.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// vgQInfomax computes the objective and gradient for infomax rotation.
// Mirrors GPArotation::vgQ.infomax(L)
//
// Returns: Gq (gradient), f (objective), method
func vgQInfomax(L *mat.Dense) (Gq *mat.Dense, f float64, method string) {
	p, k := L.Dims()

	// S = L^2
	S := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			l := L.At(i, j)
			S.Set(i, j, l*l)
		}
	}

	// s = sum(S)
	s := 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			s += S.At(i, j)
		}
	}

	// s1 = rowSums(S)
	s1 := make([]float64, p)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			s1[i] += S.At(i, j)
		}
	}

	// s2 = colSums(S)
	s2 := make([]float64, k)
	for j := 0; j < k; j++ {
		for i := 0; i < p; i++ {
			s2[j] += S.At(i, j)
		}
	}

	// E = S / s
	E := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			E.Set(i, j, S.At(i, j)/s)
		}
	}

	// e1 = s1 / s
	e1 := make([]float64, p)
	for i := 0; i < p; i++ {
		e1[i] = s1[i] / s
	}

	// e2 = s2 / s
	e2 := make([]float64, k)
	for j := 0; j < k; j++ {
		e2[j] = s2[j] / s
	}

	// Q0 = sum(-E * log(E))
	Q0 := 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			e := E.At(i, j)
			if e > 0 {
				Q0 -= e * math.Log(e)
			}
		}
	}

	// Q1 = sum(-e1 * log(e1))
	Q1 := 0.0
	for i := 0; i < p; i++ {
		if e1[i] > 0 {
			Q1 -= e1[i] * math.Log(e1[i])
		}
	}

	// Q2 = sum(-e2 * log(e2))
	Q2 := 0.0
	for j := 0; j < k; j++ {
		if e2[j] > 0 {
			Q2 -= e2[j] * math.Log(e2[j])
		}
	}

	// f = log(k) + Q0 - Q1 - Q2
	f = math.Log(float64(k)) + Q0 - Q1 - Q2

	// H = -(log(E) + 1)
	H := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			e := E.At(i, j)
			if e > 0 {
				H.Set(i, j, -(math.Log(e) + 1))
			} else {
				H.Set(i, j, 0) // or handle
			}
		}
	}

	// alpha = sum(S * H) / s^2
	alpha := 0.0
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			alpha += S.At(i, j) * H.At(i, j)
		}
	}
	alpha /= s * s

	// G0 = H/s - alpha * ones
	G0 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			G0.Set(i, j, H.At(i, j)/s-alpha)
		}
	}

	// h1 = -(log(e1) + 1)
	h1 := make([]float64, p)
	for i := 0; i < p; i++ {
		if e1[i] > 0 {
			h1[i] = -(math.Log(e1[i]) + 1)
		}
	}

	// alpha1 = sum(s1 * h1) / s^2
	alpha1 := 0.0
	for i := 0; i < p; i++ {
		alpha1 += s1[i] * h1[i]
	}
	alpha1 /= s * s

	// G1 = rep(h1, k) / s - alpha1 * ones
	G1 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			G1.Set(i, j, h1[i]/s-alpha1)
		}
	}

	// h2 = -(log(e2) + 1)
	h2 := make([]float64, k)
	for j := 0; j < k; j++ {
		if e2[j] > 0 {
			h2[j] = -(math.Log(e2[j]) + 1)
		}
	}

	// alpha2 = sum(h2 * s2) / s^2
	alpha2 := 0.0
	for j := 0; j < k; j++ {
		alpha2 += h2[j] * s2[j]
	}
	alpha2 /= s * s

	// G2 = rep(h2, p) / s - alpha2 * ones
	G2 := mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			G2.Set(i, j, h2[j]/s-alpha2)
		}
	}

	// Gq = 2 * L * (G0 - G1 - G2)
	Gq = mat.NewDense(p, k, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < k; j++ {
			diff := G0.At(i, j) - G1.At(i, j) - G2.At(i, j)
			Gq.Set(i, j, 2*L.At(i, j)*diff)
		}
	}

	method = "Infomax"
	return
}
