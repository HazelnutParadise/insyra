package stats

import (
	"math"

	"gonum.org/v1/gonum/stat/distuv"
)

func tTwoTailedPValue(t, df float64) float64 {
	if df <= 0 {
		return math.NaN()
	}
	dist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}
	return 2 * (1 - dist.CDF(math.Abs(t)))
}

func tCDF(t, df float64) float64 {
	if df <= 0 {
		return math.NaN()
	}
	return distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}.CDF(t)
}

func tQuantile(p, df float64) float64 {
	if df <= 0 {
		return math.NaN()
	}
	return distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}.Quantile(p)
}

func fOneTailedPValue(f, df1, df2 float64) float64 {
	return 1 - distuv.F{D1: df1, D2: df2}.CDF(f)
}

func fTwoTailedPValue(f, df1, df2 float64) float64 {
	dist := distuv.F{D1: df1, D2: df2}
	return 2 * math.Min(dist.CDF(f), 1-dist.CDF(f))
}

func chiSquaredPValue(chi2, df float64) float64 {
	return 1 - distuv.ChiSquared{K: df}.CDF(chi2)
}

func zCDF(z float64) float64 {
	return norm.CDF(z)
}

func zQuantile(p float64) float64 {
	return norm.Quantile(p)
}

func zPValue(z float64, alt AlternativeHypothesis) float64 {
	switch alt {
	case TwoSided:
		return 2 * (1 - zCDF(math.Abs(z)))
	case Greater:
		return 1 - zCDF(z)
	case Less:
		return zCDF(z)
	default:
		return math.NaN()
	}
}

// wilcoxonSignedRankExactCounts returns the PMF of W+ on n untied
// observations under the null. counts[k] is the number of subsets S of
// {1, 2, ..., n} whose elements sum to k; total = 2^n. P(W+ = k) =
// counts[k] / total. Used by exact p-value and quantile helpers.
//
// Implementation: standard 1-D DP rolling on
//
//	c(k, n) = c(k, n-1) + c(k-n, n-1).
//
// Time O(n^3), memory O(n^2). For n <= 50 this is trivial (max sum 1275).
func wilcoxonSignedRankExactCounts(n int) (counts []float64, total float64) {
	if n <= 0 {
		return nil, 0
	}
	maxSum := n * (n + 1) / 2
	counts = make([]float64, maxSum+1)
	counts[0] = 1.0
	for k := 1; k <= n; k++ {
		for s := maxSum; s >= k; s-- {
			counts[s] += counts[s-k]
		}
	}
	return counts, math.Pow(2, float64(n))
}

// wilcoxonSignedRankExactCDF returns P(W+ <= w) for n untied observations.
// For tied data the caller must fall back to the asymptotic path.
func wilcoxonSignedRankExactCDF(w float64, n int) float64 {
	if n <= 0 {
		return math.NaN()
	}
	maxSum := n * (n + 1) / 2
	if w < 0 {
		return 0.0
	}
	if w >= float64(maxSum) {
		return 1.0
	}
	counts, total := wilcoxonSignedRankExactCounts(n)
	wInt := int(math.Floor(w))
	sum := 0.0
	for j := 0; j <= wInt; j++ {
		sum += counts[j]
	}
	return sum / total
}

// wilcoxonSignedRankExactPValue returns the alt-adjusted p-value for the
// observed W+ under the exact null distribution on n untied observations.
// Matches R wilcox.test exact p-value:
//
//   - TwoSided: 2 * min(P(W+ <= w), P(W+ >= w)), capped at 1
//   - Greater:  P(W+ >= w) = 1 - P(W+ <= w - 1)
//   - Less:     P(W+ <= w)
func wilcoxonSignedRankExactPValue(w float64, n int, alt AlternativeHypothesis) float64 {
	if n <= 0 {
		return math.NaN()
	}
	pLeq := wilcoxonSignedRankExactCDF(w, n)
	pGeq := 1.0 - wilcoxonSignedRankExactCDF(w-1, n)
	switch alt {
	case Less:
		return pLeq
	case Greater:
		return pGeq
	case TwoSided:
		p := 2 * math.Min(pLeq, pGeq)
		if p > 1 {
			p = 1
		}
		return p
	default:
		return math.NaN()
	}
}

// qsignrank returns the smallest k such that P(W+ <= k) >= p, i.e. the
// p-th quantile of the exact W+ distribution on n untied observations.
// Matches R qsignrank(p, n).
func qsignrank(p float64, n int) int {
	if n <= 0 || p < 0 || p > 1 {
		return -1
	}
	counts, total := wilcoxonSignedRankExactCounts(n)
	maxSum := n * (n + 1) / 2
	cum := 0.0
	for k := 0; k <= maxSum; k++ {
		cum += counts[k]
		if cum/total >= p {
			return k
		}
	}
	return maxSum
}

// mannWhitneyUExactCounts returns the PMF of U on n1 vs n2 untied
// observations. counts[u] = number of arrangements producing U = u;
// total = C(n1+n2, n1) = sum(counts). P(U = u) = counts[u] / total.
//
// Construction: 2-D dynamic program on f(u, i, j) = # arrangements of i
// X's and j Y's yielding U-statistic u. By the "last element" decomposition:
//
//	f(u, i, j) = f(u-j, i-1, j) + f(u, i, j-1).
//
// We maintain dp[i][u] = f(u, i, j_current) and iterate j from 1 to n2,
// updating dp in-place via dp[i][u] += dp[i-1][u-j].
//
// Time O(n1 * n2 * maxU) = O(n1^2 * n2^2). For n1 = n2 = 25 this is
// ~400k ops. Memory O(n1 * maxU).
func mannWhitneyUExactCounts(n1, n2 int) (counts []float64, total float64) {
	if n1 <= 0 || n2 <= 0 {
		return nil, 0
	}
	maxU := n1 * n2
	dp := make([][]float64, n1+1)
	for i := range dp {
		dp[i] = make([]float64, maxU+1)
	}
	// Base (j_current = 0): f(0, i, 0) = 1 for all i; f(u>0, i, 0) = 0.
	for i := 0; i <= n1; i++ {
		dp[i][0] = 1
	}
	for j := 1; j <= n2; j++ {
		// Update dp from (i, u) for j-1 to j:
		//   dp[i][u] += dp[i-1][u-j]   when u >= j.
		// Iterate i ascending so dp[i-1] for the new j is already updated
		// before reading.
		for i := 1; i <= n1; i++ {
			for u := j; u <= maxU; u++ {
				dp[i][u] += dp[i-1][u-j]
			}
		}
	}
	counts = dp[n1]
	for _, c := range counts {
		total += c
	}
	return counts, total
}

// mannWhitneyUExactCDF returns P(U <= u) for n1 vs n2 untied observations.
// For tied data the caller must fall back to the asymptotic path.
func mannWhitneyUExactCDF(u float64, n1, n2 int) float64 {
	if n1 <= 0 || n2 <= 0 {
		return math.NaN()
	}
	maxU := n1 * n2
	if u < 0 {
		return 0.0
	}
	if u >= float64(maxU) {
		return 1.0
	}
	counts, total := mannWhitneyUExactCounts(n1, n2)
	uInt := int(math.Floor(u))
	sum := 0.0
	for j := 0; j <= uInt; j++ {
		sum += counts[j]
	}
	return sum / total
}

// mannWhitneyUExactPValue returns the alt-adjusted p-value for the observed
// U statistic under the exact null distribution. U here is conventionally
// U1 (the U for the first sample): the "greater" alternative corresponds to
// large U1. Matches R wilcox.test exact p-value:
//
//   - TwoSided: 2 * min(P(U <= u), P(U >= u)), capped at 1
//   - Greater:  P(U >= u) = 1 - P(U <= u - 1)
//   - Less:     P(U <= u)
func mannWhitneyUExactPValue(u float64, n1, n2 int, alt AlternativeHypothesis) float64 {
	if n1 <= 0 || n2 <= 0 {
		return math.NaN()
	}
	pLeq := mannWhitneyUExactCDF(u, n1, n2)
	pGeq := 1.0 - mannWhitneyUExactCDF(u-1, n1, n2)
	switch alt {
	case Less:
		return pLeq
	case Greater:
		return pGeq
	case TwoSided:
		p := 2 * math.Min(pLeq, pGeq)
		if p > 1 {
			p = 1
		}
		return p
	default:
		return math.NaN()
	}
}

// qwilcox returns the smallest u such that P(U <= u) >= p — the p-th
// quantile of the exact Mann-Whitney U distribution on (n1, n2) untied
// observations. Matches R qwilcox(p, n1, n2).
func qwilcox(p float64, n1, n2 int) int {
	if n1 <= 0 || n2 <= 0 || p < 0 || p > 1 {
		return -1
	}
	counts, total := mannWhitneyUExactCounts(n1, n2)
	maxU := n1 * n2
	cum := 0.0
	for k := 0; k <= maxU; k++ {
		cum += counts[k]
		if cum/total >= p {
			return k
		}
	}
	return maxU
}
