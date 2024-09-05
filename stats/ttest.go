// ttest.go

package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

// TTestResult holds the result of a t-test, including t-value and p-value.
type TTestResult struct {
	TValue float64 // t 值
	PValue float64 // p 值
	Df     int     // 自由度
}

// SingleSampleTTest performs a single-sample t-test, comparing the mean of the sample to a given value.
// It returns the t-value, p-value, and degrees of freedom.
func SingleSampleTTest(data insyra.IDataList, mu float64) *TTestResult {
	n := data.Len()
	if n <= 1 {
		insyra.LogWarning("stats.SingleSampleTTest: sample size too small.")
		return nil
	}

	// 計算樣本均值
	mean := data.Mean(false).(float64)

	// 計算標準差和標準誤差
	stddev := data.Stdev(false).(float64)
	standardError := stddev / math.Sqrt(float64(n))

	// 計算 t 值
	tValue := (mean - mu) / standardError

	// 計算 P 值
	pValue := calculatePValue(tValue, n-1)

	return &TTestResult{
		TValue: tValue,
		PValue: pValue,
		Df:     n - 1,
	}
}

// TwoSampleTTest performs an independent two-sample t-test, comparing the means of two samples.
// It returns the t-value, p-value, and degrees of freedom.
func TwoSampleTTest(data1, data2 insyra.IDataList, equalVariance bool) *TTestResult {
	n1 := data1.Len()
	n2 := data2.Len()
	if n1 <= 1 || n2 <= 1 {
		insyra.LogWarning("stats.TwoSampleTTest: sample sizes too small.")
		return nil
	}

	// 計算兩個樣本的均值
	mean1 := data1.Mean(false).(float64)
	mean2 := data2.Mean(false).(float64)

	// 計算兩個樣本的標準差
	stddev1 := data1.Stdev(false).(float64)
	stddev2 := data2.Stdev(false).(float64)

	var standardError float64
	var df int

	// 是否假設兩個樣本具有相等的變異數
	if equalVariance {
		// 使用合併標準差
		poolVariance := ((float64(n1-1) * stddev1 * stddev1) + (float64(n2-1) * stddev2 * stddev2)) / float64(n1+n2-2)
		standardError = math.Sqrt(poolVariance * (1/float64(n1) + 1/float64(n2)))
		df = n1 + n2 - 2
	} else {
		// 使用各自標準差
		standardError = math.Sqrt((stddev1 * stddev1 / float64(n1)) + (stddev2 * stddev2 / float64(n2)))
		df = int(math.Pow((stddev1*stddev1/float64(n1))+(stddev2*stddev2/float64(n2)), 2) /
			((math.Pow(stddev1, 4) / (float64(n1 * n1 * (n1 - 1)))) + (math.Pow(stddev2, 4) / (float64(n2 * n2 * (n2 - 1))))))
	}

	// 計算 t 值
	tValue := (mean1 - mean2) / standardError

	// 計算 P 值
	pValue := calculatePValue(tValue, df)

	return &TTestResult{
		TValue: tValue,
		PValue: pValue,
		Df:     df,
	}
}

// PairedTTest performs a paired t-test, comparing the differences between two paired samples.
// It returns the t-value, p-value, and degrees of freedom.
func PairedTTest(data1, data2 insyra.IDataList) *TTestResult {
	n := data1.Len()
	if n != data2.Len() || n <= 1 {
		insyra.LogWarning("stats.PairedTTest: paired samples must have the same non-zero length.")
		return nil
	}

	// 計算差值
	var diffs []float64
	for i := 0; i < n; i++ {
		diff := data1.Data()[i].(float64) - data2.Data()[i].(float64)
		diffs = append(diffs, diff)
	}

	// 計算差值的均值和標準差
	meanDiff := insyra.NewDataList(diffs).Mean(false).(float64)
	stddevDiff := insyra.NewDataList(diffs).Stdev(false).(float64)

	// 計算 t 值
	standardError := stddevDiff / math.Sqrt(float64(n))
	tValue := meanDiff / standardError

	// 計算自由度 (n - 1)
	df := n - 1

	// 計算 P 值
	pValue := calculatePValue(tValue, df)

	return &TTestResult{
		TValue: tValue,
		PValue: pValue,
		Df:     df,
	}
}

// calculatePValue 基於 t 值和自由度計算 P 值
func calculatePValue(tValue float64, df int) float64 {
	if df <= 0 {
		return 1.0 // 當自由度無效時，P-value 為 1
	}

	// 使用 t 分布來計算雙尾 P-value
	tDist := distuv.StudentsT{
		Mu:    0,           // 平均值
		Sigma: 1,           // 標準差
		Nu:    float64(df), // 自由度
	}

	// 計算 t 值的絕對值，然後進行雙尾檢驗
	tValueAbs := math.Abs(tValue)
	pValue := 2 * (1 - tDist.CDF(tValueAbs))

	return pValue
}
