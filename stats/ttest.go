// ttest.go

package stats

import (
	"math"
	"sync"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat/distuv"
)

type TTestResult struct {
	testResultBase
	Mean     *float64 // mean of the first group (or the only group)
	Mean2    *float64 // mean of the second group (nil if not applicable)
	MeanDiff *float64 // mean difference (only for paired t-test)
	N        int      // sample size of the first group (or the only group or paired group)
	N2       *int     // sample size of the second group (nil if not applicable)
}

// SingleSampleTTest performs a one-sample t-test comparing the sample mean to a known population mean.
// Parameters:
//   - data: The sample data to test
//   - mu: The hypothesized population mean to compare against
//   - confidenceLevel: (Optional) Confidence level for the confidence interval (e.g., 0.95 for 95%, 0.99 for 99%)
//     Must be between 0 and 1. If not provided or invalid, defaults to 0.95
//
// ** Verified using R **
func SingleSampleTTest(data insyra.IDataList, mu float64, confidenceLevel ...float64) *TTestResult {
	var n int
	var mean, stddev float64
	isFailed := false
	data.AtomicDo(func(dl *insyra.DataList) {
		n = dl.Len()
		if n <= 1 {
			insyra.LogWarning("stats", "SingleSampleTTest", "Sample size too small")
			isFailed = true
			return
		}
		mean = dl.Mean()
		stddev = dl.Stdev()
	})
	if isFailed {
		return nil
	}

	standardError := stddev / math.Sqrt(float64(n))
	tValue := (mean - mu) / standardError
	df := float64(n - 1)
	pValue := calculateTPValue(tValue, df)

	// Handle optional confidence level parameter
	var cl float64
	if len(confidenceLevel) > 0 {
		cl = confidenceLevel[0]
	} else {
		cl = defaultConfidenceLevel
	}

	if cl <= 0 || cl >= 1 {
		cl = defaultConfidenceLevel
	}

	tDist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}
	tCritical := tDist.Quantile(1 - (1-cl)/2)
	marginOfError := tCritical * standardError

	ci := &[2]float64{mean - marginOfError, mean + marginOfError}

	// Handle constant data (stddev == 0)
	if stddev == 0 {
		if mean == mu {
			effectSize := 0.0
			tValue = math.NaN()
			pValue = math.NaN()
			effectSizes := []EffectSizeEntry{{Type: "cohen_d", Value: effectSize}}
			return &TTestResult{
				testResultBase: testResultBase{
					Statistic:   tValue,
					PValue:      pValue,
					DF:          &df,
					CI:          ci,
					EffectSizes: effectSizes,
				},
				Mean: &mean,
				N:    n,
			}

		} else {
			effectSize := math.Inf(int(math.Copysign(1, mean-mu)))
			tValue = math.Inf(int(math.Copysign(1, mean-mu)))
			pValue = 0
			effectSizes := []EffectSizeEntry{{Type: "cohen_d", Value: effectSize}}
			return &TTestResult{
				testResultBase: testResultBase{
					Statistic:   tValue,
					PValue:      pValue,
					DF:          &df,
					CI:          ci,
					EffectSizes: effectSizes,
				},
				Mean: &mean,
				N:    n,
			}
		}
	}

	effectSize := (mean - mu) / stddev //Preserve the sign
	effectSizes := []EffectSizeEntry{
		{Type: "cohen_d", Value: effectSize},
	}

	return &TTestResult{
		testResultBase: testResultBase{
			Statistic:   tValue,
			PValue:      pValue,
			DF:          &df,
			CI:          ci,
			EffectSizes: effectSizes,
		},
		Mean: &mean,
		N:    n,
	}
}

// TwoSampleTTest performs a two-sample t-test comparing the means of two independent groups.
// Parameters:
//   - data1, data2: The two data groups to compare
//   - equalVariance: Whether to assume equal variances between groups
//   - confidenceLevel: (Optional) Confidence level for the confidence interval (e.g., 0.95 for 95%, 0.99 for 99%)
//     Must be between 0 and 1. If not provided or invalid, defaults to 0.95
//
// ** Verified using R **
func TwoSampleTTest(data1, data2 insyra.IDataList, equalVariance bool, confidenceLevel ...float64) *TTestResult {
	var n1, n2 int
	var mean1, mean2 float64
	var stddev1, stddev2 float64
	isFailed := false
	data1.AtomicDo(func(dl1 *insyra.DataList) {
		data2.AtomicDo(func(dl2 *insyra.DataList) {
			n1 = dl1.Len()
			n2 = dl2.Len()
			if n1 <= 1 || n2 <= 1 {
				insyra.LogWarning("stats", "TwoSampleTTest", "Sample sizes too small")
				isFailed = true
				return
			}

			mean1 = dl1.Mean()
			mean2 = dl2.Mean()
			stddev1 = dl1.Stdev()
			stddev2 = dl2.Stdev()
		})
	})
	if isFailed {
		return nil
	}

	meanDiff := mean1 - mean2

	n1Float := float64(n1)
	n2Float := float64(n2)
	var1 := stddev1 * stddev1
	var2 := stddev2 * stddev2

	var standardError float64
	var df float64

	if equalVariance {
		poolVar := ((float64(n1-1)*var1 + float64(n2-1)*var2) / float64(n1+n2-2))
		standardError = math.Sqrt(poolVar * (1/n1Float + 1/n2Float))
		df = float64(n1 + n2 - 2)
	} else {
		se1 := var1 / n1Float
		se2 := var2 / n2Float
		standardError = math.Sqrt(se1 + se2)

		seSum := se1 + se2
		num := seSum * seSum
		den := (se1 * se1 / (n1Float - 1)) + (se2 * se2 / (n2Float - 1))
		df = num / den
	}
	tValue := meanDiff / standardError
	pValue := calculateTPValue(tValue, df)

	// Handle optional confidence level parameter
	var cl float64
	if len(confidenceLevel) > 0 {
		cl = confidenceLevel[0]
	} else {
		cl = defaultConfidenceLevel
	}

	if cl <= 0 || cl >= 1 {
		cl = defaultConfidenceLevel
	}

	tDist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}
	tCritical := tDist.Quantile(1 - (1-cl)/2)
	marginOfError := tCritical * standardError

	ci := &[2]float64{meanDiff - marginOfError, meanDiff + marginOfError}

	var effectSize float64
	if equalVariance {
		pooledVar := ((n1Float-1)*var1 + (n2Float-1)*var2) / (n1Float + n2Float - 2)
		pooledStd := math.Sqrt(pooledVar)
		effectSize = meanDiff / pooledStd // Preserve the sign
	} else {
		effectSize = meanDiff / math.Sqrt((var1+var2)/2) // Preserve the sign
	}

	effectSizes := []EffectSizeEntry{
		{Type: "cohen_d", Value: effectSize},
	}

	return &TTestResult{
		testResultBase: testResultBase{
			Statistic:   tValue,
			PValue:      pValue,
			DF:          &df,
			CI:          ci,
			EffectSizes: effectSizes,
		},
		Mean:  &mean1,
		Mean2: &mean2,
		N:     n1,
		N2:    &n2,
	}
}

// PairedTTest performs a paired-samples t-test comparing the means of two related groups.
// The data must be paired observations (same subjects measured twice).
// Parameters:
//   - data1, data2: The paired data groups to compare (must have same length)
//   - confidenceLevel: (Optional) Confidence level for the confidence interval (e.g., 0.95 for 95%, 0.99 for 99%)
//     Must be between 0 and 1. If not provided or invalid, defaults to 0.95
//
// ** Verified using R **
func PairedTTest(data1, data2 insyra.IDataList, confidenceLevel ...float64) *TTestResult {
	var n int
	isFailed := false
	var data1Slice, data2Slice []any
	data1.AtomicDo(func(dl1 *insyra.DataList) {
		data2.AtomicDo(func(dl2 *insyra.DataList) {
			n = dl1.Len()
			if n != dl2.Len() || n <= 1 {
				insyra.LogWarning("stats", "PairedTTest", "Paired samples must have the same non-zero length")
				isFailed = true
				return
			}

			data1Slice = dl1.Data()
			data2Slice = dl2.Data()
		})
	})
	if isFailed {
		return nil
	}

	// 僅對大型數據集使用平行運算
	const minSizeForParallel = 5000
	var sum, sumSq float64

	if n >= minSizeForParallel {
		// 決定 goroutine 數量 (根據 CPU 核心數和數據大小調整)
		numGoroutines := 4
		if n > 50000 {
			numGoroutines = 8
		}

		chunkSize := n / numGoroutines
		var wg sync.WaitGroup

		// 創建結果集合
		sums := make([]float64, numGoroutines)
		sumSqs := make([]float64, numGoroutines)

		// 啟動多個 goroutine 平行處理數據
		for i := range numGoroutines {
			wg.Add(1)

			// 計算每個 goroutine 的數據範圍
			start := i * chunkSize
			end := start + chunkSize
			if i == numGoroutines-1 {
				end = n // 確保最後一個處理所有剩餘數據
			}

			go func(id, start, end int) {
				defer wg.Done()

				// 每個 goroutine 計算自己的部分和
				var localSum, localSumSq float64
				for j := start; j < end; j++ {
					diff := data1Slice[j].(float64) - data2Slice[j].(float64)
					localSum += diff
					localSumSq += diff * diff
				}

				// 保存到對應的結果陣列
				sums[id] = localSum
				sumSqs[id] = localSumSq
			}(i, start, end)
		}

		// 等待所有 goroutine 完成
		wg.Wait()

		// 合併所有 goroutine 的結果
		for i := range numGoroutines {
			sum += sums[i]
			sumSq += sumSqs[i]
		}
	} else {
		// 對小型數據集使用順序處理
		for i := range n {
			diff := data1Slice[i].(float64) - data2Slice[i].(float64)
			sum += diff
			sumSq += diff * diff
		}
	}

	// 計算統計量（與原始代碼相同）
	nFloat := float64(n)
	meanDiff := sum / nFloat
	variance := (sumSq - sum*sum/nFloat) / (nFloat - 1)
	stddevDiff := math.Sqrt(variance)
	standardError := stddevDiff / math.Sqrt(nFloat)
	tValue := meanDiff / standardError
	df := nFloat - 1
	pValue := calculateTPValue(tValue, df)

	// Handle optional confidence level parameter
	var cl float64
	if len(confidenceLevel) > 0 {
		cl = confidenceLevel[0]
	} else {
		cl = defaultConfidenceLevel
	}

	if cl <= 0 || cl >= 1 {
		cl = defaultConfidenceLevel
	}

	tDist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: df}
	tCritical := tDist.Quantile(1 - (1-cl)/2)
	marginOfError := tCritical * standardError

	ci := &[2]float64{meanDiff - marginOfError, meanDiff + marginOfError}

	effectSize := math.Abs(meanDiff) / stddevDiff
	effectSizes := []EffectSizeEntry{
		{Type: "cohen_d", Value: effectSize},
	}

	return &TTestResult{
		testResultBase: testResultBase{
			Statistic:   tValue,
			PValue:      pValue,
			DF:          &df,
			CI:          ci,
			EffectSizes: effectSizes,
		},
		MeanDiff: &meanDiff,
		N:        n,
	}
}

func calculateTPValue(tValue float64, df float64) float64 {
	if df <= 0 {
		return 1.0
	}

	tDist := distuv.StudentsT{
		Mu:    0,
		Sigma: 1,
		Nu:    df,
	}

	tAbs := math.Abs(tValue)
	cdfValue := tDist.CDF(tAbs)
	return 2 * (1 - cdfValue)
}
