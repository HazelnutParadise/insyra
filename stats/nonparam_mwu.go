// nonparam_mwu.go
//
// Layer 4 — Mann-Whitney U test (independent samples). The nonparametric
// counterpart to TwoSampleTTest.
//
// ** Verified against R wilcox.test (two-sample) and SciPy
// scipy.stats.mannwhitneyu **

package stats

import (
	"errors"
	"math"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// MannWhitneyUResult holds the result of a Mann-Whitney U test.
//
// Statistic = min(U1, U2). EffectSizes contains rank-biserial r_rb and
// CLES A12. CI is the Hodges-Lehmann shift CI at the requested level.
type MannWhitneyUResult struct {
	testResultBase
	U1     float64
	U2     float64
	Z      float64 // standardized z (asymptotic path); NaN for exact
	Method string  // "exact" or "asymptotic"
}

// MannWhitneyU performs the Wilcoxon-Mann-Whitney rank-sum test on two
// independent samples. Returns U1 (for data1) and U2 (for data2); the
// statistic field is min(U1, U2). The p-value is the alt-adjusted exact
// or asymptotic p-value for U1.
//
// confidenceLevel is the level for the Hodges-Lehmann shift CI (default
// 0.95). When neither sample contains ties (in the combined ranking) and
// both n1, n2 <= 25, the exact distribution is used; otherwise the
// asymptotic normal with continuity correction and tie adjustment.
//
// ** Verified using R **
func MannWhitneyU(data1, data2 insyra.IDataList, alt AlternativeHypothesis, confidenceLevel ...float64) (*MannWhitneyUResult, error) {
	if !isValidAlt(alt) {
		return nil, errors.New("invalid alternative hypothesis")
	}
	cl, err := resolveOptionalConfidenceLevel(confidenceLevel)
	if err != nil {
		return nil, err
	}

	var s1, s2 []any
	var inputErr error
	data1.AtomicDo(func(dl1 *insyra.DataList) {
		data2.AtomicDo(func(dl2 *insyra.DataList) {
			if dl1.Len() == 0 || dl2.Len() == 0 {
				inputErr = errors.New("both samples must be non-empty")
				return
			}
			s1 = dl1.Data()
			s2 = dl2.Data()
		})
	})
	if inputErr != nil {
		return nil, inputErr
	}

	x := make([]float64, len(s1))
	for i, v := range s1 {
		f, ok := insyra.ToFloat64Safe(v)
		if !ok {
			return nil, errors.New("invalid numeric value in data1")
		}
		x[i] = f
	}
	y := make([]float64, len(s2))
	for i, v := range s2 {
		f, ok := insyra.ToFloat64Safe(v)
		if !ok {
			return nil, errors.New("invalid numeric value in data2")
		}
		y[i] = f
	}
	n1 := len(x)
	n2 := len(y)

	// Joint ranking of (x, y) with mid-rank ties.
	all := make([]float64, 0, n1+n2)
	all = append(all, x...)
	all = append(all, y...)
	ranks, tieGroups := rankWithTies(all)
	// Rank sum for sample 1 = R1; U1 = R1 - n1(n1+1)/2.
	r1 := 0.0
	for i := range n1 {
		r1 += ranks[i]
	}
	n1n2 := float64(n1 * n2)
	u1 := r1 - float64(n1*(n1+1))/2.0
	u2 := n1n2 - u1

	hasTies := len(tieGroups) > 0
	// Exact path matches R wilcox.test (two-sample): untied and both n.x < 50, n.y < 50.
	useExact := !hasTies && n1 < 50 && n2 < 50

	// Mean and variance under H0:
	muU := n1n2 / 2.0
	N := n1 + n2
	tieFactor := tieCorrectionFactor(tieGroups, N)
	sigma2 := n1n2 * (float64(N) + 1) / 12.0 * tieFactor
	sigmaU := math.Sqrt(sigma2)

	var pValue, zVal float64
	var method string
	if useExact {
		pValue = mannWhitneyUExactPValue(u1, n1, n2, alt)
		zVal = math.NaN()
		method = "exact"
	} else {
		var correction float64
		switch alt {
		case TwoSided:
			if u1 > muU {
				correction = 0.5
			} else if u1 < muU {
				correction = -0.5
			}
		case Greater:
			correction = 0.5
		case Less:
			correction = -0.5
		}
		if sigmaU == 0 {
			zVal = math.NaN()
			pValue = math.NaN()
		} else {
			zVal = (u1 - muU - correction) / sigmaU
			pValue = zPValue(zVal, alt)
		}
		method = "asymptotic"
	}

	// Hodges-Lehmann shift estimate and CI: median of all (x_i - y_j).
	diffs := make([]float64, 0, n1*n2)
	for i := range n1 {
		for j := range n2 {
			diffs = append(diffs, x[i]-y[j])
		}
	}
	sort.Float64s(diffs)
	shiftEstimate := medianSorted(diffs)
	_ = shiftEstimate // not exposed separately; CI is built around it

	ciPtr := mwuShiftCI(diffs, n1, n2, alt, cl, useExact, muU, sigmaU)

	rRB := rankBiserialMWU(u1, n1, n2)
	cles := clesA12(u1, n1, n2)

	stat := math.Min(u1, u2)
	return &MannWhitneyUResult{
		testResultBase: testResultBase{
			Statistic: stat,
			PValue:    pValue,
			DF:        nil,
			CI:        ciPtr,
			EffectSizes: []EffectSizeEntry{
				{Type: "rank_biserial", Value: rRB},
				{Type: "cles_a12", Value: cles},
			},
		},
		U1:     u1,
		U2:     u2,
		Z:      zVal,
		Method: method,
	}, nil
}

// mwuShiftCI builds the (1-α) Hodges-Lehmann shift CI from the sorted
// pairwise differences diffs[i,j] = x_i - y_j.
//
// The exact path uses qwilcox(α, n1, n2); the asymptotic path uses the
// continuity-corrected normal approximation with the tie-adjusted sigmaU.
func mwuShiftCI(diffsSorted []float64, n1, n2 int, alt AlternativeHypothesis, cl float64, useExact bool, muU, sigmaU float64) *[2]float64 {
	K := len(diffsSorted)
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
		aHigh = 0
	case Less:
		aLow = 0
		aHigh = alpha
	default:
		ci := [2]float64{math.NaN(), math.NaN()}
		return &ci
	}

	quLow := mwuCIIndex(aLow, n1, n2, useExact, muU, sigmaU)
	quHigh := mwuCIIndex(aHigh, n1, n2, useExact, muU, sigmaU)

	var lo, hi float64
	if alt == Less {
		lo = math.Inf(-1)
	} else {
		quLow = max(quLow, 1)
		idx := quLow - 1
		if idx >= K {
			idx = K - 1
		}
		lo = diffsSorted[idx]
	}
	if alt == Greater {
		hi = math.Inf(1)
	} else {
		quHigh = max(quHigh, 1)
		idx := max(K-quHigh, 0)
		if idx >= K {
			idx = K - 1
		}
		hi = diffsSorted[idx]
	}
	ci := [2]float64{lo, hi}
	return &ci
}

// mwuCIIndex returns the 1-based rank cutoff qu such that the CI endpoint
// is diffsSorted[qu-1] (0-indexed).
func mwuCIIndex(alpha float64, n1, n2 int, useExact bool, muU, sigmaU float64) int {
	if alpha <= 0 {
		return 0
	}
	if useExact {
		qu := qwilcox(alpha, n1, n2)
		if qu == 0 {
			qu = 1
		}
		return qu
	}
	zcrit := norm.Quantile(1 - alpha)
	cFloat := muU - zcrit*sigmaU - 0.5
	if math.IsNaN(cFloat) {
		return 1
	}
	return max(int(math.Round(cFloat)), 1)
}
