package fa

import (
	"math/rand"
	"testing"

	"gonum.org/v1/gonum/mat"
)

var orthogonalMethods = []string{"varimax", "quartimax", "bentlerT", "geominT"}

var obliqueMethods = []string{"quartimin", "oblimin", "bentlerQ", "geominQ", "simplimax", "promax"}

// simpleStructure is the clean pattern the gradient tests use.
func simpleStructure() *mat.Dense { return loadings() }

// noisyStructure is a noisier pattern, generated once from a fixed seed and
// pinned here so the test does not depend on the generator.
func noisyStructure() *mat.Dense {
	rnd := rand.New(rand.NewSource(20260912))
	p, nf := 6, 3
	d := make([]float64, p*nf)
	for i := range d {
		d[i] = rnd.NormFloat64() * 0.5
	}
	L := mat.NewDense(p, nf, d)
	for i := 0; i < p; i++ {
		L.Set(i, i%nf, L.At(i, i%nf)+0.8)
	}
	return L
}

// Every rotation wrapper has to hand its convergence flag back, or fa.Rotate
// reports a convergence it never checked.
func TestEveryRotationReportsConvergence(t *testing.T) {
	L := simpleStructure()
	for _, method := range append(append([]string{}, orthogonalMethods...), obliqueMethods...) {
		res := FaRotations(mat.DenseCopyOf(L), nil, method, 0, 5, 4, 0.01, 1e-5, 1000)
		m, ok := res.(map[string]any)
		if !ok {
			t.Fatalf("%s: FaRotations returned %T", method, res)
		}
		if _, ok := m["convergence"].(bool); !ok {
			t.Errorf("%s: no convergence flag in the chosen candidate", method)
		}
	}
}

// A rotation that runs out of iterations has to say so. fa.Rotate used to
// report converged = true unconditionally, because the candidate the restart
// loop chose never carried the flag, so the default in Rotate was all anyone
// ever saw.
//
// This lives here rather than in the stats package because opt.MaxIter governs
// extraction and is deliberately not passed to the rotation
// (stats/factor_analysis.go, rotateFactors), so there is no public call that
// can cripple a rotation.
func TestRotateReportsThatItDidNotConverge(t *testing.T) {
	L := noisyStructure()
	for _, method := range append(append([]string{}, orthogonalMethods...), obliqueMethods...) {
		if method == "promax" {
			continue // Promax is a closed-form target rotation, not an iteration.
		}
		_, _, _, converged, err := Rotate(mat.DenseCopyOf(L), method,
			&RotOpts{Eps: 1e-12, MaxIter: 1, PromaxPower: 4, Restarts: 1})
		if err != nil {
			t.Errorf("%s: %v", method, err)
			continue
		}
		if converged {
			t.Errorf("%s: reported converged after a single iteration at eps 1e-12", method)
		}
	}
}

func TestRotateReportsThatItConverged(t *testing.T) {
	L := simpleStructure()
	for _, method := range append(append([]string{}, orthogonalMethods...), obliqueMethods...) {
		_, _, _, converged, err := Rotate(mat.DenseCopyOf(L), method,
			&RotOpts{Eps: 1e-5, MaxIter: 1000, PromaxPower: 4, Restarts: 5})
		if err != nil {
			t.Errorf("%s: %v", method, err)
			continue
		}
		if !converged {
			t.Errorf("%s: reported not converged on a clean simple structure", method)
		}
	}
}
