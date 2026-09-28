// ttest.go

package stats

import (
	"errors"
	"math"

	"github.com/HazelnutParadise/insyra"
	"gonum.org/v1/gonum/stat"
)

// TTestOptions holds the settings SingleSampleTTest, TwoSampleTTest and
// PairedTTest take. The zero value is a two-sided test with a 95%
// confidence interval.
type TTestOptions struct {
	// Alternative is TwoSided, Greater or Less. Greater tests whether the
	// mean (or the difference of means, or the mean difference) is above
	// the hypothesised value. Empty means TwoSided.
	Alternative AlternativeHypothesis
	// ConfidenceLevel is the level of the confidence interval, strictly
	// between 0 and 1. Zero means 0.95.
	ConfidenceLevel float64
}

type TTestResult struct {
	TestResult
	Mean     float64  // mean of data (one-sample) or data1
	Mean2    *float64 // mean of data2; nil for the one-sample test
	MeanDiff *float64 // mean of data1 − data2 over the pairs; nil except for the paired test
	N        int      // size of data or data1; the number of pairs for the paired test
	N2       *int     // size of data2; nil for the one-sample test
}

// SingleSampleTTest performs a one-sample t-test comparing the sample mean to a known population mean.
// Parameters:
//   - data: The sample data to test
//   - mu: The hypothesized population mean to compare against
//   - opts: (Optional) A single TTestOptions holding the alternative hypothesis
//     and the confidence level of the confidence interval. At most one may be
//     given; its zero value means a two-sided test at a 95% confidence level.
//     A one-sided alternative gives R's t.test p-value and a one-sided bound
//     whose other end is +Inf (Greater) or -Inf (Less).
//
// Constant data has no variance, so the t statistic is ±Inf when the mean
// differs from mu and NaN when it equals it, with a p-value that follows the
// alternative (0 or 1 at ±Inf, NaN when the mean equals mu) to match. That is
// the arithmetic, not a failure, and no error is returned — check the
// statistic before reporting it.
//
// The effect sizes carry their sign, unlike the z-tests', which report the
// absolute value to match the R output their reference tests are pinned to.
//
// ** Verified using R **
func SingleSampleTTest(data insyra.IDataList, mu float64, opts ...TTestOptions) (*TTestResult, error) {
	o, err := oneOptions(opts)
	if err != nil {
		return nil, err
	}
	alt, cl, err := resolveTestSettings(o.Alternative, o.ConfidenceLevel)
	if err != nil {
		return nil, err
	}

	values, err := testSeries(data, "data")
	if err != nil {
		return nil, err
	}
	n := len(values)
	if n <= 1 {
		return nil, errors.New("sample size too small")
	}
	mean := meanOfF64(values)
	stddev := math.Sqrt(sampleVarianceF64(values))

	standardError := sampleSE(stddev, float64(n))
	tValue := (mean - mu) / standardError
	df := float64(n - 1)
	pValue := tPValue(tValue, df, alt)

	var marginOfError float64
	if alt == TwoSided {
		marginOfError = tMarginOfError(cl, df, standardError)
	} else {
		marginOfError = tMarginOfErrorOneSided(cl, df, standardError)
	}
	ci := ciByAlternative(mean, marginOfError, alt)

	// Handle constant data (stddev == 0)
	if stddev == 0 {
		if mean == mu {
			effectSize := 0.0
			tValue = math.NaN()
			pValue = math.NaN()
			effectSizes := cohenDEffectSizes(effectSize)
			return &TTestResult{
				TestResult: TestResult{
					Statistic:   tValue,
					PValue:      pValue,
					DF:          &df,
					CI:          ci,
					EffectSizes: effectSizes,
				},
				Mean: mean,
				N:    n,
			}, nil

		} else {
			effectSize := math.Inf(int(math.Copysign(1, mean-mu)))
			tValue = math.Inf(int(math.Copysign(1, mean-mu)))
			pValue = tPValue(tValue, df, alt)
			effectSizes := cohenDEffectSizes(effectSize)
			return &TTestResult{
				TestResult: TestResult{
					Statistic:   tValue,
					PValue:      pValue,
					DF:          &df,
					CI:          ci,
					EffectSizes: effectSizes,
				},
				Mean: mean,
				N:    n,
			}, nil
		}
	}

	effectSize := (mean - mu) / stddev //Preserve the sign
	effectSizes := cohenDEffectSizes(effectSize)

	return &TTestResult{
		TestResult: TestResult{
			Statistic:   tValue,
			PValue:      pValue,
			DF:          &df,
			CI:          ci,
			EffectSizes: effectSizes,
		},
		Mean: mean,
		N:    n,
	}, nil
}

// TwoSampleTTest performs a two-sample t-test comparing the means of two independent groups.
// Parameters:
//   - data1, data2: The two data groups to compare
//   - equalVariance: Whether to assume equal variances between groups
//   - opts: (Optional) A single TTestOptions holding the alternative hypothesis
//     and the confidence level of the confidence interval. At most one may be
//     given; its zero value means a two-sided test at a 95% confidence level.
//     A one-sided alternative gives R's t.test p-value and a one-sided bound on
//     mean1 - mean2 whose other end is +Inf (Greater) or -Inf (Less).
//
// ** Verified using R **
func TwoSampleTTest(data1, data2 insyra.IDataList, equalVariance bool, opts ...TTestOptions) (*TTestResult, error) {
	o, err := oneOptions(opts)
	if err != nil {
		return nil, err
	}
	alt, cl, err := resolveTestSettings(o.Alternative, o.ConfidenceLevel)
	if err != nil {
		return nil, err
	}

	values1, values2, err := testSeriesPair(data1, data2, "data1", "data2")
	if err != nil {
		return nil, err
	}
	n1, n2 := len(values1), len(values2)
	if n1 <= 1 || n2 <= 1 {
		return nil, errors.New("sample sizes too small")
	}
	mean1, mean2 := meanOfF64(values1), meanOfF64(values2)
	stddev1 := math.Sqrt(sampleVarianceF64(values1))
	stddev2 := math.Sqrt(sampleVarianceF64(values2))

	meanDiff := mean1 - mean2

	n1Float := float64(n1)
	n2Float := float64(n2)
	var1 := stddev1 * stddev1
	var2 := stddev2 * stddev2

	var standardError float64
	var df float64

	pooledVar := math.NaN()
	if equalVariance {
		standardError, pooledVar = pooledSE(var1, var2, n1Float, n2Float)
		df = float64(n1 + n2 - 2)
	} else {
		standardError = twoSampleSE(var1, var2, n1Float, n2Float)
		df = welchDF(var1, var2, n1Float, n2Float)
	}
	tValue := meanDiff / standardError
	pValue := tPValue(tValue, df, alt)

	var marginOfError float64
	if alt == TwoSided {
		marginOfError = tMarginOfError(cl, df, standardError)
	} else {
		marginOfError = tMarginOfErrorOneSided(cl, df, standardError)
	}
	ci := ciByAlternative(meanDiff, marginOfError, alt)

	var effectSize float64
	if equalVariance {
		effectSize = meanDiff / math.Sqrt(pooledVar) // Preserve the sign
	} else {
		effectSize = meanDiff / math.Sqrt((var1+var2)/2) // Preserve the sign
	}

	effectSizes := cohenDEffectSizes(effectSize)

	return &TTestResult{
		TestResult: TestResult{
			Statistic:   tValue,
			PValue:      pValue,
			DF:          &df,
			CI:          ci,
			EffectSizes: effectSizes,
		},
		Mean:  mean1,
		Mean2: &mean2,
		N:     n1,
		N2:    &n2,
	}, nil
}

// PairedTTest performs a paired-samples t-test comparing the means of two related groups.
// The data must be paired observations (same subjects measured twice).
// Parameters:
//   - data1, data2: The paired data groups to compare (must have same length)
//   - opts: (Optional) A single TTestOptions holding the alternative hypothesis
//     and the confidence level of the confidence interval. At most one may be
//     given; its zero value means a two-sided test at a 95% confidence level.
//     A one-sided alternative gives R's t.test p-value and a one-sided bound on
//     the mean difference whose other end is +Inf (Greater) or -Inf (Less).
//
// ** Verified using R **
func PairedTTest(data1, data2 insyra.IDataList, opts ...TTestOptions) (*TTestResult, error) {
	o, err := oneOptions(opts)
	if err != nil {
		return nil, err
	}
	alt, cl, err := resolveTestSettings(o.Alternative, o.ConfidenceLevel)
	if err != nil {
		return nil, err
	}

	var n int
	var data1Slice, data2Slice []any
	dl1 := asDataList(data1)
	dl2 := asDataList(data2)
	insyra.AtomicDoAll(func() {
		n = dl1.Len()
		if n != dl2.Len() || n <= 1 {
			err = errors.New("paired samples must have the same non-zero length")
			return
		}

		data1Slice = dl1.Data()
		data2Slice = dl2.Data()
	}, dl1, dl2)
	if err != nil {
		return nil, err
	}

	// Compute paired-difference mean & sample variance via gonum's two-pass
	// algorithm (numerically stable; replaces the previous parallel naive
	// (sumSq - sum²/n)/(n-1) one-pass formula which suffers catastrophic
	// cancellation when |meanDiff| is small relative to data magnitude).
	values1, err := numericValues(data1Slice, "data1")
	if err != nil {
		return nil, err
	}
	values2, err := numericValues(data2Slice, "data2")
	if err != nil {
		return nil, err
	}
	diffs := make([]float64, n)
	for i := range n {
		diffs[i] = values1[i] - values2[i]
	}
	meanDiff, varDiff := stat.MeanVariance(diffs, nil)
	stddevDiff := math.Sqrt(varDiff)

	// Each group mean and the pair count, so the paired result carries the
	// same two means and sizes the two-sample one does.
	mean1 := meanOfF64(values1)
	mean2 := meanOfF64(values2)
	n2 := n

	nFloat := float64(n)
	standardError := sampleSE(stddevDiff, nFloat)
	tValue := meanDiff / standardError
	df := nFloat - 1
	pValue := tPValue(tValue, df, alt)

	var marginOfError float64
	if alt == TwoSided {
		marginOfError = tMarginOfError(cl, df, standardError)
	} else {
		marginOfError = tMarginOfErrorOneSided(cl, df, standardError)
	}
	ci := ciByAlternative(meanDiff, marginOfError, alt)

	// Cohen's d_z for paired data, sign-preserving (matches single & two-sample
	// Cohen's d in this same file — previously this used math.Abs which
	// dropped direction-of-effect information for paired tests only).
	effectSize := meanDiff / stddevDiff
	effectSizes := cohenDEffectSizes(effectSize)

	return &TTestResult{
		TestResult: TestResult{
			Statistic:   tValue,
			PValue:      pValue,
			DF:          &df,
			CI:          ci,
			EffectSizes: effectSizes,
		},
		Mean:     mean1,
		Mean2:    &mean2,
		MeanDiff: &meanDiff,
		N:        n,
		N2:       &n2,
	}, nil
}
