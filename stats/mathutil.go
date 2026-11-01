package stats

import (
	"math"
	"sort"
)

func resolveConfidenceLevel(cl float64) float64 {
	if cl > 0 && cl < 1 {
		return cl
	}
	return defaultConfidenceLevel
}

func sigmoid(x float64) float64 {
	if x >= 0 {
		z := math.Exp(-x)
		return 1 / (1 + z)
	}
	z := math.Exp(x)
	return z / (1 + z)
}

func symmetricCI(center, margin float64) *[2]float64 {
	ci := [2]float64{center - margin, center + margin}
	return &ci
}

func ciByAlternative(center, margin float64, alternative AlternativeHypothesis) *[2]float64 {
	lower, upper := 0.0, 0.0
	switch alternative {
	case TwoSided:
		lower = center - margin
		upper = center + margin
	case Greater:
		lower = center - margin
		upper = math.Inf(1)
	case Less:
		lower = math.Inf(-1)
		upper = center + margin
	default:
		return nanCIPtr()
	}
	ci := [2]float64{lower, upper}
	return &ci
}

func nanCI() [2]float64 {
	return [2]float64{math.NaN(), math.NaN()}
}

func nanCIPtr() *[2]float64 {
	ci := nanCI()
	return &ci
}

func tMarginOfError(confidenceLevel, df, standardError float64) float64 {
	return tQuantile(1-(1-confidenceLevel)/2, df) * standardError
}

// zMarginOfError returns the half-width of a *two-sided* z confidence interval.
// One-sided alternatives need zMarginOfErrorOneSided (different quantile).
func zMarginOfError(confidenceLevel, standardError float64) float64 {
	return norm.Quantile(1-(1-confidenceLevel)/2) * standardError
}

// zMarginOfErrorOneSided returns the margin for a one-sided z confidence
// bound at the given level: qnorm(cl) · SE.
func zMarginOfErrorOneSided(confidenceLevel, standardError float64) float64 {
	return norm.Quantile(confidenceLevel) * standardError
}

func cohenDEffectSizes(d float64) []EffectSizeEntry {
	return []EffectSizeEntry{{Type: "cohen_d", Value: d}}
}

func etaSquared(ssEffect, ssError float64) float64 {
	return ssEffect / (ssEffect + ssError)
}

func fRatio(ssBetween float64, dfBetween int, ssWithin float64, dfWithin int) float64 {
	return (ssBetween / float64(dfBetween)) / (ssWithin / float64(dfWithin))
}

func correlationToT(r, n float64) float64 {
	if r >= 1 {
		return math.Inf(1)
	}
	if r <= -1 {
		return math.Inf(-1)
	}
	return r * math.Sqrt(n-2) / math.Sqrt(1-r*r)
}

func fisherZTransform(r float64) float64 {
	return 0.5 * math.Log((1+r)/(1-r))
}

func fisherZInverse(z float64) float64 {
	exp2z := math.Exp(2 * z)
	return (exp2z - 1) / (exp2z + 1)
}

// rankBiserialMWU returns the rank-biserial correlation r_rb for the
// Mann-Whitney U test: r_rb = 1 - 2 * U1 / (n1 * n2). Equivalent to
// Cliff's delta; range [-1, 1]. U1 here is the U for the first sample.
func rankBiserialMWU(u1 float64, n1, n2 int) float64 {
	if n1 <= 0 || n2 <= 0 {
		return math.NaN()
	}
	return 1.0 - 2.0*u1/(float64(n1)*float64(n2))
}

// rankBiserialMatched returns the rank-biserial correlation r_rb for
// paired / single-sample Wilcoxon signed-rank tests:
//
//	r_rb = (W+ - W-) / (W+ + W-)
//	     = (2 * Wplus - n*(n+1)/2) / (n*(n+1)/2)
//
// where n is the effective sample size (zeros dropped under "wilcox"
// zero-method). Range [-1, 1].
func rankBiserialMatched(wPlus float64, nEff int) float64 {
	if nEff <= 0 {
		return math.NaN()
	}
	total := float64(nEff) * float64(nEff+1) / 2.0
	wMinus := total - wPlus
	return (wPlus - wMinus) / total
}

// clesA12 returns the Common Language Effect Size A12 = P(X > Y) + 0.5 *
// P(X = Y). When U1 is computed with mid-ranks (ties contribute 0.5),
// this collapses to A12 = U1 / (n1 * n2). Range [0, 1]; A12 = 0.5 means
// no effect.
func clesA12(u1 float64, n1, n2 int) float64 {
	if n1 <= 0 || n2 <= 0 {
		return math.NaN()
	}
	return u1 / (float64(n1) * float64(n2))
}

// epsilonSquaredKW returns the Kruskal-Wallis epsilon-squared effect size:
// epsilon^2 = H / (N - 1). Range [0, 1]. Matches the rcompanion convention
// of using the tie-corrected H and total sample size N.
func epsilonSquaredKW(h float64, n int) float64 {
	if n <= 1 {
		return math.NaN()
	}
	return h / float64(n-1)
}

// kendallsW returns Kendall's coefficient of concordance W for Friedman:
// W = Q / (n * (k - 1)), where Q is the tie-corrected Friedman statistic,
// n is the number of subjects, k is the number of conditions. Range [0, 1].
func kendallsW(q float64, n, k int) float64 {
	if n <= 0 || k <= 1 {
		return math.NaN()
	}
	return q / (float64(n) * float64(k-1))
}

func pearsonFisherCI(r, n, confidenceLevel float64) *[2]float64 {
	if n <= 3 {
		return nanCIPtr()
	}
	if r >= 1 {
		return &[2]float64{1, 1}
	}
	if r <= -1 {
		return &[2]float64{-1, -1}
	}
	z := fisherZTransform(r)
	se := 1 / math.Sqrt(n-3)
	zCritical := norm.Quantile(1 - (1-resolveConfidenceLevel(confidenceLevel))/2)
	lower := fisherZInverse(z - zCritical*se)
	upper := fisherZInverse(z + zCritical*se)
	return &[2]float64{lower, upper}
}

// adjustPValues is R's p.adjust on p-values that are all numbers in [0, 1].
func adjustPValues(p []float64, method PAdjustMethod) []float64 {
	n := len(p)
	fn := float64(n)
	adj := make([]float64, n)
	if method == PAdjustBonferroni {
		for i, v := range p {
			adj[i] = math.Min(1, fn*v)
		}
		return adj
	}

	// order lists positions by p, ascending, ties kept in input order as R's
	// order() does.
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return p[order[a]] < p[order[b]] })

	switch method {
	case PAdjustHolm:
		running := 0.0
		for rank, pos := range order {
			running = math.Max(running, (fn-float64(rank))*p[pos])
			adj[pos] = math.Min(1, running)
		}
	case PAdjustHochberg, PAdjustBH, PAdjustBY:
		scale := 1.0
		if method == PAdjustBY {
			scale = 0
			for k := 1; k <= n; k++ {
				scale += 1 / float64(k)
			}
		}
		// Walk from the largest p-value down, keeping the running minimum.
		running := math.Inf(1)
		for k := n - 1; k >= 0; k-- {
			pos := order[k]
			rank := float64(k + 1) // 1-based ascending rank
			var term float64
			switch method {
			case PAdjustHochberg:
				term = (fn - rank + 1) * p[pos]
			default: // BH and BY
				term = scale * fn / rank * p[pos]
			}
			running = math.Min(running, term)
			adj[pos] = math.Min(1, running)
		}
	}
	return adj
}

// allEqualF64 reports whether every value equals the first. The mean of equal
// decimals such as 100 copies of 0.1 is not exactly that decimal, so squared
// deviations from it can be nonzero for a series that has no variance.
func allEqualF64(x []float64) bool {
	for _, v := range x[1:] {
		if v != x[0] {
			return false
		}
	}
	return true
}

// wrightPrefixStat is Wright's (2000) variance-ratio statistic of the series
// whose prefix sums are cum (cum[0] = 0, cum[i] = x[0] + ... + x[i-1]) and
// whose sum of squares is ss, for holding period q:
// ((1/(T q)) * sum over t of the squared q-sum ending at t) / (ss/T) - 1,
// divided by sqrt(2(2q-1)(q-1)/(3qT)).
func wrightPrefixStat(cum []float64, ss float64, q int) float64 {
	n := len(cum) - 1
	nf, qf := float64(n), float64(q)
	var sq float64
	for t := q; t <= n; t++ {
		s := cum[t] - cum[t-q]
		sq += s * s
	}
	vr := (sq / (nf * qf)) / (ss / nf)
	phi := 2 * (2*qf - 1) * (qf - 1) / (3 * qf * nf)
	return (vr - 1) / math.Sqrt(phi)
}

// prefixSums returns cum with cum[0] = 0 and cum[i] = x[0] + ... + x[i-1],
// and the sum of squares of x.
func prefixSums(x []float64) (cum []float64, ss float64) {
	cum = make([]float64, len(x)+1)
	for i, v := range x {
		cum[i+1] = cum[i] + v
		ss += v * v
	}
	return cum, ss
}

// tailSides reports whether a simulated statistic is at or below, and at or
// above, the observed one, allowing for the rounding that summing the same
// values in another order brings, so a draw equal to the observed sample
// counts on both sides.
func tailSides(sim, obs float64) (below, above bool) {
	tol := 1e-12 * (1 + math.Abs(obs))
	return sim <= obs+tol, sim >= obs-tol
}

// equalTailP is the equal-tailed two-sided Monte Carlo p-value of a statistic
// with below draws at or below it and above draws at or above it out of reps:
// twice the smaller of (1 + below) / (reps + 1) and (1 + above) / (reps + 1),
// capped at 1. It matches tests whose critical values are the lower and upper
// percentiles of the simulated distribution, as Wright's are, which matters
// when that distribution is skewed.
func equalTailP(below, above, reps int) float64 {
	p := 2 * float64(1+min(below, above)) / float64(reps+1)
	return math.Min(p, 1)
}
