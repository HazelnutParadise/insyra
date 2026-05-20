package stats

import "math"

func resolveConfidenceLevel(cl float64) float64 {
	if cl > 0 && cl < 1 {
		return cl
	}
	return defaultConfidenceLevel
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
