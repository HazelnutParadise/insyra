// crosslang_nonparam_test.go
//
// Cross-language verification of the rank-based nonparametric tests against
// R wilcox.test / kruskal.test / friedman.test and SciPy
// scipy.stats.wilcoxon / mannwhitneyu / kruskal / friedmanchisquare.
//
// Each field of every public result struct is verified against both R and
// Python. Tolerances:
//   - 1e-9 for exact-distribution outputs (statistic / p-value / CI for
//     untied small-n cases — R, Python, and Go all use the same partition
//     enumeration so the answers are bit-identical up to rounding).
//   - 1e-8 for asymptotic statistics / p-values.
//   - 1e-6 for asymptotic CI (R uses uniroot; we use the closed-form
//     normal approximation snapped to the nearest Walsh average / pairwise
//     difference, which differs by at most one step).

package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// ---- Wilcoxon (single + paired) -----------------------------------------

func TestCrossLangWilcoxonPaired(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name string
		x    []float64
		y    []float64
		alt  stats.AlternativeHypothesis
		cl   float64
	}{
		{
			name: "exact_two_sided",
			x:    []float64{1.83, 0.50, 1.62, 2.48, 1.68, 1.88, 1.55, 3.06, 1.30},
			y:    []float64{0.878, 0.647, 0.598, 2.05, 1.06, 1.29, 1.06, 3.14, 1.29},
			alt:  stats.TwoSided, cl: 0.95,
		},
		{
			name: "exact_greater",
			x:    []float64{15, 12, 18, 14, 16, 20, 13, 17},
			y:    []float64{10, 11, 14, 9, 13, 15, 8, 14},
			alt:  stats.Greater, cl: 0.95,
		},
		{
			name: "exact_less",
			x:    []float64{1, 3, 2, 5, 4, 6},
			y:    []float64{5, 7, 8, 9, 6, 10},
			alt:  stats.Less, cl: 0.90,
		},
		{
			// Mild ties (mix of distinct floats with some shared |d|), still
			// asymptotic because n_eff is large enough to overlap with R's
			// continuity-corrected uniroot path.
			name: "mild_ties_asymptotic",
			x:    []float64{5.1, 6.4, 7.3, 8.2, 9.5, 5.7, 6.6, 7.8, 8.1, 9.4, 5.3, 6.5, 7.7, 8.0, 9.2, 5.8, 6.7, 7.9, 8.3, 9.6},
			y:    []float64{5.0, 6.3, 7.2, 8.0, 9.4, 5.6, 6.5, 7.7, 7.9, 9.3, 5.2, 6.4, 7.6, 7.8, 9.1, 5.7, 6.6, 7.7, 8.1, 9.4},
			alt:  stats.TwoSided, cl: 0.95,
		},
		{
			name: "large_n_asymptotic_untied",
			x: []float64{
				1.2, 2.3, 3.4, 4.5, 5.6, 6.7, 7.8, 8.9, 9.0, 10.1,
				11.2, 12.3, 13.4, 14.5, 15.6, 16.7, 17.8, 18.9, 19.0, 20.1,
				21.2, 22.3, 23.4, 24.5, 25.6, 26.7, 27.8, 28.9, 29.0, 30.1,
				31.2, 32.3, 33.4, 34.5, 35.6, 36.7, 37.8, 38.9, 39.0, 40.1,
				41.2, 42.3, 43.4, 44.5, 45.6, 46.7, 47.8, 48.9, 49.0, 50.1,
			},
			y: []float64{
				0.9, 2.1, 3.0, 4.3, 5.2, 6.1, 7.5, 8.4, 8.7, 9.8,
				10.9, 12.0, 13.1, 14.2, 15.3, 16.4, 17.5, 18.6, 18.7, 19.8,
				20.9, 22.0, 23.1, 24.2, 25.3, 26.4, 27.5, 28.6, 28.7, 29.8,
				30.9, 32.0, 33.1, 34.2, 35.3, 36.4, 37.5, 38.6, 38.7, 39.8,
				40.9, 42.0, 43.1, 44.2, 45.3, 46.4, 47.5, 48.6, 48.7, 49.8,
			},
			alt: stats.TwoSided, cl: 0.95,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			x := dataListFromFloat64(tc.x)
			y := dataListFromFloat64(tc.y)
			got, err := stats.PairedWilcoxon(x, y, tc.alt, tc.cl)
			if err != nil {
				t.Fatalf("PairedWilcoxon error: %v", err)
			}

			payload := map[string]any{
				"x": tc.x, "y": tc.y, "alt": altR(tc.alt), "cl": tc.cl,
			}
			rb := runRBaseline(t, "wilcoxon_paired", payload)
			pb := runPythonBaseline(t, "wilcoxon_paired", payload)

			tolStat := 1e-9
			if got.Method == "asymptotic" {
				tolStat = 1e-8
			}
			assertCloseToBoth(t, "stat (W+)", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), tolStat)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), tolStat)
			assertCloseToBoth(t, "rank_biserial", got.EffectSizes[0].Value, baselineFloat(t, rb, "rank_biserial"), baselineFloat(t, pb, "rank_biserial"), 1e-9)

			rMethod := baselineString(t, rb, "method")
			pMethod := baselineString(t, pb, "method")
			if got.Method != rMethod || got.Method != pMethod {
				t.Errorf("method mismatch go=%q r=%q py=%q", got.Method, rMethod, pMethod)
			}

			rNEff := int(baselineFloat(t, rb, "n_eff"))
			pNEff := int(baselineFloat(t, pb, "n_eff"))
			if got.NEffective != rNEff || got.NEffective != pNEff {
				t.Errorf("n_eff mismatch go=%d r=%d py=%d", got.NEffective, rNEff, pNEff)
			}

			if got.Method == "asymptotic" {
				assertCloseToBoth(t, "z", got.Z, baselineFloat(t, rb, "z"), baselineFloat(t, pb, "z"), 1e-8)
			} else if !math.IsNaN(got.Z) {
				t.Errorf("z should be NaN in exact mode, got %v", got.Z)
			}

			// Exact CI: bit-identical across Go / R / Python (index-based on
			// sorted Walsh averages or pairwise differences). Asymptotic CI:
			// both Go and R bracket the continuity-corrected z root via
			// uniroot/bisection, but on piecewise-constant z they may land
			// on different sides of the same Walsh-boundary discontinuity.
			// One Walsh step is the natural granularity of the asymptotic
			// CI; allow up to two steps' slack on integer-spaced data
			// (~0.1 for our tied test fixtures).
			tolCI := 1e-9
			if got.Method == "asymptotic" {
				tolCI = 0.1
			}
			assertCloseToBoth(t, "ci.lo", got.CI[0], baselineFloat(t, rb, "ci_lo"), baselineFloat(t, pb, "ci_lo"), tolCI)
			assertCloseToBoth(t, "ci.hi", got.CI[1], baselineFloat(t, rb, "ci_hi"), baselineFloat(t, pb, "ci_hi"), tolCI)
		})
	}
}

func TestCrossLangWilcoxonSingle(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name string
		x    []float64
		mu   float64
		alt  stats.AlternativeHypothesis
		cl   float64
	}{
		{name: "exact_two_sided", x: []float64{1.83, 0.50, 1.62, 2.48, 1.68, 1.88, 1.55, 3.06, 1.30}, mu: 1.5, alt: stats.TwoSided, cl: 0.95},
		{name: "exact_greater", x: []float64{5.1, 6.2, 4.8, 7.3, 5.5, 6.0, 7.1, 5.7}, mu: 5, alt: stats.Greater, cl: 0.95},
		{
			name: "mild_ties_asymptotic",
			x:    []float64{5.1, 6.4, 5.2, 7.3, 6.5, 5.4, 6.6, 7.4, 5.5, 8.0, 6.7, 5.3, 7.5, 8.1, 5.6, 7.6, 9.0, 6.8, 5.7, 8.2},
			mu:   6.5, alt: stats.TwoSided, cl: 0.95,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stats.SingleSampleWilcoxon(dataListFromFloat64(tc.x), tc.mu, tc.alt, tc.cl)
			if err != nil {
				t.Fatalf("SingleSampleWilcoxon error: %v", err)
			}
			payload := map[string]any{"x": tc.x, "mu": tc.mu, "alt": altR(tc.alt), "cl": tc.cl}
			rb := runRBaseline(t, "wilcoxon_single", payload)
			pb := runPythonBaseline(t, "wilcoxon_single", payload)

			tolStat := 1e-9
			if got.Method == "asymptotic" {
				tolStat = 1e-8
			}
			assertCloseToBoth(t, "stat (W+)", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), tolStat)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), tolStat)
			assertCloseToBoth(t, "rank_biserial", got.EffectSizes[0].Value, baselineFloat(t, rb, "rank_biserial"), baselineFloat(t, pb, "rank_biserial"), 1e-9)

			rMethod := baselineString(t, rb, "method")
			if got.Method != rMethod {
				t.Errorf("method mismatch go=%q r=%q", got.Method, rMethod)
			}
			rNEff := int(baselineFloat(t, rb, "n_eff"))
			if got.NEffective != rNEff {
				t.Errorf("n_eff mismatch go=%d r=%d", got.NEffective, rNEff)
			}

			if got.Method == "asymptotic" {
				assertCloseToBoth(t, "z", got.Z, baselineFloat(t, rb, "z"), baselineFloat(t, pb, "z"), 1e-8)
			}

			// Exact CI: bit-identical across Go / R / Python (index-based on
			// sorted Walsh averages or pairwise differences). Asymptotic CI:
			// both Go and R bracket the continuity-corrected z root via
			// uniroot/bisection, but on piecewise-constant z they may land
			// on different sides of the same Walsh-boundary discontinuity.
			// One Walsh step is the natural granularity of the asymptotic
			// CI; allow up to two steps' slack on integer-spaced data
			// (~0.1 for our tied test fixtures).
			tolCI := 1e-9
			if got.Method == "asymptotic" {
				tolCI = 0.1
			}
			assertCloseToBoth(t, "ci.lo", got.CI[0], baselineFloat(t, rb, "ci_lo"), baselineFloat(t, pb, "ci_lo"), tolCI)
			assertCloseToBoth(t, "ci.hi", got.CI[1], baselineFloat(t, rb, "ci_hi"), baselineFloat(t, pb, "ci_hi"), tolCI)
		})
	}
}

// ---- Mann-Whitney U -----------------------------------------------------

func TestCrossLangMannWhitneyU(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name string
		x    []float64
		y    []float64
		alt  stats.AlternativeHypothesis
		cl   float64
	}{
		{
			name: "exact_two_sided",
			x:    []float64{15, 18, 22, 11, 30, 14, 26, 25},
			y:    []float64{10, 9, 13, 17, 7, 12, 19, 8, 20},
			alt:  stats.TwoSided, cl: 0.95,
		},
		{
			name: "exact_greater",
			x:    []float64{55, 60, 58, 62, 65, 59, 63},
			y:    []float64{40, 45, 48, 50, 52, 47, 49, 51},
			alt:  stats.Greater, cl: 0.95,
		},
		{
			name: "tied_asymptotic",
			x:    []float64{5, 6, 5, 7, 6, 8, 7, 6, 5, 8, 7, 6},
			y:    []float64{4, 5, 4, 6, 5, 7, 6, 5, 4, 5, 6, 4},
			alt:  stats.TwoSided, cl: 0.95,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stats.MannWhitneyU(dataListFromFloat64(tc.x), dataListFromFloat64(tc.y), tc.alt, tc.cl)
			if err != nil {
				t.Fatalf("MannWhitneyU error: %v", err)
			}
			payload := map[string]any{"x": tc.x, "y": tc.y, "alt": altR(tc.alt), "cl": tc.cl}
			rb := runRBaseline(t, "mwu", payload)
			pb := runPythonBaseline(t, "mwu", payload)

			tolStat := 1e-9
			if got.Method == "asymptotic" {
				tolStat = 1e-8
			}
			assertCloseToBoth(t, "stat (min U)", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), tolStat)
			assertCloseToBoth(t, "u1", got.U1, baselineFloat(t, rb, "u1"), baselineFloat(t, pb, "u1"), 1e-9)
			assertCloseToBoth(t, "u2", got.U2, baselineFloat(t, rb, "u2"), baselineFloat(t, pb, "u2"), 1e-9)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), tolStat)
			assertCloseToBoth(t, "rank_biserial", got.EffectSizes[0].Value, baselineFloat(t, rb, "rank_biserial"), baselineFloat(t, pb, "rank_biserial"), 1e-9)
			assertCloseToBoth(t, "cles_a12", got.EffectSizes[1].Value, baselineFloat(t, rb, "cles_a12"), baselineFloat(t, pb, "cles_a12"), 1e-9)

			rMethod := baselineString(t, rb, "method")
			if got.Method != rMethod {
				t.Errorf("method mismatch go=%q r=%q", got.Method, rMethod)
			}
			if got.Method == "asymptotic" {
				assertCloseToBoth(t, "z", got.Z, baselineFloat(t, rb, "z"), baselineFloat(t, pb, "z"), 1e-8)
			}

			// Exact CI: bit-identical across Go / R / Python (index-based on
			// sorted Walsh averages or pairwise differences). Asymptotic CI:
			// both Go and R bracket the continuity-corrected z root via
			// uniroot/bisection, but on piecewise-constant z they may land
			// on different sides of the same Walsh-boundary discontinuity.
			// One Walsh step is the natural granularity of the asymptotic
			// CI; allow up to two steps' slack on integer-spaced data
			// (~0.1 for our tied test fixtures).
			tolCI := 1e-9
			if got.Method == "asymptotic" {
				tolCI = 0.1
			}
			assertCloseToBoth(t, "ci.lo", got.CI[0], baselineFloat(t, rb, "ci_lo"), baselineFloat(t, pb, "ci_lo"), tolCI)
			assertCloseToBoth(t, "ci.hi", got.CI[1], baselineFloat(t, rb, "ci_hi"), baselineFloat(t, pb, "ci_hi"), tolCI)
		})
	}
}

// ---- Kruskal-Wallis -----------------------------------------------------

func TestCrossLangKruskalWallis(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name   string
		groups [][]float64
	}{
		{
			name: "three_groups_untied",
			groups: [][]float64{
				{2.9, 3.0, 2.5, 2.6, 3.2},
				{3.8, 2.7, 4.0, 2.4},
				{2.8, 3.4, 3.7, 2.2, 2.0},
			},
		},
		{
			name: "three_groups_tied",
			groups: [][]float64{
				{5, 6, 5, 7, 6, 5, 6, 7},
				{6, 7, 8, 7, 8, 9, 8, 7},
				{5, 5, 6, 5, 6, 7, 6, 5},
			},
		},
		{
			name: "four_groups",
			groups: [][]float64{
				{1.2, 1.5, 1.7, 1.3},
				{2.1, 2.4, 2.0, 1.9, 2.3},
				{3.0, 3.5, 3.1, 3.4},
				{4.0, 4.2, 4.1, 4.3, 4.5, 4.4},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			groupLists := make([]insyra.IDataList, len(tc.groups))
			for i, g := range tc.groups {
				groupLists[i] = dataListFromFloat64(g)
			}
			got, err := stats.KruskalWallis(groupLists...)
			if err != nil {
				t.Fatalf("KruskalWallis error: %v", err)
			}
			payload := map[string]any{"groups": tc.groups}
			rb := runRBaseline(t, "kruskal", payload)
			pb := runPythonBaseline(t, "kruskal", payload)

			assertCloseToBoth(t, "stat (H)", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-9)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-9)
			assertCloseToBoth(t, "df", *got.DF, baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-12)
			rN := int(baselineFloat(t, rb, "n_total"))
			pN := int(baselineFloat(t, pb, "n_total"))
			if got.NTotal != rN || got.NTotal != pN {
				t.Errorf("n_total mismatch go=%d r=%d py=%d", got.NTotal, rN, pN)
			}
			rGS := baselineFloatSlice(t, rb, "group_rank_sum")
			pGS := baselineFloatSlice(t, pb, "group_rank_sum")
			assertSliceCloseToBoth(t, "group_rank_sum", got.GroupRankSum, rGS, pGS, 1e-9)
			assertCloseToBoth(t, "epsilon_squared", got.EffectSizes[0].Value, baselineFloat(t, rb, "epsilon_squared"), baselineFloat(t, pb, "epsilon_squared"), 1e-9)
		})
	}
}

// ---- Friedman ----------------------------------------------------------

func TestCrossLangFriedman(t *testing.T) {
	requireCrossLangTools(t)

	cases := []struct {
		name     string
		subjects [][]float64
	}{
		{
			name: "ten_subjects_three_conds_untied",
			subjects: [][]float64{
				{5.40, 5.50, 5.55}, {5.85, 5.70, 5.75}, {5.20, 5.60, 5.51},
				{5.55, 5.49, 5.40}, {5.90, 5.85, 5.69}, {5.45, 5.56, 5.61},
				{5.41, 5.42, 5.36}, {5.46, 5.51, 5.35}, {5.25, 5.15, 5.00},
				{5.85, 5.80, 5.71},
			},
		},
		{
			name: "tied",
			subjects: [][]float64{
				{5, 5, 6}, {6, 5, 7}, {5, 5, 5}, {7, 6, 7}, {6, 6, 7},
				{5, 6, 6}, {7, 7, 8}, {6, 6, 7}, {5, 5, 6}, {7, 6, 7},
			},
		},
		{
			name: "four_conditions",
			subjects: [][]float64{
				{1.1, 2.2, 3.3, 4.4}, {1.2, 2.1, 3.5, 4.0}, {1.4, 2.3, 3.1, 4.2},
				{1.0, 2.5, 3.4, 4.1}, {1.3, 2.0, 3.2, 4.3}, {1.5, 2.4, 3.0, 4.5},
				{1.1, 2.2, 3.6, 4.1}, {1.2, 2.3, 3.4, 4.2},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subjectLists := make([]insyra.IDataList, len(tc.subjects))
			for i, s := range tc.subjects {
				subjectLists[i] = dataListFromFloat64(s)
			}
			got, err := stats.FriedmanTest(subjectLists...)
			if err != nil {
				t.Fatalf("FriedmanTest error: %v", err)
			}
			payload := map[string]any{"subjects": tc.subjects}
			rb := runRBaseline(t, "friedman", payload)
			pb := runPythonBaseline(t, "friedman", payload)

			assertCloseToBoth(t, "stat (Q)", got.Statistic, baselineFloat(t, rb, "stat"), baselineFloat(t, pb, "stat"), 1e-9)
			assertCloseToBoth(t, "p", got.PValue, baselineFloat(t, rb, "p"), baselineFloat(t, pb, "p"), 1e-9)
			assertCloseToBoth(t, "df", *got.DF, baselineFloat(t, rb, "df"), baselineFloat(t, pb, "df"), 1e-12)
			rN := int(baselineFloat(t, rb, "n_subjects"))
			if got.NSubjects != rN {
				t.Errorf("n_subjects mismatch go=%d r=%d", got.NSubjects, rN)
			}
			rK := int(baselineFloat(t, rb, "k_conditions"))
			if got.KConditions != rK {
				t.Errorf("k_conditions mismatch go=%d r=%d", got.KConditions, rK)
			}
			assertCloseToBoth(t, "kendalls_w", got.EffectSizes[0].Value, baselineFloat(t, rb, "kendalls_w"), baselineFloat(t, pb, "kendalls_w"), 1e-9)
		})
	}
}

// ---- helpers ------------------------------------------------------------

// altR maps the stats.AlternativeHypothesis enum to R's wilcox.test alternative
// string format.
func altR(a stats.AlternativeHypothesis) string {
	switch a {
	case stats.TwoSided:
		return "two.sided"
	case stats.Greater:
		return "greater"
	case stats.Less:
		return "less"
	default:
		return "two.sided"
	}
}

// baselineString fetches a string field from a baseline payload.
func baselineString(t *testing.T, m crossLangBaseline, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("baseline missing key %q", key)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("baseline key %q has non-string type %T", key, v)
	}
	return s
}
