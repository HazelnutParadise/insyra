// nonparam_wilcoxon.go
//
// Layer 4 — Wilcoxon signed-rank test (single-sample and paired). The
// nonparametric counterpart to SingleSampleTTest / PairedTTest.
//
// ** Verified against R wilcox.test and SciPy scipy.stats.wilcoxon **

package stats

import (
	"errors"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// WilcoxonTestResult holds the result of a Wilcoxon signed-rank test.
//
// Statistic = W+ (sum of positive ranks); DF is unused (nil); CI is the
// Hodges-Lehmann pseudo-median confidence interval at the requested level;
// EffectSizes contains the matched-pairs rank-biserial correlation.
//
// The asymptotic z and Method ("exact" vs "asymptotic") report which
// distributional path produced the p-value. For exact mode Z is NaN.
type WilcoxonTestResult struct {
	testResultBase
	Z          float64 // standardized z (asymptotic path); NaN for exact
	Method     string  // "exact" or "asymptotic"
	NEffective int     // number of nonzero |d_i| used (zeros dropped under "wilcox")
}

// SingleSampleWilcoxon tests whether the median of `data` equals `mu`,
// using the Wilcoxon signed-rank test on (data - mu).
//
// confidenceLevel is the level for the Hodges-Lehmann pseudo-median CI
// (default 0.95). Zero differences are dropped before ranking (R's
// wilcox.test default zero-method = "wilcox"); for tied |d_i| values
// (after dropping zeros) the asymptotic z with continuity correction is
// used. Otherwise n_eff <= 50 uses the exact distribution.
//
// ** Verified using R **
func SingleSampleWilcoxon(data insyra.IDataList, mu float64, alt AlternativeHypothesis, confidenceLevel ...float64) (*WilcoxonTestResult, error) {
	cl, err := resolveOptionalConfidenceLevel(confidenceLevel)
	if err != nil {
		return nil, err
	}

	var dataSlice []any
	data.AtomicDo(func(dl *insyra.DataList) {
		dataSlice = dl.Data()
	})
	if len(dataSlice) == 0 {
		return nil, errors.New("data is empty")
	}

	diffs := make([]float64, len(dataSlice))
	for i, v := range dataSlice {
		x, ok := insyra.ToFloat64Safe(v)
		if !ok {
			return nil, errors.New("invalid numeric value in data")
		}
		diffs[i] = x - mu
	}

	// SingleSampleWilcoxon estimates the pseudo-median of `data`, so the
	// CI must be reported on the same scale: shift the Walsh-based CI
	// (computed on diffs = data - mu) back by + mu to land in the data
	// scale. Matches R wilcox.test conf.int output.
	return computeWilcoxon(diffs, alt, cl, mu)
}

// PairedWilcoxon tests whether the median of (data1 - data2) equals 0,
// using the Wilcoxon signed-rank test on the paired differences.
//
// confidenceLevel is the level for the Hodges-Lehmann pseudo-median CI of
// the median paired difference (default 0.95). data1 and data2 must have
// the same length. See SingleSampleWilcoxon for tie / zero handling.
//
// ** Verified using R **
func PairedWilcoxon(data1, data2 insyra.IDataList, alt AlternativeHypothesis, confidenceLevel ...float64) (*WilcoxonTestResult, error) {
	cl, err := resolveOptionalConfidenceLevel(confidenceLevel)
	if err != nil {
		return nil, err
	}

	var d1Slice, d2Slice []any
	var inputErr error
	data1.AtomicDo(func(dl1 *insyra.DataList) {
		data2.AtomicDo(func(dl2 *insyra.DataList) {
			if dl1.Len() != dl2.Len() {
				inputErr = errors.New("paired samples must have the same length")
				return
			}
			if dl1.Len() == 0 {
				inputErr = errors.New("paired samples are empty")
				return
			}
			d1Slice = dl1.Data()
			d2Slice = dl2.Data()
		})
	})
	if inputErr != nil {
		return nil, inputErr
	}

	diffs := make([]float64, len(d1Slice))
	for i := range d1Slice {
		x, ok := insyra.ToFloat64Safe(d1Slice[i])
		if !ok {
			return nil, errors.New("invalid numeric value in data1")
		}
		y, ok := insyra.ToFloat64Safe(d2Slice[i])
		if !ok {
			return nil, errors.New("invalid numeric value in data2")
		}
		diffs[i] = x - y
	}

	return computeWilcoxon(diffs, alt, cl, 0)
}

// resolveOptionalConfidenceLevel mirrors the validation used by
// SingleSampleTTest / PairedTTest for the variadic confidenceLevel parameter:
// at most one value, in (0, 1). Missing → defaults to 0.95.
func resolveOptionalConfidenceLevel(cl []float64) (float64, error) {
	var raw float64
	if len(cl) > 0 {
		if len(cl) > 1 {
			return 0, errors.New("confidenceLevel accepts at most one value")
		}
		raw = cl[0]
		if raw <= 0 || raw >= 1 {
			return 0, errors.New("confidenceLevel must be between 0 and 1")
		}
	}
	return resolveConfidenceLevel(raw), nil
}

// computeWilcoxon runs the signed-rank test on a vector of differences,
// using zero-method = "wilcox" (drop zeros). ciOffset is added to both CI
// endpoints to map back to the original scale (e.g., mu for single-sample
// tests; 0 for paired since the natural CI scale is the paired difference).
func computeWilcoxon(diffs []float64, alt AlternativeHypothesis, cl float64, ciOffset float64) (*WilcoxonTestResult, error) {
	if !isValidAlt(alt) {
		return nil, errors.New("invalid alternative hypothesis")
	}

	wPlus, nEff, tieGroups := signedRankPositiveSum(diffs, "wilcox")
	if nEff == 0 {
		// All differences were zero — null is trivially "true" but the test
		// statistic is undefined. Mirror SciPy's behaviour of returning NaN.
		nanCIVal := [2]float64{math.NaN(), math.NaN()}
		return &WilcoxonTestResult{
			testResultBase: testResultBase{
				Statistic:   math.NaN(),
				PValue:      math.NaN(),
				DF:          nil,
				CI:          &nanCIVal,
				EffectSizes: []EffectSizeEntry{{Type: "rank_biserial", Value: math.NaN()}},
			},
			Z:          math.NaN(),
			Method:     "undefined",
			NEffective: 0,
		}, nil
	}

	hasTies := len(tieGroups) > 0
	// Exact path matches R wilcox.test: untied and n_eff < 50.
	useExact := !hasTies && nEff < 50

	var pValue, zVal float64
	var method string
	muW := float64(nEff*(nEff+1)) / 4.0
	// Asymptotic variance with tie correction:
	// σ² = n(n+1)(2n+1)/24 - Σ(t³-t)/48
	sigma2 := float64(nEff*(nEff+1)*(2*nEff+1)) / 24.0
	for _, t := range tieGroups {
		tf := float64(t)
		sigma2 -= (tf*tf*tf - tf) / 48.0
	}
	sigmaW := math.Sqrt(sigma2)

	if useExact {
		pValue = wilcoxonSignedRankExactPValue(wPlus, nEff, alt)
		zVal = math.NaN()
		method = "exact"
	} else {
		// Continuity correction sign depends on alternative.
		var correction float64
		switch alt {
		case TwoSided:
			if wPlus > muW {
				correction = 0.5
			} else if wPlus < muW {
				correction = -0.5
			}
		case Greater:
			correction = 0.5
		case Less:
			correction = -0.5
		}
		zVal = (wPlus - muW - correction) / sigmaW
		pValue = zPValue(zVal, alt)
		method = "asymptotic"
	}

	// Walsh averages of nonzero diffs (Hodges-Lehmann construction).
	nonzero := make([]float64, 0, len(diffs))
	for _, d := range diffs {
		if d != 0 {
			nonzero = append(nonzero, d)
		}
	}
	walsh := walshAverages(nonzero)
	sort.Float64s(walsh)
	pseudoMedian := medianSorted(walsh)

	var ciPtr *[2]float64
	if useExact {
		ciPtr = hodgesLehmannCI(walsh, nEff, tieGroups, alt, cl, true, sigmaW, muW)
	} else {
		// Asymptotic path: solve z(diffs - c) = ±z_crit via uniroot, matching
		// R wilcox.test conf.int.
		lo, hi := asymptoticWilcoxonCI(diffs, cl, alt)
		ci := [2]float64{lo, hi}
		ciPtr = &ci
	}
	// Shift CI to the original scale (no-op when ciOffset == 0).
	if ciOffset != 0 && ciPtr != nil {
		shifted := [2]float64{ciPtr[0], ciPtr[1]}
		if !math.IsInf(shifted[0], 0) && !math.IsNaN(shifted[0]) {
			shifted[0] += ciOffset
		}
		if !math.IsInf(shifted[1], 0) && !math.IsNaN(shifted[1]) {
			shifted[1] += ciOffset
		}
		ciPtr = &shifted
	}

	rRB := rankBiserialMatched(wPlus, nEff)

	res := &WilcoxonTestResult{
		testResultBase: testResultBase{
			Statistic:   wPlus,
			PValue:      pValue,
			DF:          nil,
			CI:          ciPtr,
			EffectSizes: []EffectSizeEntry{{Type: "rank_biserial", Value: rRB}},
		},
		Z:          zVal,
		Method:     method,
		NEffective: nEff,
	}
	_ = pseudoMedian // point estimate is implicit in CI center; not exposed separately to keep API parity with t-test
	return res, nil
}

// walshAverages returns all (x_i + x_j) / 2 for 0 <= i <= j < n.
// Length = n*(n+1)/2.
func walshAverages(x []float64) []float64 {
	n := len(x)
	w := make([]float64, 0, n*(n+1)/2)
	for i := range n {
		for j := i; j < n; j++ {
			w = append(w, (x[i]+x[j])/2)
		}
	}
	return w
}

// medianSorted returns the median of a sorted float64 slice. Empty → NaN.
func medianSorted(v []float64) float64 {
	n := len(v)
	if n == 0 {
		return math.NaN()
	}
	if n%2 == 1 {
		return v[n/2]
	}
	return (v[n/2-1] + v[n/2]) / 2
}

// asymptoticWilcoxonCI returns the asymptotic (1 - α) Hodges-Lehmann CI by
// solving for c in z(diffs - c) = ±z_crit, matching R wilcox.test's
// uniroot-based output. diffs is the input vector (already in shift scale).
// Caller is responsible for adding back the location offset.
//
// The standardized z uses continuity correction with sign(W - μ_W) * 0.5,
// the same form Go's asymptotic p-value path uses. Tie correction is
// applied to σ_W via the |d_i - c| tie groups at each candidate c.
func asymptoticWilcoxonCI(diffs []float64, cl float64, alt AlternativeHypothesis) (lower, upper float64) {
	n := len(diffs)
	if n == 0 {
		return math.NaN(), math.NaN()
	}
	minD := diffs[0]
	maxD := diffs[0]
	for _, d := range diffs {
		if d < minD {
			minD = d
		}
		if d > maxD {
			maxD = d
		}
	}
	span := maxD - minD
	if span == 0 {
		span = 1
	}
	lo := minD - span
	hi := maxD + span
	alpha := 1 - cl

	// z(diffs - c) — continuity-corrected, tie-adjusted.
	zAt := func(c float64) float64 {
		shifted := make([]float64, len(diffs))
		for i, d := range diffs {
			shifted[i] = d - c
		}
		wplus, nEff, tieGroups := signedRankPositiveSum(shifted, "wilcox")
		if nEff == 0 {
			return math.NaN()
		}
		muW := float64(nEff*(nEff+1)) / 4.0
		sigma2 := float64(nEff*(nEff+1)*(2*nEff+1)) / 24.0
		for _, t := range tieGroups {
			tf := float64(t)
			sigma2 -= (tf*tf*tf - tf) / 48.0
		}
		sigma := math.Sqrt(sigma2)
		if sigma == 0 {
			return math.NaN()
		}
		sign := 0.0
		if wplus > muW {
			sign = 1
		} else if wplus < muW {
			sign = -1
		}
		return (wplus - muW - sign*0.5) / sigma
	}

	var zLow, zHigh float64
	switch alt {
	case TwoSided:
		zLow = norm.Quantile(1 - alpha/2)
		zHigh = -zLow
	case Greater:
		zLow = norm.Quantile(1 - alpha)
	case Less:
		zHigh = -norm.Quantile(1 - alpha)
	}

	if alt == Less {
		lower = math.Inf(-1)
	} else {
		lower = uniroot(func(c float64) float64 { return zAt(c) - zLow }, lo, hi, 1e-5)
	}
	if alt == Greater {
		upper = math.Inf(1)
	} else {
		upper = uniroot(func(c float64) float64 { return zAt(c) - zHigh }, lo, hi, 1e-5)
	}
	return lower, upper
}

// uniroot finds c in [lo, hi] such that f(c) ≈ 0 via bisection. Mirrors
// R uniroot for piecewise-constant z(c): the returned value is on the
// "negative" side of the sign change, tol-close to the discontinuity.
func uniroot(f func(float64) float64, lo, hi, tol float64) float64 {
	fLo := f(lo)
	fHi := f(hi)
	if fLo == 0 {
		return lo
	}
	if fHi == 0 {
		return hi
	}
	if fLo*fHi > 0 {
		// No sign change in bracket: return the endpoint closer to zero,
		// signalling the bracket is too narrow (caller should widen).
		if math.Abs(fLo) < math.Abs(fHi) {
			return lo
		}
		return hi
	}
	for range 200 {
		mid := (lo + hi) / 2
		fMid := f(mid)
		if hi-lo < tol {
			return mid
		}
		if fLo*fMid < 0 {
			hi = mid
		} else {
			lo = mid
			fLo = fMid
		}
	}
	return (lo + hi) / 2
}

// hodgesLehmannCI builds the (1 - α) confidence interval for the
// pseudo-median, given the sorted Walsh averages and the test parameters.
//
//   - useExact: derive cutoff index via qsignrank (rank-based on Walsh
//     averages, matching R wilcox.test exact path).
//   - Otherwise: solve for the asymptotic CI endpoints via uniroot on the
//     continuity-corrected z, matching R's asymptotic path.
//
// For one-sided alternatives, the half not bounded by the data is set to
// ±Inf — matching R wilcox.test convention.
func hodgesLehmannCI(walshSorted []float64, nEff int, tieGroups []int, alt AlternativeHypothesis, cl float64, useExact bool, sigmaW, muW float64) *[2]float64 {
	K := len(walshSorted)
	if K == 0 {
		ci := [2]float64{math.NaN(), math.NaN()}
		return &ci
	}
	alpha := 1 - cl
	var aLow, aHigh float64
	switch alt {
	case TwoSided:
		aLow = alpha / 2
		aHigh = alpha / 2
	case Greater:
		aLow = alpha
		aHigh = 0 // upper bound is +Inf
	case Less:
		aLow = 0 // lower bound is -Inf
		aHigh = alpha
	default:
		ci := [2]float64{math.NaN(), math.NaN()}
		return &ci
	}

	quLower := wilcoxonCIIndex(aLow, nEff, useExact, muW, sigmaW)
	quUpper := wilcoxonCIIndex(aHigh, nEff, useExact, muW, sigmaW)
	_ = tieGroups // tie correction already baked into muW / sigmaW upstream

	var lo, hi float64
	if alt == Less {
		lo = math.Inf(-1)
	} else {
		quLower = max(quLower, 1)
		idx := quLower - 1
		if idx >= K {
			idx = K - 1
		}
		lo = walshSorted[idx]
	}
	if alt == Greater {
		hi = math.Inf(1)
	} else {
		quUpper = max(quUpper, 1)
		idx := max(K-quUpper, 0)
		if idx >= K {
			idx = K - 1
		}
		hi = walshSorted[idx]
	}
	ci := [2]float64{lo, hi}
	return &ci
}

// wilcoxonCIIndex returns the 1-based rank cutoff qu such that the CI
// endpoint is walshSorted[qu-1] (0-indexed). For useExact it calls
// qsignrank(alpha, n); otherwise it inverts the continuity-corrected
// normal approximation.
//
// alpha == 0 corresponds to "no cutoff" (one-sided unbounded side) and
// the caller is responsible for not using the returned index in that case.
func wilcoxonCIIndex(alpha float64, nEff int, useExact bool, muW, sigmaW float64) int {
	if alpha <= 0 {
		return 0
	}
	if useExact {
		qu := qsignrank(alpha, nEff)
		if qu == 0 {
			qu = 1
		}
		return qu
	}
	// Asymptotic: qu ≈ muW - z_{1-alpha} * sigmaW - 0.5  (continuity).
	zcrit := norm.Quantile(1 - alpha)
	cFloat := muW - zcrit*sigmaW - 0.5
	if math.IsNaN(cFloat) {
		return 1
	}
	return max(int(math.Round(cFloat)), 1)
}

// isValidAlt checks whether the alternative-hypothesis enum value is in
// the known set.
func isValidAlt(alt AlternativeHypothesis) bool {
	switch alt {
	case TwoSided, Greater, Less:
		return true
	default:
		return false
	}
}
