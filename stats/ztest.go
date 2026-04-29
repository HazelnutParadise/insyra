package stats

import (
	"errors"
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

func SingleSampleZTest(data insyra.IDataList, mu float64, sigma float64, alternative AlternativeHypothesis, confidenceLevel float64) (*ZTestResult, error) {
	if sigma <= 0 {
		return nil, errors.New("sigma must be greater than zero")
	}
	if alternative != TwoSided && alternative != Greater && alternative != Less {
		return nil, errors.New("unsupported alternative hypothesis")
	}
	if confidenceLevel <= 0 || confidenceLevel >= 1 {
		return nil, errors.New("confidenceLevel must be between 0 and 1")
	}

	var n int
	var mean float64
	var err error
	data.AtomicDo(func(dl *insyra.DataList) {
		n = dl.Len()
		if n <= 0 {
			err = errors.New("sample size too small")
			return
		}

		mean = dl.Mean()
	})
	if err != nil {
		return nil, err
	}

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

func TwoSampleZTest(data1, data2 insyra.IDataList, sigma1, sigma2 float64, alternative AlternativeHypothesis, confidenceLevel float64) (*ZTestResult, error) {
	if sigma1 <= 0 || sigma2 <= 0 {
		return nil, errors.New("sigma1 and sigma2 must be greater than zero")
	}
	if alternative != TwoSided && alternative != Greater && alternative != Less {
		return nil, errors.New("unsupported alternative hypothesis")
	}
	if confidenceLevel <= 0 || confidenceLevel >= 1 {
		return nil, errors.New("confidenceLevel must be between 0 and 1")
	}

	var n1, n2 int
	var mean1, mean2 float64
	var err error
	data1.AtomicDo(func(dl1 *insyra.DataList) {
		data2.AtomicDo(func(dl2 *insyra.DataList) {
			n1 = dl1.Len()
			n2 = dl2.Len()
			if n1 <= 0 || n2 <= 0 {
				err = errors.New("sample sizes too small")
				return
			}

			mean1 = dl1.Mean()
			mean2 = dl2.Mean()
		})
	})
	if err != nil {
		return nil, err
	}

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

	pooledSigma := math.Sqrt((n1Float*sigma1Sq + n2Float*sigma2Sq) / (n1Float + n2Float))
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
