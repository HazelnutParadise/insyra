package stats

import (
	"math"

	"github.com/HazelnutParadise/insyra"
)

type ZTestResult struct {
	testResultBase
	Mean  float64  // mean of the first group (or the only group)
	Mean2 *float64 // mean of the second group (nil if not applicable)
	N     int      // sample size of the first group (or the only group)
	N2    *int     // sample size of the second group (nil if not applicable)
}

func SingleSampleZTest(data insyra.IDataList, mu float64, sigma float64, alternative AlternativeHypothesis, confidenceLevel float64) *ZTestResult {
	var n int
	var mean float64
	isFailed := false
	data.AtomicDo(func(dl *insyra.DataList) {
		n = dl.Len()
		if n <= 0 {
			insyra.LogWarning("stats", "SingleSampleZTest", "Sample size too small")
			isFailed = true
			return
		}

		mean = dl.Mean()
	})
	if isFailed {
		return nil
	}

	standardError := sigma / math.Sqrt(float64(n))
	zValue := (mean - mu) / standardError
	pValue := calculateZPValue(zValue, alternative)

	if !(confidenceLevel > 0 && confidenceLevel < 1) {
		confidenceLevel = defaultConfidenceLevel
	}

	// 將重複使用的計算結果提前儲存
	zCritical := norm.Quantile(1 - (1-confidenceLevel)/2)

	marginOfError := zCritical * standardError
	var lowerCI, upperCI float64
	switch alternative {
	case TwoSided:
		lowerCI = mean - marginOfError
		upperCI = mean + marginOfError
	case Greater:
		lowerCI = mean - marginOfError
		upperCI = math.Inf(1)
	case Less:
		lowerCI = math.Inf(-1)
		upperCI = mean + marginOfError
	}

	effectSize := math.Abs(mean-mu) / sigma
	effectSizes := []EffectSizeEntry{
		{Type: "cohen_d", Value: effectSize},
	}
	ci := &[2]float64{lowerCI, upperCI}

	return &ZTestResult{
		testResultBase: testResultBase{
			Statistic:   zValue,
			PValue:      pValue,
			DF:          nil,
			CI:          ci,
			EffectSizes: effectSizes,
		},
		Mean:  mean,
		Mean2: nil,
		N:     n,
		N2:    nil,
	}
}

func TwoSampleZTest(data1, data2 insyra.IDataList, sigma1, sigma2 float64, alternative AlternativeHypothesis, confidenceLevel float64) *ZTestResult {
	var n1, n2 int
	var mean1, mean2 float64
	isFailed := false
	data1.AtomicDo(func(dl1 *insyra.DataList) {
		data2.AtomicDo(func(dl2 *insyra.DataList) {
			n1 = dl1.Len()
			n2 = dl2.Len()
			if n1 <= 0 || n2 <= 0 {
				insyra.LogWarning("stats", "TwoSampleZTest", "Sample sizes too small")
				isFailed = true
				return
			}

			mean1 = dl1.Mean()
			mean2 = dl2.Mean()
		})
	})
	if isFailed {
		return nil
	}

	meanDiff := mean1 - mean2

	// 避免重複計算
	n1Float := float64(n1)
	n2Float := float64(n2)
	sigma1Sq := sigma1 * sigma1
	sigma2Sq := sigma2 * sigma2

	standardError := math.Sqrt((sigma1Sq / n1Float) + (sigma2Sq / n2Float))
	zValue := meanDiff / standardError
	pValue := calculateZPValue(zValue, alternative)

	if !(confidenceLevel > 0 && confidenceLevel < 1) {
		confidenceLevel = defaultConfidenceLevel
	}

	// 將重複使用的計算結果提前儲存
	zCritical := norm.Quantile(1 - (1-confidenceLevel)/2)

	marginOfError := zCritical * standardError
	var lowerCI, upperCI float64
	switch alternative {
	case TwoSided:
		lowerCI = meanDiff - marginOfError
		upperCI = meanDiff + marginOfError
	case Greater:
		lowerCI = meanDiff - marginOfError
		upperCI = math.Inf(1)
	case Less:
		lowerCI = math.Inf(-1)
		upperCI = meanDiff + marginOfError
	}

	pooledSigma := math.Sqrt((n1Float*sigma1Sq + n2Float*sigma2Sq) / (n1Float + n2Float))
	effectSize := math.Abs(meanDiff) / pooledSigma
	effectSizes := []EffectSizeEntry{
		{Type: "cohen_d", Value: effectSize},
	}
	ci := &[2]float64{lowerCI, upperCI}

	return &ZTestResult{
		testResultBase: testResultBase{
			Statistic:   zValue,
			PValue:      pValue,
			DF:          nil,
			CI:          ci,
			EffectSizes: effectSizes,
		},
		Mean:  mean1,
		Mean2: &mean2,
		N:     n1,
		N2:    &n2,
	}
}

func calculateZPValue(zValue float64, alternative AlternativeHypothesis) float64 {
	// 重複使用同一個分佈物件
	zAbs := math.Abs(zValue)

	switch alternative {
	case TwoSided:
		// 只需要計算一次CDF
		cdfVal := norm.CDF(zAbs)
		return 2 * (1 - cdfVal)
	case Greater:
		return 1 - norm.CDF(zValue)
	case Less:
		return norm.CDF(zValue)
	default:
		return math.NaN()
	}
}
