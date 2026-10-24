package stats

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// VarianceRatioResult is the result of VarianceRatio for one holding period q.
type VarianceRatioResult struct {
	Q  int     // the holding period, in observations
	VR float64 // the variance ratio VR(q); 1 under a random walk
	// ZHomoscedastic is the Lo-MacKinlay z statistic, valid when the
	// increments are independent with a constant variance; PHomoscedastic is
	// its two-tailed normal p-value.
	ZHomoscedastic, PHomoscedastic float64
	// ZRobust is the z statistic that stays valid under heteroskedastic
	// increments (Lo and MacKinlay's z*); PRobust is its two-tailed p-value.
	ZRobust, PRobust float64
}

// VarianceRatio tests whether returns follow a random walk by comparing the
// variance of q-period returns with q times the variance of one-period returns
// (Lo and MacKinlay 1988). A ratio above 1 means returns trend (positive
// autocorrelation), below 1 that they reverse.
//
// returns are single-period changes, which need not be stock returns: any
// series of per-period changes works (an exchange-rate change, a daily
// temperature difference). Pass log returns for a price, or first differences
// for a level series, never the price itself. Changes must add up over q
// periods, which log returns do, so q-period returns are their sums.
//
// It computes the unbiased estimators and the overlapping q-period sums of Lo
// and MacKinlay, with n the number of returns, and gives both the homoscedastic z and the heteroskedasticity-robust
// z*. Each q must be between 2 and n-1. The p-values come from the standard
// normal distribution, which is an asymptotic result and needs a long series.
func VarianceRatio(returns insyra.IDataList, qs ...int) ([]VarianceRatioResult, error) {
	if len(qs) == 0 {
		return nil, errors.New("at least one holding period q is needed")
	}
	x, err := testSeries(returns, "returns")
	if err != nil {
		return nil, err
	}
	n := len(x)
	for _, q := range qs {
		if q < 2 || q > n-1 {
			return nil, fmt.Errorf("q = %d must be between 2 and the number of returns minus 1 (%d)", q, n-1)
		}
	}

	nf := float64(n)
	mu := meanOfF64(x)
	d := make([]float64, n) // deviations from the mean
	var ss float64
	for i, v := range x {
		d[i] = v - mu
		ss += d[i] * d[i]
	}
	if ss == 0 || allEqualF64(x) {
		return nil, errors.New("returns have no variance")
	}
	varA := ss / (nf - 1)
	cum := make([]float64, n+1) // cum[i] = x[0] + ... + x[i-1]
	for i, v := range x {
		cum[i+1] = cum[i] + v
	}

	out := make([]VarianceRatioResult, len(qs))
	for k, q := range qs {
		qf := float64(q)
		var sq float64
		for t := q; t <= n; t++ {
			sum := cum[t] - cum[t-q] - qf*mu
			sq += sum * sum
		}
		m := qf * (nf - qf + 1) * (1 - qf/nf)
		vr := (sq / m) / varA

		phi := 2 * (2*qf - 1) * (qf - 1) / (3 * qf * nf)
		zHomo := (vr - 1) / math.Sqrt(phi)

		var theta float64
		for j := 1; j < q; j++ {
			var num float64
			for t := j; t < n; t++ {
				num += d[t] * d[t] * d[t-j] * d[t-j]
			}
			delta := nf * num / (ss * ss)
			w := 2 * (qf - float64(j)) / qf
			theta += w * w * delta
		}
		zRobust := (vr - 1) / math.Sqrt(theta/nf)

		out[k] = VarianceRatioResult{
			Q: q, VR: vr,
			ZHomoscedastic: zHomo, PHomoscedastic: zPValue(zHomo, TwoSided),
			ZRobust: zRobust, PRobust: zPValue(zRobust, TwoSided),
		}
	}
	return out, nil
}

// ChowDenningResult is the result of ChowDenning.
type ChowDenningResult struct {
	Qs []int
	// MaxZ is the largest absolute z over the holding periods and P the p-value
	// of that maximum, adjusted for having looked at several periods.
	MaxZ, P float64
}

// ChowDenning tests the random-walk hypothesis jointly over several holding
// periods, so that picking the q with the strongest result does not inflate the
// false-positive rate (Chow and Denning 1993). It takes the largest absolute z
// over the periods and adjusts its p-value with the Sidak correction
// 1 - (2*Phi(max) - 1)^k for k periods. Robust selects the heteroskedasticity-
// robust z* instead of the homoscedastic z.
func ChowDenning(returns insyra.IDataList, robust bool, qs ...int) (*ChowDenningResult, error) {
	results, err := VarianceRatio(returns, qs...)
	if err != nil {
		return nil, err
	}
	maxZ := 0.0
	for _, r := range results {
		z := r.ZHomoscedastic
		if robust {
			z = r.ZRobust
		}
		if a := math.Abs(z); a > maxZ || math.IsNaN(a) {
			maxZ = a
		}
	}
	k := float64(len(qs))
	p := 1 - math.Pow(2*zCDF(maxZ)-1, k)
	return &ChowDenningResult{Qs: append([]int(nil), qs...), MaxZ: maxZ, P: p}, nil
}
