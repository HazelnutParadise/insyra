package quant

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// volumeSeries reads a volume series, which must be finite and not negative.
func volumeSeries(dl insyra.IDataList) ([]float64, error) {
	volume, err := numericSeries(dl, "volume")
	if err != nil {
		return nil, err
	}
	for i, v := range volume {
		if v < 0 {
			return nil, fmt.Errorf("volume is negative at row index %d: %v", i, v)
		}
	}
	return volume, nil
}

func sign(x float64) float64 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

// OBV returns Granville's on-balance volume: the running sum of volume, added
// on a day the close rose and subtracted on a day it fell, with a flat day
// adding nothing. The first value is 0; the level of OBV is arbitrary and only
// its differences carry information. close and volume must have the same
// length, every value finite, and volume not negative.
//
// To use raw share volume with dividend-adjusted moves, where the raw close
// drops on an ex-dividend day without a real down move, call OBVFromReturns.
func OBV(close, volume insyra.IDataList) (*insyra.DataList, error) {
	prices, err := numericSeries(close, "close")
	if err != nil {
		return nil, err
	}
	vols, err := volumeSeries(volume)
	if err != nil {
		return nil, err
	}
	if len(prices) != len(vols) {
		return nil, fmt.Errorf("close has %d rows and volume has %d", len(prices), len(vols))
	}
	moves := make([]float64, len(prices))
	for t := 1; t < len(prices); t++ {
		moves[t] = prices[t] - prices[t-1]
	}
	return accumulateOBV(moves, vols), nil
}

// OBVFromReturns is OBV with the direction of each day taken from a return
// series instead of from the change in close, so a caller can pass
// dividend-adjusted returns together with raw share volume. The return at
// row index 0 has no earlier day in the data to compare with and is not used,
// so the first value is 0, as in OBV.
func OBVFromReturns(returns, volume insyra.IDataList) (*insyra.DataList, error) {
	rets, err := numericSeries(returns, "returns")
	if err != nil {
		return nil, err
	}
	vols, err := volumeSeries(volume)
	if err != nil {
		return nil, err
	}
	if len(rets) != len(vols) {
		return nil, fmt.Errorf("returns has %d rows and volume has %d", len(rets), len(vols))
	}
	return accumulateOBV(rets, vols), nil
}

// accumulateOBV sums sign(moves[t]) * volume[t] from row index 1 on.
func accumulateOBV(moves, volume []float64) *insyra.DataList {
	out := make([]any, len(moves))
	level := 0.0
	for t := range moves {
		if t > 0 {
			level += sign(moves[t]) * volume[t]
		}
		out[t] = level
	}
	return insyra.NewDataList(out...)
}

// SignedVolumeRatio returns, for each day, the sum of sign(return) * volume
// over the last n days divided by the sum of volume over the same days. It lies
// in [-1, 1] and has no unit, so it compares across stocks where the OBV slope
// does not. The first n-1 values, which lack a full window, and any window with
// no volume are NaN.
func SignedVolumeRatio(returns, volume insyra.IDataList, n int) (*insyra.DataList, error) {
	if n < 1 {
		return nil, errors.New("window n must be at least 1")
	}
	rets, err := numericSeries(returns, "returns")
	if err != nil {
		return nil, err
	}
	vols, err := volumeSeries(volume)
	if err != nil {
		return nil, err
	}
	if len(rets) != len(vols) {
		return nil, fmt.Errorf("returns has %d rows and volume has %d", len(rets), len(vols))
	}
	out := make([]any, len(rets))
	for t := range out {
		out[t] = math.NaN()
		if t < n-1 {
			continue
		}
		signed, total := 0.0, 0.0
		for i := t - n + 1; i <= t; i++ {
			signed += sign(rets[i]) * vols[i]
			total += vols[i]
		}
		if total > 0 {
			out[t] = signed / total
		}
	}
	return insyra.NewDataList(out...), nil
}

// Divergence flags the days a price makes a new n-day extreme that an
// indicator does not confirm. A day is -1 (bearish) when price is above its
// highest value over the previous n days while the indicator is below its
// highest over the same days, +1 (bullish) when price is below its lowest of
// the previous n days while the indicator is above its lowest, and 0
// otherwise. The first n rows have no full history and are 0.
//
// cooldown is the number of days after a flag of one kind during which that
// kind is not flagged again, so one episode is one event; 0 flags every day
// that qualifies.
//
// The indicator may start with NaN, as RSI and SignedVolumeRatio do before they
// have enough rows: a day whose indicator or any of the previous n values of it
// is NaN is 0, as the first n rows are. Price must be finite. An infinite
// indicator value is an error.
func Divergence(price, indicator insyra.IDataList, n, cooldown int) (*insyra.DataList, error) {
	if n < 1 {
		return nil, errors.New("window n must be at least 1")
	}
	if cooldown < 0 {
		return nil, errors.New("cooldown must not be negative")
	}
	p, err := numericSeries(price, "price")
	if err != nil {
		return nil, err
	}
	ind, err := numericSeriesAllowNaN(indicator, "indicator")
	if err != nil {
		return nil, err
	}
	if len(p) != len(ind) {
		return nil, fmt.Errorf("price has %d rows and indicator has %d", len(p), len(ind))
	}
	out := make([]any, len(p))
	lastBear, lastBull := -1<<30, -1<<30
	for t := range out {
		out[t] = 0
		if t < n {
			continue
		}
		if hasNaN(ind[t-n : t+1]) {
			continue
		}
		pMax, pMin := p[t-n], p[t-n]
		iMax, iMin := ind[t-n], ind[t-n]
		for i := t - n + 1; i < t; i++ {
			pMax, pMin = math.Max(pMax, p[i]), math.Min(pMin, p[i])
			iMax, iMin = math.Max(iMax, ind[i]), math.Min(iMin, ind[i])
		}
		switch {
		case p[t] > pMax && ind[t] < iMax && t-lastBear > cooldown:
			out[t], lastBear = -1, t
		case p[t] < pMin && ind[t] > iMin && t-lastBull > cooldown:
			out[t], lastBull = 1, t
		}
	}
	return insyra.NewDataList(out...), nil
}

func hasNaN(v []float64) bool {
	for _, x := range v {
		if math.IsNaN(x) {
			return true
		}
	}
	return false
}
