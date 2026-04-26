package stats_test

import (
	"math"
	"os"
	"os/exec"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

const factorParityTol = 2e-5

func requireFactorAnalysisRTools(t *testing.T) {
	t.Helper()
	if os.Getenv("INSYRA_STRICT_FACTOR_R_PARITY") != "1" {
		t.Skip("set INSYRA_STRICT_FACTOR_R_PARITY=1 to run strict R factor-analysis parity tests")
	}
	if _, err := exec.LookPath("Rscript"); err != nil {
		t.Skipf("Rscript not found: %v", err)
	}
	checkR := exec.Command("Rscript", "-e", "pkgs <- c('psych','GPArotation','jsonlite'); ok <- all(sapply(pkgs, function(p) requireNamespace(p, quietly=TRUE))); if (!ok) quit(status=1)")
	if out, err := checkR.CombinedOutput(); err != nil {
		t.Skipf("R factor-analysis stack unavailable: %v, out=%s", err, string(out))
	}
}

func factorAnalysisRows() [][]float64 {
	return [][]float64{
		{1.0, 1.1, 0.9, 5.0, 5.2, 4.8},
		{1.2, 1.0, 1.1, 4.8, 5.0, 4.9},
		{0.8, 0.9, 1.0, 5.3, 5.1, 5.2},
		{4.9, 5.1, 5.0, 1.1, 1.0, 1.2},
		{5.2, 4.8, 5.1, 0.9, 1.2, 1.0},
		{5.0, 5.2, 4.9, 1.0, 0.8, 1.1},
		{2.6, 2.7, 2.5, 3.2, 3.1, 3.3},
		{3.1, 3.0, 3.2, 2.6, 2.5, 2.7},
		{1.5, 1.4, 1.6, 4.6, 4.4, 4.5},
		{4.5, 4.6, 4.4, 1.5, 1.6, 1.4},
	}
}

type factorAnalysisDataset struct {
	name     string
	rows     [][]any
	nFactors int
}

func factorAnalysisRowsAny() [][]any {
	return floatRowsToAny(factorAnalysisRows())
}

func generatedOneFactorRows() [][]any {
	rows := make([][]any, 18)
	for i := range rows {
		x := float64(i)
		f := -2.2 + 0.27*x + 0.35*math.Sin(0.47*x)
		rows[i] = []any{
			1.0 + 0.92*f + 0.22*math.Sin(1.31*x),
			-0.5 + 0.84*f + 0.25*math.Cos(1.07*x),
			0.7 + 1.05*f + 0.20*math.Sin(0.83*x+0.4),
			2.0 + 0.76*f + 0.24*math.Cos(1.53*x+0.2),
		}
	}
	return rows
}

func factorAnalysisDatasets() []factorAnalysisDataset {
	return []factorAnalysisDataset{
		{name: "two_blocks", rows: factorAnalysisRowsAny(), nFactors: 2},
		{name: "one_factor", rows: generatedOneFactorRows(), nFactors: 1},
		{name: "three_blocks", rows: [][]any{
			{5.0, 4.8, 1.1, 1.2, 2.5, 2.7},
			{4.7, 5.1, 1.3, 1.0, 2.8, 2.6},
			{5.2, 4.9, 1.0, 1.4, 2.4, 2.9},
			{1.2, 1.0, 5.1, 4.9, 2.7, 2.5},
			{1.4, 1.3, 4.8, 5.2, 2.6, 2.8},
			{1.0, 1.1, 5.3, 5.0, 2.9, 2.4},
			{2.7, 2.5, 1.2, 1.4, 5.1, 4.9},
			{2.5, 2.8, 1.1, 1.0, 4.8, 5.2},
			{2.9, 2.6, 1.3, 1.2, 5.3, 5.0},
			{4.4, 4.2, 2.0, 2.1, 3.3, 3.5},
			{2.1, 2.0, 4.3, 4.5, 3.4, 3.2},
			{3.2, 3.4, 2.2, 2.0, 4.4, 4.6},
		}, nFactors: 3},
		{name: "cross_loading", rows: [][]any{
			{1.0, 1.2, 4.9, 5.1, 3.0},
			{1.3, 1.1, 4.7, 5.0, 3.2},
			{0.9, 1.0, 5.2, 4.8, 3.1},
			{4.8, 5.0, 1.2, 1.1, 3.5},
			{5.1, 4.9, 1.0, 1.3, 3.4},
			{4.9, 5.2, 1.3, 0.9, 3.6},
			{2.4, 2.6, 3.2, 3.3, 2.9},
			{3.2, 3.1, 2.5, 2.7, 3.3},
			{1.5, 1.7, 4.4, 4.6, 3.0},
			{4.5, 4.3, 1.6, 1.4, 3.7},
			{2.0, 2.1, 3.8, 3.7, 3.2},
			{3.7, 3.9, 2.0, 2.2, 3.5},
		}, nFactors: 2},
		{name: "missing_rows", rows: [][]any{
			{1.0, 1.1, 0.9, 5.0, 5.2, 4.8},
			{1.2, 1.0, 1.1, 4.8, 5.0, 4.9},
			{0.8, 0.9, 1.0, 5.3, 5.1, 5.2},
			{4.9, 5.1, 5.0, 1.1, 1.0, 1.2},
			{5.2, 4.8, 5.1, 0.9, 1.2, 1.0},
			{5.0, 5.2, 4.9, 1.0, 0.8, 1.1},
			{2.6, 2.7, 2.5, 3.2, 3.1, 3.3},
			{3.1, 3.0, 3.2, 2.6, 2.5, 2.7},
			{1.5, 1.4, 1.6, 4.6, 4.4, 4.5},
			{4.5, 4.6, 4.4, 1.5, 1.6, 1.4},
			{2.0, 2.2, 2.1, nil, 3.8, 3.9},
			{3.8, 3.7, 3.9, 2.1, 2.0, 2.2},
		}, nFactors: 2},
	}
}

func generatedFactorAnalysisRows(n int, variant int) [][]any {
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := float64(i%7) - 3.0 + 0.18*float64(i/7)
		f2 := 1.15*math.Sin(0.73*x) + 0.35*math.Cos(0.29*x)
		f3 := 0.75*math.Cos(0.41*x) - 0.20*float64(i%5)
		j1 := 0.55 * math.Sin(1.37*x+float64(variant))
		j2 := 0.50 * math.Cos(0.91*x+float64(variant))
		shift := float64(variant-1) * 1.7
		scale := 1.0 + 0.35*float64(variant)
		rows[i] = []any{
			shift + scale*(0.72*f1+0.18*f2+j1+0.55*math.Sin(2.11*x)),
			-0.6 + scale*(0.64*f1-0.13*f2+0.08*f3+j2+0.58*math.Cos(1.73*x)),
			1.3 + scale*(0.15*f1+0.70*f2-0.06*f3-j1+0.56*math.Sin(1.19*x)),
			-1.1 + scale*(-0.10*f1+0.66*f2+0.12*f3+j2+0.54*math.Cos(2.37*x)),
			2.4 + scale*(0.38*f1+0.34*f2+0.28*f3+j1-j2+0.52*math.Sin(2.71*x)),
			-2.0 + scale*(-0.32*f1+0.22*f2+0.58*f3-j1+0.56*math.Cos(1.53*x)),
		}
	}
	return rows
}

func generatedModerateThreeFactorRows() [][]any {
	const n = 32
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := 1.20*math.Sin(0.39*x) + 0.32*float64(i%5-2)
		f2 := 1.05*math.Cos(0.53*x+0.4) - 0.24*float64(i%7-3)
		f3 := 0.88*math.Sin(0.71*x-0.2) + 0.42*math.Cos(0.17*x)
		n1 := 0.42 * math.Sin(1.37*x+0.2)
		n2 := 0.39 * math.Cos(1.11*x-0.4)
		n3 := 0.36 * math.Sin(1.83*x+0.7)
		rows[i] = []any{
			1.0 + 0.78*f1 + 0.18*f2 + n1,
			-0.7 + 0.72*f1 - 0.12*f3 + n2,
			0.4 + 0.20*f1 + 0.70*f2 + n3,
			-1.2 + 0.75*f2 + 0.14*f3 - n1,
			1.8 + 0.16*f1 + 0.76*f3 - n2,
			-2.1 - 0.18*f2 + 0.68*f3 + n3,
			0.2 + 0.34*f1 + 0.31*f2 + 0.29*f3 + 0.33*math.Cos(2.07*x),
		}
	}
	return rows
}

func generatedWeakTwoFactorRows() [][]any {
	const n = 34
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := 0.95*math.Sin(0.31*x) + 0.18*float64(i%6-2)
		f2 := 0.90*math.Cos(0.47*x+0.3) - 0.14*float64(i%5-2)
		rows[i] = []any{
			0.2 + 0.54*f1 + 0.10*f2 + 0.64*math.Sin(1.19*x),
			-0.4 + 0.49*f1 - 0.08*f2 + 0.61*math.Cos(1.43*x+0.2),
			1.1 + 0.12*f1 + 0.52*f2 + 0.66*math.Sin(1.71*x-0.3),
			-1.3 - 0.11*f1 + 0.47*f2 + 0.63*math.Cos(1.97*x),
			0.7 + 0.32*f1 + 0.28*f2 + 0.67*math.Sin(2.23*x+0.5),
		}
	}
	return rows
}

func generatedHighCorrelationRows() [][]any {
	const n = 28
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f := 1.10*math.Sin(0.27*x) + 0.72*math.Cos(0.13*x) + 0.10*float64(i%4-1)
		rows[i] = []any{
			1.0 + 0.96*f + 0.11*math.Sin(1.07*x),
			-0.5 + 0.93*f + 0.10*math.Cos(1.29*x+0.1),
			0.8 + 0.98*f + 0.09*math.Sin(1.51*x-0.2),
			-1.4 + 0.91*f + 0.12*math.Cos(1.73*x+0.4),
			0.1 + 0.88*f + 0.13*math.Sin(1.91*x),
		}
	}
	return rows
}

func generatedMixedSignRows() [][]any {
	const n = 30
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := 1.15*math.Sin(0.35*x) + 0.22*float64(i%7-3)
		f2 := 1.00*math.Cos(0.49*x-0.2) - 0.18*float64(i%6-2)
		rows[i] = []any{
			0.3 + 0.80*f1 + 0.15*f2 + 0.38*math.Sin(1.23*x),
			-0.8 - 0.74*f1 + 0.18*f2 + 0.36*math.Cos(1.41*x),
			1.2 + 0.22*f1 + 0.76*f2 + 0.35*math.Sin(1.69*x+0.2),
			-1.5 + 0.16*f1 - 0.72*f2 + 0.37*math.Cos(1.87*x-0.4),
			0.6 + 0.46*f1 - 0.34*f2 + 0.40*math.Sin(2.09*x),
			-0.2 - 0.38*f1 - 0.42*f2 + 0.39*math.Cos(2.31*x+0.1),
		}
	}
	return rows
}

func generatedWideTwoFactorRows() [][]any {
	const n = 36
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := 1.08*math.Sin(0.29*x) + 0.24*float64(i%8-3)
		f2 := 0.97*math.Cos(0.43*x+0.5) - 0.21*float64(i%6-2)
		cross := 0.28 * math.Sin(0.17*x+0.4)
		rows[i] = []any{
			1.5 + 0.78*f1 + 0.16*f2 + 0.34*math.Sin(1.13*x),
			-0.2 + 0.70*f1 - 0.11*f2 + 0.32*math.Cos(1.31*x),
			0.8 + 0.64*f1 + 0.22*f2 + 0.35*math.Sin(1.59*x+0.3),
			-1.1 - 0.58*f1 + 0.19*f2 + 0.33*math.Cos(1.77*x-0.2),
			2.0 + 0.13*f1 + 0.76*f2 + 0.31*math.Sin(2.01*x),
			-1.7 - 0.18*f1 + 0.69*f2 + 0.36*math.Cos(2.19*x+0.4),
			0.4 + 0.26*f1 - 0.62*f2 + cross + 0.30*math.Sin(2.41*x),
			-0.9 + 0.34*f1 + 0.48*f2 - cross + 0.37*math.Cos(2.63*x),
		}
	}
	return rows
}

func generatedAlternatingMissingRows() [][]any {
	rows := generatedWideTwoFactorRows()
	rows[2][1] = nil
	rows[7][5] = nil
	rows[13][0] = nil
	rows[19][6] = nil
	rows[27][3] = nil
	return rows
}

// generatedNearCollinearRows produces a 30x5 matrix where columns 0 and 1 are
// near-duplicates (correlation ~0.999). Stresses ML/MINRES optimization,
// pseudo-inverse fallback in scoring, and KMO partial-correlation stability.
func generatedNearCollinearRows() [][]any {
	const n = 30
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		base := 1.20*math.Sin(0.31*x) + 0.42*math.Cos(0.19*x)
		other := 0.85*math.Sin(0.57*x+0.3) - 0.30*math.Cos(0.41*x)
		rows[i] = []any{
			0.5 + base + 0.0021*math.Sin(13.7*x), // near-duplicate of col 1
			0.5 + base + 0.0023*math.Cos(11.3*x),
			-0.7 + 0.78*base + 0.62*math.Sin(1.71*x),
			1.4 + 0.71*other + 0.58*math.Cos(2.13*x),
			-0.9 + 0.36*base + 0.41*other + 0.55*math.Sin(2.91*x),
		}
	}
	return rows
}

// generatedHeywoodProneRows produces 9x4 data where one variable is almost
// entirely explained by a latent factor (low n, p=4, low residual). Likely to
// produce a Heywood case (communality > 1 / negative uniqueness clamp).
func generatedHeywoodProneRows() [][]any {
	const n = 9
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f := 1.10*math.Sin(0.41*x) + 0.85*math.Cos(0.23*x)
		rows[i] = []any{
			0.4 + 0.99*f + 0.005*math.Sin(11.0*x), // near-perfect indicator
			-0.7 + 0.92*f + 0.18*math.Cos(1.31*x),
			1.1 + 0.88*f + 0.23*math.Sin(1.97*x+0.4),
			-1.3 + 0.81*f + 0.27*math.Cos(2.41*x-0.2),
		}
	}
	return rows
}

// generatedMixedScaleRows produces 22x5 data where columns span very different
// magnitudes (1e-3 to 1e3). Standardization should still produce a valid
// correlation matrix; tests numerical stability of the eigendecomposition.
func generatedMixedScaleRows() [][]any {
	const n = 22
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := math.Sin(0.39*x) + 0.30*math.Cos(0.17*x)
		f2 := math.Cos(0.51*x+0.4) - 0.22*math.Sin(0.29*x)
		rows[i] = []any{
			1e-3 * (1.0 + 0.78*f1 + 0.16*math.Sin(1.13*x)),
			1e3 * (-0.5 + 0.71*f1 + 0.18*math.Cos(1.29*x)),
			1e1 * (0.8 + 0.18*f1 + 0.74*f2 + 0.21*math.Sin(1.57*x)),
			1e-2 * (-1.1 - 0.14*f1 + 0.69*f2 + 0.24*math.Cos(1.83*x)),
			1e2 * (0.6 + 0.41*f1 - 0.36*f2 + 0.27*math.Sin(2.07*x)),
		}
	}
	return rows
}

// generatedNegativeDominantRows produces 26x5 data where the dominant factor
// loads negatively on most variables. Tests sign-standardization parity.
func generatedNegativeDominantRows() [][]any {
	const n = 26
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f := 1.05*math.Sin(0.33*x) + 0.42*math.Cos(0.19*x)
		rows[i] = []any{
			-1.4 - 0.92*f + 0.18*math.Sin(1.21*x),
			-0.6 - 0.88*f + 0.21*math.Cos(1.43*x+0.2),
			-2.1 - 0.81*f + 0.23*math.Sin(1.69*x-0.3),
			-0.9 - 0.74*f + 0.25*math.Cos(1.91*x),
			-1.7 - 0.67*f + 0.27*math.Sin(2.13*x+0.5),
		}
	}
	return rows
}

// generatedNarrowPlusGroupRows produces 14x6 data where N is barely larger
// than p, with two factors of very different strength. Stresses degree of
// freedom calculation and ML's small-sample behavior.
func generatedNarrowPlusGroupRows() [][]any {
	const n = 14
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		fStrong := 1.30*math.Sin(0.43*x) + 0.55*math.Cos(0.21*x)
		fWeak := 0.42*math.Sin(0.71*x+0.3) - 0.18*math.Cos(0.59*x)
		rows[i] = []any{
			0.5 + 0.86*fStrong + 0.11*fWeak + 0.34*math.Sin(1.13*x),
			-0.7 + 0.83*fStrong - 0.09*fWeak + 0.31*math.Cos(1.31*x),
			1.2 + 0.79*fStrong + 0.13*fWeak + 0.33*math.Sin(1.57*x+0.2),
			-0.4 + 0.12*fStrong + 0.49*fWeak + 0.36*math.Cos(1.79*x),
			0.9 - 0.10*fStrong + 0.45*fWeak + 0.32*math.Sin(2.01*x+0.4),
			-1.1 + 0.16*fStrong - 0.41*fWeak + 0.34*math.Cos(2.23*x),
		}
	}
	return rows
}

// generatedHeavyTailRows produces 28x5 data with heavy-tailed (cubic) noise
// that violates multivariate normality assumed by ML. Useful to check
// extraction-method divergence between PAF/MINRES (least-squares) and ML.
func generatedHeavyTailRows() [][]any {
	const n = 28
	rows := make([][]any, n)
	for i := range n {
		x := float64(i)
		f1 := math.Sin(0.31*x) + 0.30*math.Cos(0.17*x)
		f2 := math.Cos(0.49*x+0.3) - 0.22*math.Sin(0.27*x)
		// cubic noise gives heavy tails
		n1 := math.Pow(0.55*math.Sin(1.31*x), 3)
		n2 := math.Pow(0.50*math.Cos(1.09*x+0.2), 3)
		rows[i] = []any{
			0.4 + 0.78*f1 + 0.15*f2 + n1,
			-0.6 + 0.71*f1 - 0.11*f2 + n2,
			1.1 + 0.18*f1 + 0.74*f2 + n1*0.6,
			-1.4 - 0.13*f1 + 0.69*f2 + n2*0.7,
			0.7 + 0.39*f1 - 0.34*f2 + n1*0.5 + n2*0.4,
		}
	}
	return rows
}

func generatedFactorAnalysisDatasets() []factorAnalysisDataset {
	rowsWithMissing := generatedFactorAnalysisRows(22, 2)
	rowsWithMissing[3][4] = nil
	rowsWithMissing[17][1] = nil

	return []factorAnalysisDataset{
		{name: "generated_oblique", rows: generatedFactorAnalysisRows(24, 0), nFactors: 2},
		{name: "generated_scaled_shifted", rows: generatedFactorAnalysisRows(26, 1), nFactors: 2},
		{name: "generated_complete_case", rows: rowsWithMissing, nFactors: 2},
		{name: "generated_moderate_three_factor", rows: generatedModerateThreeFactorRows(), nFactors: 3},
		{name: "generated_weak_two_factor", rows: generatedWeakTwoFactorRows(), nFactors: 2},
		{name: "generated_high_correlation", rows: generatedHighCorrelationRows(), nFactors: 1},
		{name: "generated_mixed_sign", rows: generatedMixedSignRows(), nFactors: 2},
		{name: "generated_wide_two_factor", rows: generatedWideTwoFactorRows(), nFactors: 2},
		{name: "generated_alternating_missing", rows: generatedAlternatingMissingRows(), nFactors: 2},
		// adversarial / divergence-prone datasets
		{name: "generated_near_collinear", rows: generatedNearCollinearRows(), nFactors: 2},
		{name: "generated_heywood_prone", rows: generatedHeywoodProneRows(), nFactors: 1},
		{name: "generated_mixed_scale", rows: generatedMixedScaleRows(), nFactors: 2},
		{name: "generated_negative_dominant", rows: generatedNegativeDominantRows(), nFactors: 1},
		{name: "generated_narrow_plus_group", rows: generatedNarrowPlusGroupRows(), nFactors: 2},
		{name: "generated_heavy_tail", rows: generatedHeavyTailRows(), nFactors: 2},
	}
}

func factorAnalysisFullCombinationDatasets() []factorAnalysisDataset {
	base := factorAnalysisDatasets()
	generated := generatedFactorAnalysisDatasets()
	return []factorAnalysisDataset{
		base[0],       // two_blocks: 10x6, two clean factors
		base[1],       // one_factor: 18x4, single latent factor
		base[2],       // three_blocks: 12x6, three orthogonal blocks
		base[3],       // cross_loading: 12x5, factors with cross-loading variable
		base[4],       // missing_rows: triggers listwise deletion path
		generated[0],  // generated_oblique: 24x6, correlated latent factors
		generated[3],  // generated_moderate_three_factor: 32x7
		generated[5],  // generated_high_correlation: 28x5, single dominant factor
		generated[7],  // generated_wide_two_factor: 36x8, wider variable count
		generated[8],  // generated_alternating_missing: missing values across cols
		generated[9],  // generated_near_collinear: tests Pinv fallback / KMO stability
		generated[11], // generated_mixed_scale: standardization stress test
		generated[12], // generated_negative_dominant: sign-standardization stress test
		generated[14], // generated_heavy_tail: ML vs PAF/MINRES divergence
	}
}

// factorAnalysisAdversarialDatasets returns the small but high-signal weird
// datasets used for targeted divergence-detection tests.
func factorAnalysisAdversarialDatasets() []factorAnalysisDataset {
	g := generatedFactorAnalysisDatasets()
	return []factorAnalysisDataset{
		g[9],  // near_collinear
		g[10], // heywood_prone
		g[11], // mixed_scale
		g[12], // negative_dominant
		g[13], // narrow_plus_group
		g[14], // heavy_tail
	}
}

func floatRowsToAny(rows [][]float64) [][]any {
	out := make([][]any, len(rows))
	for i := range rows {
		out[i] = make([]any, len(rows[i]))
		for j := range rows[i] {
			out[i][j] = rows[i][j]
		}
	}
	return out
}

func dataTableFromAnyRows(rows [][]any) *insyra.DataTable {
	if len(rows) == 0 {
		return insyra.NewDataTable()
	}
	nCols := len(rows[0])
	dt := insyra.NewDataTable()
	for c := 0; c < nCols; c++ {
		col := insyra.NewDataList().SetName("C")
		for r := 0; r < len(rows); r++ {
			if rows[r][c] == nil {
				col.Append(math.NaN())
			} else {
				col.Append(rows[r][c])
			}
		}
		dt.AppendCols(col)
	}
	return dt
}

func factorOptions(extraction stats.FactorExtractionMethod, rotation stats.FactorRotationMethod, scoring stats.FactorScoreMethod) stats.FactorAnalysisOptions {
	opt := stats.DefaultFactorAnalysisOptions()
	opt.Count.Method = stats.FactorCountFixed
	opt.Count.FixedK = 2
	opt.Extraction = extraction
	opt.Rotation.Method = rotation
	opt.Scoring = scoring
	return opt
}

func dataTableMatrix(dt insyra.IDataTable) [][]float64 {
	if dt == nil {
		return nil
	}
	rows, cols := dt.Size()
	out := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		out[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			out[i][j] = dt.GetElementByNumberIndex(i, j).(float64)
		}
	}
	return out
}

func assertFactorAnalysisMatchesR(t *testing.T, got *stats.FactorModel, rb crossLangBaseline, tol float64) {
	t.Helper()
	if got.CountUsed <= 0 {
		t.Fatalf("expected positive factor count, got %d", got.CountUsed)
	}
	if !got.Converged {
		t.Fatalf("expected factor extraction to converge")
	}
	if !got.RotationConverged {
		t.Fatalf("expected factor rotation to converge")
	}
	assertMatrixCloseToBoth(t, "loadings", dataTableMatrix(got.Loadings), baselineFloatMatrix(t, rb, "loadings"), baselineFloatMatrix(t, rb, "loadings"), tol)
	assertMatrixCloseToBoth(t, "unrotated_loadings", dataTableMatrix(got.UnrotatedLoadings), baselineFloatMatrix(t, rb, "unrotated_loadings"), baselineFloatMatrix(t, rb, "unrotated_loadings"), tol)
	assertMatrixCloseToBoth(t, "structure", dataTableMatrix(got.Structure), baselineFloatMatrix(t, rb, "structure"), baselineFloatMatrix(t, rb, "structure"), tol)
	assertSliceCloseToBoth(t, "uniquenesses", got.Uniquenesses.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "uniquenesses"), baselineFloatSlice(t, rb, "uniquenesses"), tol)
	assertMatrixCloseToBoth(t, "communalities", dataTableMatrix(got.Communalities), baselineFloatMatrix(t, rb, "communalities"), baselineFloatMatrix(t, rb, "communalities"), tol)
	assertSliceCloseToBoth(t, "eigenvalues", got.Eigenvalues.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "eigenvalues"), baselineFloatSlice(t, rb, "eigenvalues"), tol)
	assertSliceCloseToBoth(t, "explained", got.ExplainedProportion.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "explained_proportion"), baselineFloatSlice(t, rb, "explained_proportion"), tol)
	assertSliceCloseToBoth(t, "cumulative", got.CumulativeProportion.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "cumulative_proportion"), baselineFloatSlice(t, rb, "cumulative_proportion"), tol)
	assertOptionalMatrixCloseToR(t, "phi", dataTableMatrix(got.Phi), rb, "phi", tol)
	assertOptionalMatrixCloseToR(t, "rotation_matrix", dataTableMatrix(got.RotationMatrix), rb, "rotation_matrix", tol)
	assertSliceCloseToBoth(t, "sampling_adequacy", got.SamplingAdequacy.GetColByNumber(0).ToF64Slice(), baselineFloatSlice(t, rb, "sampling_adequacy"), baselineFloatSlice(t, rb, "sampling_adequacy"), tol)
	bart := baselineMap(t, rb, "bartlett")
	assertCloseToBoth(t, "bartlett.chi_square", got.BartlettTest.ChiSquare, baselineFloat(t, bart, "chi_square"), baselineFloat(t, bart, "chi_square"), tol)
	assertCloseToBoth(t, "bartlett.df", float64(got.BartlettTest.DegreesOfFreedom), baselineFloat(t, bart, "df"), baselineFloat(t, bart, "df"), tol)
	assertCloseToBoth(t, "bartlett.p_value", got.BartlettTest.PValue, baselineFloat(t, bart, "p_value"), baselineFloat(t, bart, "p_value"), tol)
	assertCloseToBoth(t, "bartlett.sample_size", float64(got.BartlettTest.SampleSize), baselineFloat(t, bart, "sample_size"), baselineFloat(t, bart, "sample_size"), tol)
}

func baselineMap(t *testing.T, m crossLangBaseline, key string) crossLangBaseline {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	raw, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("baseline key %q has non-object type %T", key, v)
	}
	return crossLangBaseline(raw)
}

func assertOptionalMatrixCloseToR(t *testing.T, label string, got [][]float64, rb crossLangBaseline, key string, tol float64) {
	t.Helper()
	if rb[key] == nil {
		if got != nil {
			t.Fatalf("%s expected nil, got %v", label, got)
		}
		return
	}
	if rawMap, ok := rb[key].(map[string]any); ok && len(rawMap) == 0 {
		if got != nil {
			t.Fatalf("%s expected nil, got %v", label, got)
		}
		return
	}
	want := baselineFloatMatrix(t, rb, key)
	assertMatrixCloseToBoth(t, label, got, want, want, tol)
}

func TestCrossLangFactorAnalysisExtractions(t *testing.T) {
	requireFactorAnalysisRTools(t)
	rows := factorAnalysisRows()
	cases := []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionPAF,
		stats.FactorExtractionML,
		stats.FactorExtractionMINRES,
	}
	for _, extraction := range cases {
		t.Run(string(extraction), func(t *testing.T) {
			opt := factorOptions(extraction, stats.FactorRotationOblimin, stats.FactorScoreRegression)
			got, err := stats.FactorAnalysis(dataTableFromRows(rows), opt)
			if err != nil {
				t.Fatalf("FactorAnalysis error: %v", err)
			}
			rb := runRBaseline(t, "factor_analysis", map[string]any{
				"rows": rows, "extraction": string(extraction), "rotation": "oblimin", "scoring": "regression", "nfactors": 2,
			})
			assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
		})
	}
}

func TestCrossLangFactorAnalysisRotations(t *testing.T) {
	requireFactorAnalysisRTools(t)
	rows := factorAnalysisRows()
	cases := []stats.FactorRotationMethod{
		stats.FactorRotationNone,
		stats.FactorRotationVarimax,
		stats.FactorRotationQuartimax,
		stats.FactorRotationQuartimin,
		stats.FactorRotationOblimin,
		stats.FactorRotationGeominT,
		stats.FactorRotationBentlerT,
		stats.FactorRotationSimplimax,
		stats.FactorRotationGeominQ,
		stats.FactorRotationBentlerQ,
		stats.FactorRotationPromax,
	}
	for _, rotation := range cases {
		t.Run(string(rotation), func(t *testing.T) {
			opt := factorOptions(stats.FactorExtractionMINRES, rotation, stats.FactorScoreNone)
			got, err := stats.FactorAnalysis(dataTableFromRows(rows), opt)
			if err != nil {
				t.Fatalf("FactorAnalysis error: %v", err)
			}
			rb := runRBaseline(t, "factor_analysis", map[string]any{
				"rows": rows, "extraction": "minres", "rotation": string(rotation), "scoring": "none", "nfactors": 2,
			})
			assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
		})
	}
}

func TestCrossLangFactorAnalysisScoring(t *testing.T) {
	requireFactorAnalysisRTools(t)
	rows := factorAnalysisRows()
	cases := []stats.FactorScoreMethod{
		stats.FactorScoreNone,
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
		stats.FactorScoreAndersonRubin,
	}
	for _, scoring := range cases {
		t.Run(string(scoring), func(t *testing.T) {
			opt := factorOptions(stats.FactorExtractionMINRES, stats.FactorRotationOblimin, scoring)
			got, err := stats.FactorAnalysis(dataTableFromRows(rows), opt)
			if err != nil {
				t.Fatalf("FactorAnalysis error: %v", err)
			}
			rb := runRBaseline(t, "factor_analysis", map[string]any{
				"rows": rows, "extraction": "minres", "rotation": "oblimin", "scoring": string(scoring), "nfactors": 2,
			})
			assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
			if scoring != stats.FactorScoreNone {
				assertMatrixCloseToBoth(t, "scores", dataTableMatrix(got.Scores), baselineFloatMatrix(t, rb, "scores"), baselineFloatMatrix(t, rb, "scores"), factorParityTol)
				assertMatrixCloseToBoth(t, "score_coefficients", dataTableMatrix(got.ScoreCoefficients), baselineFloatMatrix(t, rb, "score_coefficients"), baselineFloatMatrix(t, rb, "score_coefficients"), factorParityTol)
				assertMatrixCloseToBoth(t, "score_covariance", dataTableMatrix(got.ScoreCovariance), baselineFloatMatrix(t, rb, "score_covariance"), baselineFloatMatrix(t, rb, "score_covariance"), factorParityTol)
			}
		})
	}
}

func TestCrossLangFactorAnalysisAllModeCombinations(t *testing.T) {
	requireFactorAnalysisRTools(t)
	extractions := []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionPAF,
		stats.FactorExtractionML,
		stats.FactorExtractionMINRES,
	}
	rotations := []stats.FactorRotationMethod{
		stats.FactorRotationNone,
		stats.FactorRotationVarimax,
		stats.FactorRotationQuartimax,
		stats.FactorRotationQuartimin,
		stats.FactorRotationOblimin,
		stats.FactorRotationGeominT,
		stats.FactorRotationBentlerT,
		stats.FactorRotationSimplimax,
		stats.FactorRotationGeominQ,
		stats.FactorRotationBentlerQ,
		stats.FactorRotationPromax,
	}
	scorings := []stats.FactorScoreMethod{
		stats.FactorScoreNone,
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
		stats.FactorScoreAndersonRubin,
	}
	for _, ds := range factorAnalysisFullCombinationDatasets() {
		for _, extraction := range extractions {
			for _, rotation := range rotations {
				for _, scoring := range scorings {
					t.Run(ds.name+"/"+string(extraction)+"/"+string(rotation)+"/"+string(scoring), func(t *testing.T) {
						opt := factorOptions(extraction, rotation, scoring)
						opt.Count.FixedK = ds.nFactors
						got, err := stats.FactorAnalysis(dataTableFromAnyRows(ds.rows), opt)
						if err != nil {
							t.Fatalf("FactorAnalysis error: %v", err)
						}
						rb := runRBaseline(t, "factor_analysis", map[string]any{
							"rows": ds.rows, "extraction": string(extraction), "rotation": string(rotation), "scoring": string(scoring), "nfactors": ds.nFactors,
						})
						assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
						if scoring != stats.FactorScoreNone {
							assertMatrixCloseToBoth(t, "scores", dataTableMatrix(got.Scores), baselineFloatMatrix(t, rb, "scores"), baselineFloatMatrix(t, rb, "scores"), factorParityTol)
							assertMatrixCloseToBoth(t, "score_coefficients", dataTableMatrix(got.ScoreCoefficients), baselineFloatMatrix(t, rb, "score_coefficients"), baselineFloatMatrix(t, rb, "score_coefficients"), factorParityTol)
							assertMatrixCloseToBoth(t, "score_covariance", dataTableMatrix(got.ScoreCovariance), baselineFloatMatrix(t, rb, "score_covariance"), baselineFloatMatrix(t, rb, "score_covariance"), factorParityTol)
						}
					})
				}
			}
		}
	}
}

func TestCrossLangFactorAnalysisRepresentativeDatasets(t *testing.T) {
	requireFactorAnalysisRTools(t)
	base := factorAnalysisDatasets()
	generated := generatedFactorAnalysisDatasets()
	cases := []struct {
		dataset    factorAnalysisDataset
		extraction stats.FactorExtractionMethod
		rotation   stats.FactorRotationMethod
		scoring    stats.FactorScoreMethod
	}{
		{base[1], stats.FactorExtractionPAF, stats.FactorRotationNone, stats.FactorScoreRegression},
		{base[1], stats.FactorExtractionPCA, stats.FactorRotationNone, stats.FactorScoreRegression},
		{base[2], stats.FactorExtractionMINRES, stats.FactorRotationOblimin, stats.FactorScoreRegression},
		{base[2], stats.FactorExtractionML, stats.FactorRotationVarimax, stats.FactorScoreBartlett},
		{base[3], stats.FactorExtractionPAF, stats.FactorRotationPromax, stats.FactorScoreAndersonRubin},
		{base[4], stats.FactorExtractionMINRES, stats.FactorRotationOblimin, stats.FactorScoreRegression},
		{generated[0], stats.FactorExtractionPCA, stats.FactorRotationBentlerQ, stats.FactorScoreAndersonRubin},
		{generated[0], stats.FactorExtractionPAF, stats.FactorRotationGeominQ, stats.FactorScoreBartlett},
		{generated[1], stats.FactorExtractionML, stats.FactorRotationGeominT, stats.FactorScoreBartlett},
		{generated[2], stats.FactorExtractionPAF, stats.FactorRotationQuartimin, stats.FactorScoreRegression},
		{generated[3], stats.FactorExtractionPCA, stats.FactorRotationOblimin, stats.FactorScoreRegression},
		{generated[3], stats.FactorExtractionPAF, stats.FactorRotationGeominQ, stats.FactorScoreAndersonRubin},
		{generated[4], stats.FactorExtractionPCA, stats.FactorRotationQuartimax, stats.FactorScoreRegression},
		{generated[4], stats.FactorExtractionPAF, stats.FactorRotationNone, stats.FactorScoreBartlett},
		{generated[5], stats.FactorExtractionPCA, stats.FactorRotationNone, stats.FactorScoreRegression},
		{generated[5], stats.FactorExtractionPAF, stats.FactorRotationNone, stats.FactorScoreBartlett},
		{generated[6], stats.FactorExtractionML, stats.FactorRotationQuartimin, stats.FactorScoreRegression},
		{generated[6], stats.FactorExtractionPAF, stats.FactorRotationPromax, stats.FactorScoreBartlett},
		{generated[7], stats.FactorExtractionPCA, stats.FactorRotationBentlerQ, stats.FactorScoreAndersonRubin},
		{generated[7], stats.FactorExtractionPAF, stats.FactorRotationGeominT, stats.FactorScoreRegression},
		{generated[8], stats.FactorExtractionPCA, stats.FactorRotationPromax, stats.FactorScoreRegression},
		{generated[8], stats.FactorExtractionPAF, stats.FactorRotationQuartimin, stats.FactorScoreBartlett},
	}
	for _, tc := range cases {
		t.Run(tc.dataset.name+"/"+string(tc.extraction)+"/"+string(tc.rotation)+"/"+string(tc.scoring), func(t *testing.T) {
			opt := factorOptions(tc.extraction, tc.rotation, tc.scoring)
			opt.Count.FixedK = tc.dataset.nFactors
			got, err := stats.FactorAnalysis(dataTableFromAnyRows(tc.dataset.rows), opt)
			if err != nil {
				t.Fatalf("FactorAnalysis error: %v", err)
			}
			rb := runRBaseline(t, "factor_analysis", map[string]any{
				"rows": tc.dataset.rows, "extraction": string(tc.extraction), "rotation": string(tc.rotation), "scoring": string(tc.scoring), "nfactors": tc.dataset.nFactors,
			})
			assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
			if tc.scoring != stats.FactorScoreNone {
				assertMatrixCloseToBoth(t, "scores", dataTableMatrix(got.Scores), baselineFloatMatrix(t, rb, "scores"), baselineFloatMatrix(t, rb, "scores"), factorParityTol)
				assertMatrixCloseToBoth(t, "score_coefficients", dataTableMatrix(got.ScoreCoefficients), baselineFloatMatrix(t, rb, "score_coefficients"), baselineFloatMatrix(t, rb, "score_coefficients"), factorParityTol)
				assertMatrixCloseToBoth(t, "score_covariance", dataTableMatrix(got.ScoreCovariance), baselineFloatMatrix(t, rb, "score_covariance"), baselineFloatMatrix(t, rb, "score_covariance"), factorParityTol)
			}
		})
	}
}

// TestCrossLangFactorAnalysisAdversarialDatasets exercises every extraction on
// the adversarial datasets (near-collinear, Heywood-prone, mixed-scale,
// negative-dominant, narrow-N, heavy-tailed). Designed to catch silent
// algorithmic divergence from R that the well-conditioned datasets miss.
func TestCrossLangFactorAnalysisAdversarialDatasets(t *testing.T) {
	requireFactorAnalysisRTools(t)
	extractions := []stats.FactorExtractionMethod{
		stats.FactorExtractionPCA,
		stats.FactorExtractionPAF,
		stats.FactorExtractionML,
		stats.FactorExtractionMINRES,
	}
	rotations := []stats.FactorRotationMethod{
		stats.FactorRotationNone,
		stats.FactorRotationVarimax,
		stats.FactorRotationOblimin,
		stats.FactorRotationPromax,
	}
	scorings := []stats.FactorScoreMethod{
		stats.FactorScoreRegression,
		stats.FactorScoreBartlett,
	}
	for _, ds := range factorAnalysisAdversarialDatasets() {
		for _, extraction := range extractions {
			for _, rotation := range rotations {
				if ds.nFactors < 2 && rotation != stats.FactorRotationNone {
					continue // single-factor: rotation has no effect
				}
				for _, scoring := range scorings {
					t.Run(ds.name+"/"+string(extraction)+"/"+string(rotation)+"/"+string(scoring), func(t *testing.T) {
						opt := factorOptions(extraction, rotation, scoring)
						opt.Count.FixedK = ds.nFactors
						got, err := stats.FactorAnalysis(dataTableFromAnyRows(ds.rows), opt)
						if err != nil {
							t.Fatalf("FactorAnalysis error: %v", err)
						}
						rb := runRBaseline(t, "factor_analysis", map[string]any{
							"rows": ds.rows, "extraction": string(extraction), "rotation": string(rotation), "scoring": string(scoring), "nfactors": ds.nFactors,
						})
						assertFactorAnalysisMatchesR(t, got, rb, factorParityTol)
						assertMatrixCloseToBoth(t, "scores", dataTableMatrix(got.Scores), baselineFloatMatrix(t, rb, "scores"), baselineFloatMatrix(t, rb, "scores"), factorParityTol)
						assertMatrixCloseToBoth(t, "score_coefficients", dataTableMatrix(got.ScoreCoefficients), baselineFloatMatrix(t, rb, "score_coefficients"), baselineFloatMatrix(t, rb, "score_coefficients"), factorParityTol)
						assertMatrixCloseToBoth(t, "score_covariance", dataTableMatrix(got.ScoreCovariance), baselineFloatMatrix(t, rb, "score_covariance"), baselineFloatMatrix(t, rb, "score_covariance"), factorParityTol)
					})
				}
			}
		}
	}
}
