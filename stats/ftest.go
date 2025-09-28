package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

type FTestResult struct {
	testResultBase
	DF2 float64 // degree of freedom for the second group
}

// FTestForVarianceEquality performs an F-test for variance equality
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

	// 計算 F 值
	var fValue float64
	if var1 > var2 {
		fValue = var1 / var2
	} else {
		fValue = var2 / var1
	}

	// 計算自由度
	df1 := float64(len1 - 1)
	df2 := float64(len2 - 1)

	// 使用卡方分佈計算 P 值
	fDist := distuv.F{D1: df1, D2: float64(df2)}
	pValue := 2 * math.Min(fDist.CDF(fValue), 1-fDist.CDF(fValue)) // 使用雙尾檢定

	return &FTestResult{
		testResultBase: testResultBase{
			Statistic: fValue,
			PValue:    pValue,
			DF:        &df1,
		},
		DF2: df2,
	}
}

// LeveneTest performs Levene's Test for equality of variances across multiple groups.
// Input: slice of *insyra.DataList, each representing a group.
// Output: *FTestResult
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

	// 對 |x - median| 做單因子 ANOVA
	return oneWayANOVA(allDiffs, groupLabels, k)
}

// BartlettTest performs Bartlett's test for equality of variances.
// Input: slice of *insyra.DataList, each representing a group.
func BartlettTest(groups []insyra.IDataList) *FTestResult {
	k := len(groups)
	if k < 2 {
		insyra.LogWarning("stats", "BartlettTest", "At least two groups required")
		return nil
	}

	var totalN int
	var pooledLogVar float64
	var sumLogVar float64
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

		totalN += n
		sumNMinus1 += n - 1
		pooledLogVar += float64(n-1) * math.Log(v)
		sumLogVar += float64(n - 1)
		weight += 1.0 / float64(n-1)
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
	pValue := 1 - distuv.ChiSquared{K: df}.CDF(chiSquared)

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
// ssr: regression sum of squares
// sse: error sum of squares
// df1: degrees of freedom for the model (number of predictors)
// df2: degrees of freedom for residuals (n - k - 1)
func FTestForRegression(ssr, sse float64, df1, df2 int) *FTestResult {
	fValue := (ssr / float64(df1)) / (sse / float64(df2))
	fDist := distuv.F{D1: float64(df1), D2: float64(df2)}
	pValue := 1 - fDist.CDF(fValue)

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
// rssReduced: residual sum of squares of reduced model
// rssFull: residual sum of squares of full model
// dfReduced, dfFull: degrees of freedom of both models
func FTestForNestedModels(rssReduced, rssFull float64, dfReduced, dfFull int) *FTestResult {
	numeratorDF := dfReduced - dfFull
	denominatorDF := dfFull

	fValue := ((rssReduced - rssFull) / float64(numeratorDF)) / (rssFull / float64(denominatorDF))
	fDist := distuv.F{D1: float64(numeratorDF), D2: float64(denominatorDF)}
	pValue := 1 - fDist.CDF(fValue)

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

// 之後用正式ANOVA替代
// oneWayANOVA 用於 LeveneTest 中
func oneWayANOVA(values []float64, labels []int, k int) *FTestResult {
	if len(values) != len(labels) {
		return nil
	}

	n := len(values)
	groupSums := make([]float64, k)
	groupCounts := make([]int, k)
	totalSum := 0.0

	for i, v := range values {
		group := labels[i]
		groupSums[group] += v
		groupCounts[group]++
		totalSum += v
	}

	totalMean := totalSum / float64(n)

	// 計算組間平方和（SSB）
	ssb := 0.0
	for i := range k {
		mean := groupSums[i] / float64(groupCounts[i])
		ssb += float64(groupCounts[i]) * (mean - totalMean) * (mean - totalMean)
	}

	// 計算組內平方和（SSW）
	ssw := 0.0
	for i, v := range values {
		group := labels[i]
		mean := groupSums[group] / float64(groupCounts[group])
		ssw += (v - mean) * (v - mean)
	}

	df1 := float64(k - 1)
	df2 := float64(n - k)
	msb := ssb / df1
	msw := ssw / df2
	fValue := msb / msw

	pValue := 1 - distuv.F{D1: df1, D2: df2}.CDF(fValue)

	return &FTestResult{
		testResultBase: testResultBase{
			Statistic: fValue,
			PValue:    pValue,
			DF:        &df1,
		},
		DF2: df2,
	}
}
