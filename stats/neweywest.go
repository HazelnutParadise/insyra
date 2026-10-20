package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// NeweyWestResult is the result of NeweyWestMeanTest.
type NeweyWestResult struct {
	TestResult         // Statistic is the t value, DF is N-1
	Mean       float64 // sample mean
	StdErr     float64 // Newey-West (HAC) standard error of the mean
	Lag        int     // the lag used, after any automatic choice
	N          int     // number of observations
}

// AutoLag, passed as a lag, asks for NeweyWestLag(n). Any negative lag means
// the same; 0 and above are used as given, and 0 is a literal lag of 0.
const AutoLag = -1

// NeweyWestLag returns the lag NeweyWestMeanTest uses when it is asked to
// choose one: floor(4 * (n/100)^(2/9)), the rule of thumb Newey and West (1994)
// give for the Bartlett kernel.
func NeweyWestLag(n int) int {
	if n <= 0 {
		return 0
	}
	return int(math.Floor(4 * math.Pow(float64(n)/100, 2.0/9.0)))
}

// NeweyWestMeanTest tests whether the mean of a series is zero when the series
// is autocorrelated, for example daily returns of a position held 20 days, where
// consecutive values share 19 days. The standard error is the Newey-West
// heteroskedasticity-and-autocorrelation-consistent one with Bartlett weights
// 1 - j/(lag+1), and the statistic is mean / standard error.
//
// lag is the number of autocovariances to include. A negative lag
// (AutoLag) asks for NeweyWestLag(n); 0 and above are used as given. A lag of 0 gives the one-sample t-test of mean zero whose
// variance divides by n instead of n-1. The lag must be smaller than the
// number of observations.
//
// The p-value is two-tailed from the t distribution with n-1 degrees of
// freedom, as R's lmtest::coeftest reports it for an intercept-only model. No
// small-sample correction is applied to the variance, which matches
// sandwich::NeweyWest(prewhite = FALSE, adjust = FALSE) and statsmodels'
// cov_type="HAC" with use_correction=False; statsmodels reads the p-value from
// the normal distribution instead.
func NeweyWestMeanTest(series insyra.IDataList, lag int) (*NeweyWestResult, error) {
	values, err := testSeries(series, "series")
	if err != nil {
		return nil, err
	}
	n := len(values)
	if n < 2 {
		return nil, errors.New("sample size too small")
	}
	if lag < 0 {
		lag = NeweyWestLag(n)
	}
	if lag >= n {
		return nil, fmt.Errorf("lag %d must be smaller than the number of observations %d", lag, n)
	}

	mean := meanOfF64(values)
	longRun := 0.0
	for j := 0; j <= lag; j++ {
		gamma := 0.0
		for t := j; t < n; t++ {
			gamma += (values[t] - mean) * (values[t-j] - mean)
		}
		gamma /= float64(n)
		if j == 0 {
			longRun += gamma
			continue
		}
		longRun += 2 * (1 - float64(j)/float64(lag+1)) * gamma
	}

	df := float64(n - 1)
	result := &NeweyWestResult{Mean: mean, Lag: lag, N: n}
	result.DF = &df
	if longRun <= 0 {
		// A constant series has no variance: the arithmetic gives ±Inf or NaN.
		result.StdErr = 0
		if mean == 0 {
			result.Statistic, result.PValue = math.NaN(), math.NaN()
		} else {
			result.Statistic, result.PValue = math.Copysign(math.Inf(1), mean), 0
		}
		return result, nil
	}
	result.StdErr = math.Sqrt(longRun / float64(n))
	result.Statistic = mean / result.StdErr
	result.PValue = tTwoTailedPValue(result.Statistic, df)
	return result, nil
}
