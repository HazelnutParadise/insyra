package fa

import (
	"fmt"
	"math"
	"testing"

	statslinalg "github.com/HazelnutParadise/insyra/stats/internal/linalg"
	"gonum.org/v1/gonum/mat"
)

// TestMLDivergeOnModerateThreeFactor reproduces the failing case
// generated_moderate_three_factor / ml / none.  Runs ML extraction
// twice — once with our dsyevr wrapper, once with gonum's dsyev (via
// statslinalg) — and dumps both to compare against R's expected
// unrotated_loadings[0,1] = 0.511567642393511.
func TestMLDivergeOnModerateThreeFactor(t *testing.T) {
	const n = 32
	const p = 7
	const nfactors = 3

	rows := make([][]float64, n)
	for i := 0; i < n; i++ {
		x := float64(i)
		f1 := 1.20*math.Sin(0.39*x) + 0.32*float64(i%5-2)
		f2 := 1.05*math.Cos(0.53*x+0.4) - 0.24*float64(i%7-3)
		f3 := 0.88*math.Sin(0.71*x-0.2) + 0.42*math.Cos(0.17*x)
		n1 := 0.42 * math.Sin(1.37*x+0.2)
		n2 := 0.39 * math.Cos(1.11*x-0.4)
		n3 := 0.36 * math.Sin(1.83*x+0.7)
		rows[i] = []float64{
			1.0 + 0.78*f1 + 0.18*f2 + n1,
			-0.7 + 0.72*f1 - 0.12*f3 + n2,
			0.4 + 0.20*f1 + 0.70*f2 + n3,
			-1.2 + 0.75*f2 + 0.14*f3 - n1,
			1.8 + 0.16*f1 + 0.76*f3 - n2,
			-2.1 - 0.18*f2 + 0.68*f3 + n3,
			0.2 + 0.34*f1 + 0.31*f2 + 0.29*f3 + 0.33*math.Cos(2.07*x),
		}
	}

	// Compute correlation matrix.
	mean := make([]float64, p)
	for j := 0; j < p; j++ {
		s := 0.0
		for i := 0; i < n; i++ {
			s += rows[i][j]
		}
		mean[j] = s / float64(n)
	}
	sd := make([]float64, p)
	for j := 0; j < p; j++ {
		s := 0.0
		for i := 0; i < n; i++ {
			d := rows[i][j] - mean[j]
			s += d * d
		}
		sd[j] = math.Sqrt(s / float64(n-1))
	}
	cor := mat.NewDense(p, p, nil)
	for i := 0; i < p; i++ {
		for j := 0; j < p; j++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += (rows[k][i] - mean[i]) * (rows[k][j] - mean[j])
			}
			cor.Set(i, j, s/(float64(n-1)*sd[i]*sd[j]))
		}
	}

	// First, sanity check: dsyevr eigenvalues vs gonum on the correlation matrix.
	valDsyevr, _, ok1 := symmetricEigenDescendingDsyevr(cor)
	valGonum, _, ok2 := statslinalg.SymmetricEigenDescending(cor)
	if !ok1 || !ok2 {
		t.Fatal("eigendecomposition failed")
	}
	t.Logf("dsyevr eigenvalues: %v", valDsyevr)
	t.Logf("gonum  eigenvalues: %v", valGonum)
	maxEvDiff := 0.0
	for i := 0; i < p; i++ {
		d := math.Abs(valDsyevr[i] - valGonum[i])
		if d > maxEvDiff {
			maxEvDiff = d
		}
	}
	t.Logf("eigenvalue max diff = %g", maxEvDiff)

	// Now run ML extraction with both eigendecompositions.
	rMat := mat.NewDense(p, p, nil)
	rMat.CloneFrom(cor)

	for _, label := range []string{"dsyevr", "gonum"} {
		// Set diagonal to SMC for the input.
		smcRes, _ := Smc(cor, &SmcOptions{Covar: false})
		smcVec := make([]float64, p)
		for i := 0; i < p; i++ {
			smcVec[i] = smcRes.AtVec(i)
		}
		rWork := mat.NewDense(p, p, nil)
		rWork.CloneFrom(cor)
		for i := 0; i < p; i++ {
			rWork.Set(i, i, smcVec[i])
		}

		// Override the eigendecomposition path by swapping wrapper.
		// We'll just use the global one but record results from each.
		var loadings *mat.Dense
		var err error
		if label == "dsyevr" {
			loadings, _, _, err = maximumLikelihoodFactoring(cor, rWork, nfactors, false, 0.001, 50, 1e7, 100)
		} else {
			// Quick branch: run with statslinalg by temporarily swapping
			// — easier to just expect drift here since we wired everything
			// to dsyevr.  Skip the gonum path for now.
			continue
		}
		if err != nil {
			t.Errorf("%s ML failed: %v", label, err)
			continue
		}
		fmt.Printf("\n=== %s loadings ===\n", label)
		for i := 0; i < p; i++ {
			row := []float64{}
			for j := 0; j < nfactors; j++ {
				row = append(row, loadings.At(i, j))
			}
			fmt.Printf("  row %d: %v\n", i, row)
		}
		t.Logf("%s loadings[0,1] = %g  (R expects 0.511567642393511)", label, loadings.At(0, 1))
	}
}
