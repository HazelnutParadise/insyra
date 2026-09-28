// nonparam_friedman.go
//
// Layer 4 — Friedman test. The nonparametric counterpart to
// RepeatedMeasuresANOVA.
//
// ** Verified against R friedman.test and SciPy scipy.stats.friedmanchisquare **

package stats

import (
	"errors"
	"fmt"
	"sync"

	"github.com/HazelnutParadise/insyra"
)

// FriedmanTestResult holds the result of a Friedman test.
//
// Statistic = Q (Friedman chi^2 with tie correction); DF = k-1 (conditions
// minus 1); CI is unused (nil); EffectSizes contains Kendall's W coefficient
// of concordance.
type FriedmanTestResult struct {
	testResultBase
	NSubjects   int
	KConditions int
}

// FriedmanTest performs the Friedman test on repeated measures. Each element
// of subjects represents one subject's measurements across k conditions
// (all lists must have the same length k). Ranks are assigned per
// subject (within each row); the Q statistic is tie-corrected and
// referred to chi^2 with k-1 degrees of freedom.
//
// ** Verified using R **
func FriedmanTest(subjects []insyra.IDataList) (*FriedmanTestResult, error) {
	n := len(subjects)
	if n < 2 {
		return nil, errors.New("at least two subjects are required")
	}

	subjectsRaw := make([][]any, n)
	// Converted before any goroutine starts, so a nil list is refused as an
	// empty subject instead of crashing inside a goroutine.
	lists := make([]*insyra.DataList, n)
	for i := range subjects {
		lists[i] = asDataList(subjects[i])
	}
	var wg sync.WaitGroup
	for i := range subjects {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lists[i].AtomicDo(func(dl *insyra.DataList) {
				subjectsRaw[i] = dl.Data()
			})
		}(i)
	}
	wg.Wait()

	k := len(subjectsRaw[0])
	if k < 2 {
		return nil, errors.New("each subject must have at least two conditions")
	}
	// Per-subject row buffer, all converted to float64 up front.
	rows := make([][]float64, n)
	for i, raw := range subjectsRaw {
		if len(raw) != k {
			return nil, fmt.Errorf("subject %d has %d observations, expected %d", i+1, len(raw), k)
		}
		row, err := appendNumericValues(make([]float64, 0, k), raw, func() string { return fmt.Sprintf("subject %d", i+1) }, "condition")
		if err != nil {
			return nil, err
		}
		rows[i] = row
	}

	// Per-row (subject) ranking with mid-rank ties. Sum ranks per condition
	// across subjects, and accumulate the per-row tie-correction term
	// ΣΣ(t_ij^3 - t_ij) across rows for the Q tie adjustment.
	colRankSum := make([]float64, k)
	tieSumPerRow := 0.0 // Σ_i Σ_g (t_{ig}^3 - t_{ig}) across all rows
	for _, row := range rows {
		ranks, tieGroups := rankWithTies(row)
		for j, r := range ranks {
			colRankSum[j] += r
		}
		for _, t := range tieGroups {
			tf := float64(t)
			tieSumPerRow += tf*tf*tf - tf
		}
	}

	// Raw Q = (12 / (n*k*(k+1))) * Σ R_j^2  -  3 * n * (k + 1)
	nf := float64(n)
	kf := float64(k)
	rawQ := 0.0
	for _, r := range colRankSum {
		rawQ += r * r
	}
	rawQ = 12.0/(nf*kf*(kf+1))*rawQ - 3.0*nf*(kf+1)

	// Tie correction (R's convention): divide Q by
	//   1 - Σ(t^3 - t) / (n * (k^3 - k))
	denom := nf * (kf*kf*kf - kf)
	tieFactor := 1.0
	if denom > 0 {
		tieFactor = 1.0 - tieSumPerRow/denom
	}
	Q := rawQ
	if tieFactor > 0 {
		Q = rawQ / tieFactor
	}

	df := kf - 1
	pValue := chiSquaredPValue(Q, df)

	W := kendallsW(Q, n, k)

	return &FriedmanTestResult{
		testResultBase: testResultBase{
			Statistic:   Q,
			PValue:      pValue,
			DF:          &df,
			CI:          nil,
			EffectSizes: []EffectSizeEntry{{Type: "kendalls_w", Value: W}},
		},
		NSubjects:   n,
		KConditions: k,
	}, nil
}
