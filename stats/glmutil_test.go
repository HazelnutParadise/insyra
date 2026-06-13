package stats

import (
	"math"
	"testing"
)

func TestComputeGLMInferenceAndCI(t *testing.T) {
	beta := []float64{1, -2}
	cov := [][]float64{{4, 0}, {0, 9}}
	se, z, p := computeGLMInference(beta, cov, 0.25)
	if se[0] != 1 || se[1] != 1.5 {
		t.Fatalf("unexpected se: %v", se)
	}
	if math.Abs(z[0]-1) > 1e-12 || math.Abs(z[1]+1.3333333333333333) > 1e-12 {
		t.Fatalf("unexpected z: %v", z)
	}
	if p[0] <= 0 || p[0] >= 1 || p[1] <= 0 || p[1] >= 1 {
		t.Fatalf("unexpected p: %v", p)
	}

	cis := buildGLMCoeffCIs(beta, se, 0.95)
	if math.Abs(cis[0][0]-(-0.959963984540054)) > 1e-12 {
		t.Fatalf("unexpected CI: %v", cis)
	}
}

func TestPearsonDispersion(t *testing.T) {
	y := []float64{1, 2, 3}
	mu := []float64{1.1, 1.9, 3.1}
	w := []float64{1, 1, 1}
	chi := pearsonChiSq(y, mu, w, gaussianFamily{})
	if math.Abs(chi-0.03) > 1e-12 {
		t.Fatalf("pearson chi-square = %.15g", chi)
	}
	if got := pearsonDispersion(chi, 1); math.Abs(got-0.03) > 1e-12 {
		t.Fatalf("pearson dispersion = %.15g", got)
	}
}
