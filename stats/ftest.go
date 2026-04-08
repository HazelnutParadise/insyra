package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
)

type FTestResult struct {
	testResultBase
	DF2 float64 // degree of freedom for the second group
}

// FTestForVarianceEquality performs an F-test for variance equality.
func FTestForVarianceEquality(data1, data2 insyra.IDataList) *FTestResult {
	var var1, var2 float64
	var len1, len2 int

	data1.AtomicDo(func(d1 *insyra.DataList) {
		data2.AtomicDo(func(d2 *insyra.DataList) {
			var1 = d1.Var()
			var2 = d2.Var()
			len1 = d1.Len()
			len2 = d2.Len()
		})
	})

	var fValue float64
	if var1 > var2 {
		fValue = var1 / var2
	} else {
		fValue = var2 / var1
	}

	df1 := float64(len1 - 1)
	df2 := float64(len2 - 1)
	pValue := fTwoTailedPValue(fValue, df1, df2)

	return &FTestResult{
		testResultBase: testResultBase{
			Statistic: fValue,
			PValue:    pValue,
			DF:        &df1,
		},
		DF2: df2,
	}
}

// LeveneTest performs Levene's test for equality of variances across groups.
func LeveneTest(groups []insyra.IDataList) *FTestResult {
	k := len(groups)
	if k < 2 {
		insyra.LogWarning("stats", "LeveneTest", "At least two groups required")
		return nil
	}

	var allDiffs []float64
	var groupLabels []int

	for i, group := range groups {
		group.AtomicDo(func(gdl *insyra.DataList) {
			median := gdl.Median()
			for _, v := range gdl.Data() {
				x, ok := insyra.ToFloat64Safe(v)
				if !ok {
					continue
				}
				allDiffs = append(allDiffs, math.Abs(x-median))
				groupLabels = append(groupLabels, i)
			}
		})
	}

	return oneWayANOVAForLevene(allDiffs, groupLabels, k)
}

// BartlettTest performs Bartlett's test for equality of variances.
func BartlettTest(groups []insyra.IDataList) *FTestResult {
	k := len(groups)
	if k < 2 {
		insyra.LogWarning("stats", "BartlettTest", "At least two groups required")
		return nil
	}

	var pooledLogVar float64
	var sumNMinus1 int
	var weight float64

	for _, group := range groups {
		var n int
		var v float64
		group.AtomicDo(func(gdl *insyra.DataList) {
			n = gdl.Len()
			v = gdl.Var()
		})

		if n < 2 || v <= 0 {
			continue
		}

		sumNMinus1 += n - 1
		pooledLogVar += float64(n-1) * math.Log(v)
		weight += 1.0 / float64(n-1)
	}

	if sumNMinus1 == 0 {
		return nil
	}

	meanVar := 0.0
	for _, group := range groups {
		if group.Len() >= 2 {
			meanVar += float64(group.Len()-1) * group.Var()
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
	}
}

// FTestForRegression performs an overall F-test for a regression model.
func FTestForRegression(ssr, sse float64, df1, df2 int) *FTestResult {
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
	}
}

// FTestForNestedModels compares two nested regression models.
func FTestForNestedModels(rssReduced, rssFull float64, dfReduced, dfFull int) *FTestResult {
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
	}
}

func oneWayANOVAForLevene(values []float64, labels []int, k int) *FTestResult {
	stats := oneWayANOVAFromSlices(values, labels, k)
	if stats == nil {
		return nil
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
	}
}
