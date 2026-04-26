// fa/kaiser_varimax.go
package fa

import (
	"math"

	"gonum.org/v1/gonum/mat"
)

// KaiserVarimaxWithRotationMatrix returns stats::varimax-style rotated loadings
// and the orthogonal rotation matrix.
func KaiserVarimaxWithRotationMatrix(A *mat.Dense, normalize bool, maxIter int, epsilon float64) (*mat.Dense, *mat.Dense, error) {
	p, q := A.Dims()
	if q < 2 {
		return mat.DenseCopyOf(A), identityMatrix(q), nil
	}
	if maxIter <= 0 {
		maxIter = 1000
	}
	if epsilon <= 0 {
		epsilon = 1e-5
	}

	x := mat.DenseCopyOf(A)
	scale := make([]float64, p)
	for i := range scale {
		scale[i] = 1
	}
	if normalize {
		for i := 0; i < p; i++ {
			sum := 0.0
			for j := 0; j < q; j++ {
				v := x.At(i, j)
				sum += v * v
			}
			scale[i] = math.Sqrt(sum)
			if scale[i] == 0 {
				scale[i] = math.Nextafter(0, 1)
			}
			for j := 0; j < q; j++ {
				x.Set(i, j, x.At(i, j)/scale[i])
			}
		}
	}

	rotmat := identityMatrix(q)
	d := 0.0
	for iter := 0; iter < maxIter; iter++ {
		z := mat.NewDense(p, q, nil)
		z.Mul(x, rotmat)

		colSumsZ2 := make([]float64, q)
		for j := 0; j < q; j++ {
			for i := 0; i < p; i++ {
				v := z.At(i, j)
				colSumsZ2[j] += v * v
			}
		}

		inner := mat.NewDense(p, q, nil)
		for i := 0; i < p; i++ {
			for j := 0; j < q; j++ {
				v := z.At(i, j)
				inner.Set(i, j, v*v*v-v*colSumsZ2[j]/float64(p))
			}
		}

		b := mat.NewDense(q, q, nil)
		b.Mul(x.T(), inner)

		var svd mat.SVD
		if !svd.Factorize(b, mat.SVDFull) {
			return nil, nil, nil
		}

		var u, v mat.Dense
		svd.UTo(&u)
		svd.VTo(&v)
		nextRot := mat.NewDense(q, q, nil)
		nextRot.Mul(&u, v.T())

		dPast := d
		d = 0
		for _, singular := range svd.Values(nil) {
			d += singular
		}
		rotmat = nextRot
		if d < dPast*(1+epsilon) {
			break
		}
	}

	z := mat.NewDense(p, q, nil)
	z.Mul(x, rotmat)
	if normalize {
		for i := 0; i < p; i++ {
			for j := 0; j < q; j++ {
				z.Set(i, j, z.At(i, j)*scale[i])
			}
		}
	}

	return z, rotmat, nil
}
