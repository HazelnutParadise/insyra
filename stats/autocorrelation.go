package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// AutocorrelationResult is the autocorrelation of a series at one lag.
type AutocorrelationResult struct {
	Lag int     // the lag, in observations
	Rho float64 // the sample autocorrelation at Lag
	// T is sqrt(n)*Rho, which is standard normal when the series is white noise
	// (independent with a constant variance); P is its two-tailed normal p-value.
	T, P float64
	// TRobust is sqrt(n)*Rho/sqrt(delta), where delta is Lo and MacKinlay's
	// (1988) asymptotic variance of sqrt(n)*Rho. It stays valid when the
	// variance changes over time, as stock returns do; PRobust is its
	// two-tailed normal p-value.
	TRobust, PRobust float64
}

// Autocorrelation estimates the autocorrelation of series at lags 1 to
// maxLag. It asks whether a value is related to the one before it, or two, or
// k before it: for daily stock returns, a positive lag-1 value means a rise
// tends to follow a rise.
//
// Rho divides the sum of lagged products of deviations from the mean by the sum
// of squared deviations, as R's acf does. T and P test each lag against white
// noise. Volatility that comes in clusters makes T reject too often, so TRobust
// and PRobust use Lo and MacKinlay's heteroskedasticity-robust variance, the
// same one VarianceRatio uses. The p-values are asymptotic and need a long
// series. maxLag must be between 1 and n-1, and the series must not be
// constant. It fails when the robust variance at a lag is zero, which happens
// when every lagged product of deviations is zero, as in a sparse series.
func Autocorrelation(series insyra.IDataList, maxLag int) ([]AutocorrelationResult, error) {
	x, err := testSeries(series, "series")
	if err != nil {
		return nil, err
	}
	n := len(x)
	if maxLag < 1 || maxLag > n-1 {
		return nil, fmt.Errorf("maxLag = %d must be between 1 and the number of observations minus 1 (%d)", maxLag, n-1)
	}
	nf := float64(n)
	mu := meanOfF64(x)
	d := make([]float64, n)
	var ss float64
	for i, v := range x {
		d[i] = v - mu
		ss += d[i] * d[i]
	}
	if ss == 0 {
		return nil, errors.New("series has no variance")
	}

	out := make([]AutocorrelationResult, maxLag)
	for j := 1; j <= maxLag; j++ {
		var cross, sq float64
		for t := j; t < n; t++ {
			cross += d[t] * d[t-j]
			sq += d[t] * d[t] * d[t-j] * d[t-j]
		}
		rho := cross / ss
		t := math.Sqrt(nf) * rho
		delta := nf * sq / (ss * ss)
		if delta == 0 {
			return nil, fmt.Errorf("the robust variance is zero at lag %d: every lagged product of deviations is zero, so TRobust is undefined", j)
		}
		tr := t / math.Sqrt(delta)
		out[j-1] = AutocorrelationResult{
			Lag: j, Rho: rho,
			T: t, P: zPValue(t, TwoSided),
			TRobust: tr, PRobust: zPValue(tr, TwoSided),
		}
	}
	return out, nil
}
