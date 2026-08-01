// nonparam_kw.go
//
// Layer 4 — Kruskal-Wallis H test. The nonparametric counterpart to
// OneWayANOVA.
//
// ** Verified against R kruskal.test and SciPy scipy.stats.kruskal **

package stats

import (
	"errors"
	"fmt"
	"sync"

	"github.com/HazelnutParadise/insyra"
)

// KruskalWallisResult holds the result of a Kruskal-Wallis H test.
//
// Statistic = H (tie-corrected); DF = k-1 (number of groups minus 1);
// CI is unused (nil); EffectSizes contains the rank-based epsilon^2.
type KruskalWallisResult struct {
	testResultBase
	NTotal       int
	GroupRankSum []float64 // sum of ranks per group, in input order
}

// KruskalWallis performs the Kruskal-Wallis H test on >= 2 independent
// samples. Ranks are assigned with mid-rank ties; H is tie-corrected
// (divided by 1 - Σ(t^3-t)/(N^3-N)) so the asymptotic chi^2 p-value uses
// the same construction as R kruskal.test.
//
// ** Verified using R **
func KruskalWallis(groups ...insyra.IDataList) (*KruskalWallisResult, error) {
	if len(groups) < 2 {
		return nil, errors.New("at least two groups are required")
	}

	// Pull data in parallel (each list is its own actor; no contention).
	groupsRaw := make([][]any, len(groups))
	var wg sync.WaitGroup
	for i := range groups {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			groups[i].AtomicDo(func(dl *insyra.DataList) {
				groupsRaw[i] = dl.Data()
			})
		}(i)
	}
	wg.Wait()

	values := make([]float64, 0)
	labels := make([]int, 0)
	for i, gd := range groupsRaw {
		if len(gd) == 0 {
			return nil, fmt.Errorf("group %d is empty", i)
		}
		for j, v := range gd {
			x, ok := insyra.ToFloat64Safe(v)
			if !ok {
				return nil, fmt.Errorf("invalid numeric value at group %d index %d", i, j)
			}
			values = append(values, x)
			labels = append(labels, i)
		}
	}

	k := len(groups)
	n := len(values)
	if n < k+1 {
		return nil, errors.New("not enough observations across groups")
	}

	ranks, tieGroups := rankWithTies(values)

	// Per-group rank sum and size.
	groupSum := make([]float64, k)
	groupSize := make([]int, k)
	for i, lbl := range labels {
		groupSum[lbl] += ranks[i]
		groupSize[lbl]++
	}

	// Raw H = 12 / (N(N+1)) * Σ R_i^2 / n_i  -  3(N+1)
	N := float64(n)
	rawH := 0.0
	for i := range k {
		if groupSize[i] == 0 {
			return nil, fmt.Errorf("group %d is empty after rank assignment", i)
		}
		rawH += groupSum[i] * groupSum[i] / float64(groupSize[i])
	}
	rawH = 12.0/(N*(N+1))*rawH - 3.0*(N+1)

	// Tie correction: divide H by (1 - Σ(t^3-t)/(N^3-N))
	tieFactor := tieCorrectionFactor(tieGroups, n)
	var H float64
	if tieFactor > 0 {
		H = rawH / tieFactor
	} else {
		H = rawH
	}

	df := float64(k - 1)
	pValue := chiSquaredPValue(H, df)

	eps2 := epsilonSquaredKW(H, n)

	return &KruskalWallisResult{
		testResultBase: testResultBase{
			Statistic:   H,
			PValue:      pValue,
			DF:          &df,
			CI:          nil,
			EffectSizes: []EffectSizeEntry{{Type: "epsilon_squared", Value: eps2}},
		},
		NTotal:       n,
		GroupRankSum: groupSum,
	}, nil
}
