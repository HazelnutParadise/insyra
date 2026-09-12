package fa

import (
	"math"
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func maxAbsRTRI(R *mat.Dense) float64 {
	n, _ := R.Dims()
	RT := mat.DenseCopyOf(R.T())
	var RTR mat.Dense
	RTR.Mul(RT, R)
	maxd := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			want := 0.0
			if i == j {
				want = 1
			}
			if d := math.Abs(RTR.At(i, j) - want); d > maxd {
				maxd = d
			}
		}
	}
	return maxd
}

func maxCommunalityChange(Lu, L *mat.Dense) float64 {
	p, k := Lu.Dims()
	maxd := 0.0
	for i := 0; i < p; i++ {
		hu, h := 0.0, 0.0
		for j := 0; j < k; j++ {
			hu += Lu.At(i, j) * Lu.At(i, j)
			h += L.At(i, j) * L.At(i, j)
		}
		if d := math.Abs(h - hu); d > maxd {
			maxd = d
		}
	}
	return maxd
}

// blockyLoadings builds a 6×3 loading pattern similar to issue #373's synthetic case.
func blockyLoadings(seed int64) *mat.Dense {
	rnd := rand.New(rand.NewSource(seed))
	L := mat.NewDense(6, 3, nil)
	for i := 0; i < 6; i++ {
		for j := 0; j < 3; j++ {
			v := 0.1 * rnd.NormFloat64()
			if i/2 == j {
				v += 0.7
			}
			L.Set(i, j, v)
		}
	}
	return L
}

func TestFaRotations_OrthogonalRestartsKeepOrthogonality(t *testing.T) {
	L := blockyLoadings(42)
	for _, method := range []string{"quartimax", "varimax", "geomint", "bentlert"} {
		for _, restarts := range []int{1, 2, 5, 20} {
			rotL, R, _, _, err := Rotate(L, method, &RotOpts{Restarts: restarts, Eps: 1e-5, MaxIter: 1000})
			if err != nil {
				t.Errorf("[%s restarts=%d] %v", method, restarts, err)
				continue
			}
			orth := maxAbsRTRI(R)
			h2 := maxCommunalityChange(L, rotL)
			t.Logf("[%s restarts=%d] max|R'R-I|=%.3e max|h2Δ|=%.3e", method, restarts, orth, h2)
			if orth > 1e-8 {
				t.Errorf("[%s restarts=%d] rotmat not orthogonal: max|R'R-I|=%.3e", method, restarts, orth)
			}
			if h2 > 1e-8 {
				t.Errorf("[%s restarts=%d] communalities changed under orthogonal rotation: max|h2Δ|=%.3e", method, restarts, h2)
			}
		}
	}
}
