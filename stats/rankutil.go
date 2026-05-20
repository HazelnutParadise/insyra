// rankutil.go
//
// Layer 1 — rank-based primitives shared by Wilcoxon, Mann-Whitney U,
// Kruskal-Wallis, and Friedman tests.

package stats

import (
	"cmp"

	"github.com/HazelnutParadise/insyra/internal/algorithms"
)

// rankWithTies assigns mid-ranks (average ranks on ties) to values.
// Matches R's rank(values, ties.method = "average") and SciPy
// scipy.stats.rankdata(values, method='average').
//
// Returns:
//   - ranks: ranks[i] is the rank of values[i].
//   - tieGroups: sizes of tied groups with size >= 2, in sorted-value order.
//     Singletons (size 1) are omitted because their (t^3 - t) contribution
//     to the rank-variance tie correction is zero.
func rankWithTies(values []float64) (ranks []float64, tieGroups []int) {
	n := len(values)
	if n == 0 {
		return nil, nil
	}
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	algorithms.ParallelSortStableFunc(idx, func(a, b int) int {
		return cmp.Compare(values[a], values[b])
	})
	ranks = make([]float64, n)
	i := 0
	for i < n {
		j := i + 1
		for j < n && values[idx[j]] == values[idx[i]] {
			j++
		}
		// Positions covered: i .. j-1 (0-indexed) = (i+1) .. j (1-indexed).
		// Mid-rank = average of consecutive integers (i+1)..j = (i + j + 1) / 2.
		avg := float64(i+j+1) / 2.0
		for k := i; k < j; k++ {
			ranks[idx[k]] = avg
		}
		if j-i >= 2 {
			tieGroups = append(tieGroups, j-i)
		}
		i = j
	}
	return ranks, tieGroups
}

// tieCorrectionFactor returns 1 - Σ(t^3 - t) / (N^3 - N), the multiplicative
// correction applied to rank-based variance / statistics when ties are
// present. Returns 1.0 when there are no ties or N <= 1.
//
// Used by:
//   - Mann-Whitney U asymptotic variance
//   - Kruskal-Wallis H (statistic is divided by this factor)
//   - Friedman Q (statistic is divided by a related factor; see Friedman impl)
func tieCorrectionFactor(tieGroups []int, n int) float64 {
	if n <= 1 {
		return 1.0
	}
	N := float64(n)
	denom := N*N*N - N
	if denom == 0 {
		return 1.0
	}
	sum := 0.0
	for _, t := range tieGroups {
		tf := float64(t)
		sum += tf*tf*tf - tf
	}
	return 1.0 - sum/denom
}

// signedRankPositiveSum implements the Wilcoxon signed-rank construction:
// from a slice of differences (x - y for paired, x - mu for single-sample),
// produce W+ = sum of ranks of |d_i| over indices where d_i > 0.
//
// zeroMethod controls handling of zero differences:
//   - "wilcox" (default): drop zero differences before ranking. Matches
//     R wilcox.test and SciPy scipy.stats.wilcoxon defaults.
//   - "pratt": include zeros in the ranking (they get the lowest tied ranks),
//     but their sign is 0 so they contribute to neither W+ nor W-.
//
// Returns Wplus (may be a half-integer when nonzero |d_i| values tie),
// the effective sample size nEff (number of nonzero diffs under "wilcox";
// total nonzero diffs under "pratt"), and the tie-group sizes of |d_i|
// for the asymptotic variance adjustment.
func signedRankPositiveSum(diffs []float64, zeroMethod string) (Wplus float64, nEff int, tieGroups []int) {
	abs := make([]float64, 0, len(diffs))
	signs := make([]int, 0, len(diffs))
	for _, d := range diffs {
		if d == 0 {
			if zeroMethod == "pratt" {
				abs = append(abs, 0)
				signs = append(signs, 0)
			}
			// "wilcox": drop zero entries entirely.
			continue
		}
		if d > 0 {
			abs = append(abs, d)
			signs = append(signs, 1)
		} else {
			abs = append(abs, -d)
			signs = append(signs, -1)
		}
	}
	if zeroMethod == "pratt" {
		// In pratt mode nEff excludes the zeros — they participate in ranking
		// to shift the ranks of nonzero observations, but are not counted in
		// the asymptotic mean/variance denominator. This matches the
		// "modified" Pratt convention.
		nonzero := 0
		for _, s := range signs {
			if s != 0 {
				nonzero++
			}
		}
		nEff = nonzero
	} else {
		nEff = len(abs)
	}
	if len(abs) == 0 {
		return 0, 0, nil
	}
	ranks, tg := rankWithTies(abs)
	for i, s := range signs {
		if s == 1 {
			Wplus += ranks[i]
		}
	}
	return Wplus, nEff, tg
}
