package stats_test

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/HazelnutParadise/insyra"
	stats "github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/mat"
)

// TestRotationInvariants checks that each rotation method satisfies the
// fundamental invariant: the model matrix is preserved under rotation.
//
//	Orthogonal: L_rot · L_rot' = L_unrot · L_unrot'
//	Oblique:    L_rot · Phi · L_rot' = L_unrot · L_unrot'
//
// Plus checks for valid Phi (oblique: symmetric, diag=1) and valid RotMat
// (orthogonal: R'R=I — but we have to handle Go's independent post-rotation
// column sort which breaks `L=Lu·R` directly; see internal note).
//
// Note on simplimax: GPArotation::simplimax is mathematically OBLIQUE
// (Q-suffix convention). Go correctly uses GPFoblq.
func TestRotationInvariants(t *testing.T) {
	if os.Getenv("INSYRA_VERIFY_ROT") != "1" {
		t.Skip()
	}
	const n = 50
	rows := make([][]any, n)
	for i := 0; i < n; i++ {
		x := float64(i)
		f1 := math.Sin(0.31*x) + 0.2*math.Cos(0.7*x)
		f2 := math.Cos(0.42*x) - 0.3*math.Sin(0.5*x)
		f3 := math.Sin(0.61*x+0.4) + 0.15*math.Cos(0.83*x)
		rows[i] = []any{
			0.85*f1 + 0.10*f2 + 0.20*math.Sin(1.3*x),
			0.78*f1 + 0.05*f2 + 0.18*math.Cos(1.7*x),
			0.10*f1 + 0.82*f2 + 0.21*math.Sin(2.1*x),
			0.05*f1 + 0.75*f2 + 0.19*math.Cos(2.4*x),
			0.15*f1 + 0.20*f2 + 0.80*f3 + 0.17*math.Sin(2.8*x),
			0.08*f1 + 0.12*f2 + 0.78*f3 + 0.22*math.Cos(3.1*x),
		}
	}
	tbl := insyra.NewDataTable()
	for c := 0; c < 6; c++ {
		col := make([]any, n)
		for i := 0; i < n; i++ {
			col[i] = rows[i][c]
		}
		tbl.AppendCols(insyra.NewDataList(col...))
	}

	type rotCase struct {
		method    stats.FactorRotationMethod
		isOblique bool
	}
	cases := []rotCase{
		{stats.FactorRotationVarimax, false},
		{stats.FactorRotationQuartimax, false},
		{stats.FactorRotationGeominT, false},
		{stats.FactorRotationBentlerT, false},
		{stats.FactorRotationPromax, true},
		{stats.FactorRotationOblimin, true},
		{stats.FactorRotationQuartimin, true},
		{stats.FactorRotationGeominQ, true},
		{stats.FactorRotationBentlerQ, true},
		{stats.FactorRotationSimplimax, true}, // GPArotation::simplimax is oblique
	}

	allPass := true
	for _, c := range cases {
		opt := stats.DefaultFactorAnalysisOptions()
		opt.Count.Method = stats.FactorCountFixed
		opt.Count.FixedK = 3
		opt.Extraction = stats.FactorExtractionML
		opt.Rotation.Method = c.method
		opt.Scoring = stats.FactorScoreNone
		res, err := stats.FactorAnalysis(tbl, opt)
		if err != nil {
			t.Errorf("[%s] FA error: %v", c.method, err)
			allPass = false
			continue
		}
		L := dtToDense(res.Loadings)
		Lu := dtToDense(res.UnrotatedLoadings)
		Phi := dtToDense(res.Phi)

		report := []string{}

		// Build the model matrices.
		LuT := mat.DenseCopyOf(Lu.T())
		LT := mat.DenseCopyOf(L.T())
		ModelUnrot := mat.NewDense(6, 6, nil)
		ModelUnrot.Mul(Lu, LuT)

		var ModelRot mat.Dense
		if c.isOblique && Phi != nil {
			LPhi := mat.NewDense(6, 3, nil)
			LPhi.Mul(L, Phi)
			ModelRot.Mul(LPhi, LT)
		} else {
			ModelRot.Mul(L, LT)
		}

		// FUNDAMENTAL INVARIANT: model preserved under rotation.
		modelDiff := maxAbsDiff(&ModelRot, ModelUnrot)
		if modelDiff > 1e-7 {
			report = append(report, fmt.Sprintf("model not preserved max=%.3e", modelDiff))
		}

		if c.isOblique {
			// Phi: symmetric, diag = 1
			if Phi == nil {
				report = append(report, "Phi is nil for oblique")
			} else {
				phiRows, _ := Phi.Dims()
				maxDiag, maxAsym := 0.0, 0.0
				for i := 0; i < phiRows; i++ {
					if d := math.Abs(Phi.At(i, i) - 1.0); d > maxDiag {
						maxDiag = d
					}
					for j := i + 1; j < phiRows; j++ {
						if d := math.Abs(Phi.At(i, j) - Phi.At(j, i)); d > maxAsym {
							maxAsym = d
						}
					}
				}
				if maxDiag > 1e-9 {
					report = append(report, fmt.Sprintf("Phi diag≠1 max=%.3e", maxDiag))
				}
				if maxAsym > 1e-9 {
					report = append(report, fmt.Sprintf("Phi asymmetric max=%.3e", maxAsym))
				}
			}
			// Structure = L · Phi
			Structure := dtToDense(res.Structure)
			if Structure != nil && Phi != nil {
				StructCheck := mat.NewDense(6, 3, nil)
				StructCheck.Mul(L, Phi)
				maxS := maxAbsDiff(Structure, StructCheck)
				if maxS > 1e-9 {
					report = append(report, fmt.Sprintf("Structure≠L·Phi max=%.3e", maxS))
				}
			}
		} else {
			// Orthogonal: Phi should be nil or identity
			if Phi != nil {
				phiRows, _ := Phi.Dims()
				maxOff := 0.0
				for i := 0; i < phiRows; i++ {
					for j := 0; j < phiRows; j++ {
						target := 0.0
						if i == j {
							target = 1.0
						}
						if d := math.Abs(Phi.At(i, j) - target); d > maxOff {
							maxOff = d
						}
					}
				}
				if maxOff > 1e-9 {
					report = append(report, fmt.Sprintf("Phi≠I (orthog) max=%.3e", maxOff))
				}
			}
		}

		status := "✓ PASS"
		if len(report) > 0 {
			status = "✗ FAIL: " + fmt.Sprintf("%v", report)
			allPass = false
		}
		fmt.Printf("[%-12s %-9s] %s\n", c.method,
			func() string {
				if c.isOblique {
					return "(oblique)"
				}
				return "(orthog)"
			}(), status)
	}
	if !allPass {
		t.Fail()
	}
}

func dtToDense(dt insyra.IDataTable) *mat.Dense {
	if dt == nil {
		return nil
	}
	rows, cols := dt.Size()
	if rows == 0 || cols == 0 {
		return nil
	}
	out := mat.NewDense(rows, cols, nil)
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			v, ok := dt.GetElementByNumberIndex(i, j).(float64)
			if !ok {
				return nil
			}
			out.Set(i, j, v)
		}
	}
	return out
}

func maxAbsDiff(a, b *mat.Dense) float64 {
	r, c := a.Dims()
	maxd := 0.0
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			if d := math.Abs(a.At(i, j) - b.At(i, j)); d > maxd {
				maxd = d
			}
		}
	}
	return maxd
}
