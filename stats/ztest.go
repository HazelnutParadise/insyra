package stats

import (
	"errors"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// ZTestOptions holds the settings SingleSampleZTest and TwoSampleZTest
// take. The zero value is a two-sided test with a 95% confidence interval.
type ZTestOptions struct {
	// Alternative is TwoSided, Greater or Less. Empty means TwoSided.
	Alternative AlternativeHypothesis
	// ConfidenceLevel is the level of the confidence interval, strictly
	// between 0 and 1. Zero means 0.95.
	ConfidenceLevel float64
}

type ZTestResult struct {
	testResultBase
	Mean  float64  // mean of the first group (or the only group)
	Mean2 *float64 // mean of the second group (nil if not applicable)
	N     int      // sample size of the first group (or the only group)
	N2    *int     // sample size of the second group (nil if not applicable)
}

// SingleSampleZTest tests whether the mean of a single sample differs from
// mu, with the population standard deviation sigma known. mu and sigma are
// required; opts is an optional ZTestOptions and at most one may be given,
// its zero value meaning a two-sided test at a 95% confidence level.
//
// The effect size is reported as an absolute value, so it carries no
// direction. That is deliberate: the R comparison these tests are checked
// against reports |d|, and the t-tests keep the sign.
func SingleSampleZTest(data insyra.IDataList, mu float64, sigma float64, opts ...ZTestOptions) (*ZTestResult, error) {
	if sigma <= 0 {
		return nil, errors.New("sigma must be greater than zero")
	}
	o, err := oneOptions(opts)
	if err != nil {
		return nil, err
	}
	alternative, confidenceLevel, err := resolveTestSettings(o.Alternative, o.ConfidenceLevel)
	if err != nil {
		return nil, err
	}

	values, err := testSeries(data, "data")
	if err != nil {
		return nil, err
	}
	n := len(values)
	if n <= 0 {
		return nil, errors.New("sample size too small")
	}
	mean := meanOfF64(values)

	standardError := sampleSE(sigma, float64(n))
	zValue := (mean - mu) / standardError
	pValue := zPValue(zValue, alternative)

	var marginOfError float64
	if alternative == TwoSided {
		marginOfError = zMarginOfError(confidenceLevel, standardError)
	} else {
		marginOfError = zMarginOfErrorOneSided(confidenceLevel, standardError)
	}

	effectSize := math.Abs(mean-mu) / sigma
	effectSizes := cohenDEffectSizes(effectSize)
	ci := ciByAlternative(mean, marginOfError, alternative)

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
	}, nil
}

// TwoSampleZTest tests whether the means of two independent samples differ,
// with the population standard deviations sigma1 and sigma2 known. sigma1
// and sigma2 are required; opts is an optional ZTestOptions and at most one
// may be given, its zero value meaning a two-sided test at a 95% confidence
// level.
//
// The effect size is reported as an absolute value, so it carries no
// direction. That is deliberate: the R comparison these tests are checked
// against reports |d|, and the t-tests keep the sign.
func TwoSampleZTest(data1, data2 insyra.IDataList, sigma1, sigma2 float64, opts ...ZTestOptions) (*ZTestResult, error) {
	if sigma1 <= 0 || sigma2 <= 0 {
		return nil, errors.New("sigma1 and sigma2 must be greater than zero")
	}
	o, err := oneOptions(opts)
	if err != nil {
		return nil, err
	}
	alternative, confidenceLevel, err := resolveTestSettings(o.Alternative, o.ConfidenceLevel)
	if err != nil {
		return nil, err
	}

	values1, values2, err := testSeriesPair(data1, data2, "data1", "data2")
	if err != nil {
		return nil, err
	}
	n1, n2 := len(values1), len(values2)
	if n1 <= 0 || n2 <= 0 {
		return nil, errors.New("sample sizes too small")
	}
	mean1, mean2 := meanOfF64(values1), meanOfF64(values2)

	meanDiff := mean1 - mean2

	n1Float := float64(n1)
	n2Float := float64(n2)
	sigma1Sq := sigma1 * sigma1
	sigma2Sq := sigma2 * sigma2

	standardError := twoSampleSE(sigma1Sq, sigma2Sq, n1Float, n2Float)
	zValue := meanDiff / standardError
	pValue := zPValue(zValue, alternative)

	var marginOfError float64
	if alternative == TwoSided {
		marginOfError = zMarginOfError(confidenceLevel, standardError)
	} else {
		marginOfError = zMarginOfErrorOneSided(confidenceLevel, standardError)
	}

	// Cohen's d_av for two-sample z-test with known population sigmas:
	// the standard textbook formula (matches R's effectsize package). The
	// previous version weighted by sample size — n1·σ1² + n2·σ2² /(n1+n2) —
	// which has no published authority. Sample size is used to estimate the
	// mean, not to revise our knowledge of the (already known) population
	// dispersion.
	pooledSigma := math.Sqrt((sigma1Sq + sigma2Sq) / 2)
	effectSize := math.Abs(meanDiff) / pooledSigma
	effectSizes := cohenDEffectSizes(effectSize)
	ci := ciByAlternative(meanDiff, marginOfError, alternative)

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
	}, nil
}
