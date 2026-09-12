package fa

import (
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
)

// psych 2.6.5 changed Promax's first step from stats::varimax to
// GPArotation::Varimax ("replaced with GPArotation Varimax 5/9/26" in
// psych/R/Promax.R). On loadings whose varimax criterion is nearly flat the
// two stop at different angles, and Promax raises that difference to the
// fourth power: on the parity suite's ten-row table our loading[0,0] was
// 0.700 against psych's 0.625. The fixture is that table's Kaiser-weighted
// MINRES loadings, identical to psych's to ten digits, and the expected
// values are psych 2.6.5's Promax(weighted, m = 4).
func TestPromaxMatchesPsych(t *testing.T) {
	weighted := mat.NewDense(6, 2, []float64{
		0.9993373586, 0.0363984021,
		0.9999280442, -0.0119961008,
		0.9998527931, 0.0171578571,
		-0.9999867509, -0.0051476227,
		-0.9979025685, 0.0647337919,
		-0.9998420695, -0.0177717788,
	})
	wantL := mat.NewDense(6, 2, []float64{
		0.6246570040, -0.4050701569,
		0.5263241854, -0.5048720363,
		0.5857166845, -0.4448838471,
		-0.5613061229, 0.4696459636,
		-0.4176962082, 0.6123488462,
		-0.5869623332, 0.4436162316,
	})
	const wantPhi = -0.8807689857

	res := Promax(weighted, 4, false)
	if msg, _ := res["error"].(string); msg != "" {
		t.Fatal(msg)
	}
	L := res["loadings"].(*mat.Dense)
	Phi := res["Phi"].(*mat.Dense)

	// Up to a sign flip of both factors (the same solution).
	best := math.Inf(1)
	for _, s := range []float64{1, -1} {
		d := 0.0
		for i := 0; i < 6; i++ {
			for j := 0; j < 2; j++ {
				d = math.Max(d, math.Abs(s*L.At(i, j)-wantL.At(i, j)))
			}
		}
		best = math.Min(best, d)
	}
	const tol = 5e-4
	if best > tol {
		t.Errorf("Promax loadings differ from psych 2.6.5 by %.3e, want <= %.0e:\n%v", best, tol, mat.Formatted(L))
	}
	if d := math.Abs(Phi.At(0, 1) - wantPhi); d > tol {
		t.Errorf("Promax Phi[0,1] = %.10f, psych 2.6.5 has %.10f (off by %.3e)", Phi.At(0, 1), wantPhi, d)
	}
}
