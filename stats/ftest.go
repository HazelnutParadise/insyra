package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

type FTestResult struct {
	testResultBase
	DF2 float64 // degree of freedom for the second group
}

// FTestForVarianceEquality performs an F-test for variance equality.
func FTestForVarianceEquality(data1, data2 insyra.IDataList) (*FTestResult, error) {
	values1, err := testSeries(data1, "data1")
	if err != nil {
		return nil, err
	}
	values2, err := testSeries(data2, "data2")
	if err != nil {
		return nil, err
	}
	len1, len2 := len(values1), len(values2)
	var1, var2 := sampleVarianceF64(values1), sampleVarianceF64(values2)
	if len1 < 2 || len2 < 2 {
		return nil, errors.New("both samples must have at least two observations")
	}
	if var1 <= 0 || var2 <= 0 {
		return nil, errors.New("sample variances must be greater than zero")
	}

	// F = (larger sample variance) / (smaller sample variance). The numerator
	// and denominator degrees of freedom must follow whichever variance ended
	// up in each position — previously this always used (n1−1, n2−1)
	// regardless, which gave a wrong p-value whenever var2 > var1.
	var fValue, df1, df2 float64
	if var1 > var2 {
		fValue = var1 / var2
		df1 = float64(len1 - 1)
		df2 = float64(len2 - 1)
	} else {
		fValue = var2 / var1
		df1 = float64(len2 - 1)
		df2 = float64(len1 - 1)
	}
	pValue := fTwoTailedPValue(fValue, df1, df2)

	return &FTestResult{
		testResultBase: testResultBase{
			Statistic: fValue,
			PValue:    pValue,
			DF:        &df1,
		},
		DF2: df2,
	}, nil
}

// LeveneTest performs Levene's test for equality of variances across groups.
func LeveneTest(groups []insyra.IDataList) (*FTestResult, error) {
	k := len(groups)
	if k < 2 {
		return nil, errors.New("at least two groups required")
	}

	// Per-group: pull median + raw []any in parallel. Each group is its own
	// actor so per-group AtomicDo entries can run concurrently. Collapses
	// the previously-serial actor-handshake chain (same fix pattern as
	// OneWayANOVA / TwoWayANOVA).
	var allDiffs []float64
	var groupLabels []int
	for i := range groups {
		values, err := testSeries(groups[i], fmt.Sprintf("group %d", i))
		if err != nil {
			return nil, err
		}
		median := medianOfF64(values)
		for _, x := range values {
			allDiffs = append(allDiffs, math.Abs(x-median))
			groupLabels = append(groupLabels, i)
		}
	}

	return oneWayANOVAForLevene(allDiffs, groupLabels, k)
}

// BartlettTest performs Bartlett's test for equality of variances.
func BartlettTest(groups []insyra.IDataList) (*FTestResult, error) {
	k := len(groups)
	if k < 2 {
		return nil, errors.New("at least two groups required")
	}

	// Per-group n + Var via parallel actor entries. Var() is the bulk of
	// the work (two-pass sum + sumsq); doing it in parallel saves the
	// serial actor-entry chain for ≥3 groups.
	type bartlettExtract struct {
		n int
		v float64
	}
	bx := make([]bartlettExtract, len(groups))
	for i := range groups {
		values, err := testSeries(groups[i], fmt.Sprintf("group %d", i))
		if err != nil {
			return nil, err
		}
		bx[i] = bartlettExtract{n: len(values), v: sampleVarianceF64(values)}
	}

	var pooledLogVar float64
	var sumNMinus1 int
	var weight float64
	for i, e := range bx {
		if e.n < 2 || e.v <= 0 {
			return nil, fmt.Errorf("group %d must have at least two observations and positive variance", i)
		}
		sumNMinus1 += e.n - 1
		pooledLogVar += float64(e.n-1) * math.Log(e.v)
		weight += 1.0 / float64(e.n-1)
	}

	if sumNMinus1 == 0 {
		return nil, errors.New("insufficient valid groups for Bartlett test")
	}

	// meanVar reuses the already-extracted (n, var) pairs — no extra
	// AtomicDo calls. The previous form did `group.Len()` + `group.Var()`
	// once more per group inside this loop (= 2k more actor entries).
	meanVar := 0.0
	for _, e := range bx {
		if e.n >= 2 {
			meanVar += float64(e.n-1) * e.v
		}
	}
	meanVar /= float64(sumNMinus1)

	T := (float64(sumNMinus1) * math.Log(meanVar)) - pooledLogVar
	correction := 1.0 + (1.0/(3.0*float64(k-1)))*(weight-1.0/float64(sumNMinus1))
	chiSquared := T / correction

	df := float64(k - 1)
	pValue := chiSquaredPValue(chiSquared, df)

	return &FTestResult{
		testResultBase: testResultBase{
			Statistic: chiSquared,
			PValue:    pValue,
			DF:        &df,
		},
		DF2: 0,
	}, nil
}

// FTestForRegression performs an overall F-test for a regression model.
func FTestForRegression(ssr, sse float64, df1, df2 int) (*FTestResult, error) {
	if df1 <= 0 || df2 <= 0 {
		return nil, errors.New("degrees of freedom must be positive")
	}
	if ssr < 0 || sse <= 0 {
		return nil, errors.New("ssr must be non-negative and sse must be positive")
	}

	fValue := fRatio(ssr, df1, sse, df2)
	pValue := fOneTailedPValue(fValue, float64(df1), float64(df2))

	df1f := float64(df1)
	df2f := float64(df2)
	return &FTestResult{
		testResultBase: testResultBase{
			Statistic: fValue,
			PValue:    pValue,
			DF:        &df1f,
		},
		DF2: df2f,
	}, nil
}

// FTestForNestedModels compares two nested regression models.
func FTestForNestedModels(rssReduced, rssFull float64, dfReduced, dfFull int) (*FTestResult, error) {
	if dfReduced <= dfFull || dfFull <= 0 {
		return nil, errors.New("invalid degrees of freedom for nested models")
	}
	if rssReduced < rssFull || rssFull <= 0 {
		return nil, errors.New("rssReduced must be >= rssFull and rssFull must be positive")
	}

	numeratorDF := dfReduced - dfFull
	denominatorDF := dfFull

	fValue := fRatio(rssReduced-rssFull, numeratorDF, rssFull, denominatorDF)
	pValue := fOneTailedPValue(fValue, float64(numeratorDF), float64(denominatorDF))

	df1f := float64(numeratorDF)
	df2f := float64(denominatorDF)
	return &FTestResult{
		testResultBase: testResultBase{
			Statistic: fValue,
			PValue:    pValue,
			DF:        &df1f,
		},
		DF2: df2f,
	}, nil
}

func oneWayANOVAForLevene(values []float64, labels []int, k int) (*FTestResult, error) {
	stats, err := oneWayANOVAFromSlices(values, labels, k)
	if err != nil {
		return nil, err
	}

	df1 := float64(stats.DFB)
	df2 := float64(stats.DFW)
	return &FTestResult{
		testResultBase: testResultBase{
			Statistic: stats.F,
			PValue:    stats.P,
			DF:        &df1,
		},
		DF2: df2,
	}, nil
}
