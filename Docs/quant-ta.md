# [ ta ] Package

This document describes the public API of `github.com/HazelnutParadise/insyra/quant/ta`, the technical-analysis tools of Insyra.

> **Technical analysis is bullshit.** The tools in this package are here so you can test that claim yourself, not because they make money. The rest of `quant` (performance and risk metrics, CAPM and factor models, options, portfolio optimization, bootstrap, walk-forward and overfitting diagnostics, event studies, signal buckets and backtests) is not technical analysis, and this warning does not apply to it.
>
> In our own study (October 2026), 17 common theories (Dow theory, moving averages, KD, RSI, MACD, Elliott waves, chart patterns and others) were checked on real data: Taiwan stocks (about 2,300, over 300 of them delisted), US stocks, and 47 national indexes, with trading costs deducted and the rules fixed before looking at the results. Scored only on 2015 to 2026, data the rules were not tuned on:
>
> - Following an indicator stock by stock beat buy-and-hold for 0% to 0.1% of Taiwan and US stocks, after costs and after correcting for luck.
> - Of 90-odd rules for moving in and out of the whole market, none beat buy-and-hold significantly (Taiwan after 2003, US after 1970).
> - Elliott waves, head and shoulders and triangles appear as often in shuffled prices as in real ones, and ascending triangles break down nearly 80% of the time in both, the opposite of the textbook.
> - The rules that did once work (US moving-average timing before 1970, for one) stopped working. None of the 17 theories was tested on a large sample when it was proposed.
>
> Prices are not perfectly random, but what is left is smaller than the cost of trading it or has already vanished. This is a test of history, not investment advice.
>
> **What in this package it covers, and what the study found:**
>
> | Functions | Theory | Finding |
> | --- | --- | --- |
> | `SMA`, `EMA` | Moving averages (均線), Granville's rules | Partly usable. A stock below a falling 200-day line that bounces and fails to get back above it went on to lag by 0.91% over the next 20 days in Taiwan, the only claim that beat the costs, and only on stocks that still exist. As a market-timing rule no moving-average rule beat buy-and-hold (US rules won every year until 1969 and none since 1970). |
> | `MACD` | MACD | Ineffective. It is the distance between two moving averages, so by the time it moves the price already has. |
> | `KD` | Stochastic (KD, KDJ) | Ineffective. In Taiwan, stocks that were "overbought" went on to do better, and it is the same data as RSI. |
> | `Bollinger`, `TrueRange`, `ATR`, `Keltner`, `KeltnerClassic`, `Donchian`, `SuperTrend`, `ParabolicSAR`, `Chandelier`, `Aroon`, `TRIX`, `KAMA`, `Ichimoku`, `CCI`, `Momentum`, `ROC`, `UltimateOscillator`, `StochRSI`, `ARBR`, `CR`, `ForceIndex`, `ElderRay`, `MFI`, `AccumulationDistribution`, `CMF`, `ChaikinOscillator`, `EaseOfMovement`, `NVI`, `PVI`, `VolumeProfile`, `PivotPoints`, `HeikinAshi`, `TDSequential`, `PointAndFigure` | Bollinger bands, channels, trailing stops, Aroon, TRIX, KAMA, Ichimoku, CCI, momentum, ultimate oscillator, stochastic RSI, AR/BR, CR, force index, Elder-ray, money flow index, accumulation/distribution, Chaikin, ease of movement, volume indexes, volume profile, pivot points, Heikin-Ashi, TD Sequential, point-and-figure | Not tested in the study (the study found volume indicators "description only": volume is a shadow of the price move). (KD and RSI, which the study did test, are rows above; StochRSI is built from the RSI.) |
> | `RSI` | RSI, overbought and oversold | Ineffective. In Taiwan, stocks with a high RSI did better afterwards, not worse. Trading it stock by stock lost to holding. |
> | `Bias`, `BiasZ` | Bias ratio (乖離率) | Only useful read backwards. Taiwan stocks continue for one to three months, they do not revert, and the gap does not pay the costs. |
> | `OBV`, `OBVFromReturns`, `SignedVolumeRatio`, `PriceDivergence` | On-balance volume, volume and price | Description only. Volume is a shadow of the price move, not the fuel for it. |
> | `AdvanceDeclineRatio`, `AdvanceDeclineLine`, `OverboughtOversold`, `NetAdvanceRatio` | Market breadth | Description only. A majority of advancers marks broad strength, not an overbought market, and the "triple confirmation" is one dataset. |
> | `ZigZag`, `Regimes` | Dow theory | Description only. Bull stretches are long and bear ones short, and pullbacks are common, but mechanical Dow trading did not beat holding. |

---

## Installation

```bash
go get github.com/HazelnutParadise/insyra/quant/ta
```

---

## Overview

- **Moving averages and bands**: `SMA`, `EMA`, `MACD`, `Bollinger`, `TrueRange`, `ATR`, `KD`
- **Price structure**: `PivotPoints`, `HeikinAshi`, `TDSequential`, `PointAndFigure`
- **Money flow and volume profile**: `MFI`, `AccumulationDistribution`, `CMF`, `ChaikinOscillator`, `EaseOfMovement`, `NVI`, `PVI`, `VolumeProfile`
- **Oscillators and momentum**: `CCI`, `Momentum`, `ROC`, `UltimateOscillator`, `StochRSI`, `ARBR`, `CR`, `ForceIndex`, `ElderRay`
- **Trend**: `Aroon`, `TRIX`, `KAMA`, `Ichimoku`
- **Channels and stops**: `Keltner`, `KeltnerClassic`, `Donchian`, `SuperTrend`, `ParabolicSAR`, `Chandelier`
- **Oscillators and strength**: `RSI`, `Bias`, `BiasZ`
- **Volume**: `OBV`, `OBVFromReturns`, `SignedVolumeRatio`, `PriceDivergence`
- **Market breadth**: `AdvanceDeclineRatio`, `AdvanceDeclineLine`, `OverboughtOversold`, `NetAdvanceRatio`
- **Turning points and regimes**: `ZigZag`, `Regimes`

**Conventions.** Every function takes its series as `insyra.IDataList` and returns an `error` for invalid input, naming the row index (counted from 0). The tools use only information known at the close of each row, so a series cut short gives the results of the full series for the rows before the cut. A row without enough history is `NaN`. They were first part of `quant`, which keeps the evaluation tools (performance and risk metrics, event studies, signal buckets, backtests); see [`quant`](./quant.md).

---

## Volume Indicators

All four take series of the same length and return a `*insyra.DataList` of the same length. They refuse a `NaN`, an infinity, a non-number, a nil series and a length mismatch, as the rest of the package does, and a negative volume.

### OBV and OBVFromReturns

```go
func OBV(close, volume insyra.IDataList) (*insyra.DataList, error)
func OBVFromReturns(returns, volume insyra.IDataList) (*insyra.DataList, error)
```

Granville's on-balance volume: `OBV[t] = OBV[t-1] + sign(change[t]) * volume[t]`, where a flat day adds nothing. `OBV` takes the change from the close; `OBVFromReturns` takes the direction from a return series, so you can pass dividend-adjusted returns with raw share volume (on an ex-dividend day the raw close falls without a real down move). The first value is `0`; the level of OBV is arbitrary and only its differences carry information.

### SignedVolumeRatio

```go
func SignedVolumeRatio(returns, volume insyra.IDataList, n int) (*insyra.DataList, error)
```

For each day, `sum(sign(return) * volume) / sum(volume)` over the last `n` days. The value lies in [-1, 1] and has no unit, so it compares across stocks where the OBV slope does not. The first `n-1` values and any window with no volume are `NaN`.

### PriceDivergence

```go
func PriceDivergence(price, indicator insyra.IDataList, n, cooldown int) (*insyra.DataList, error)
```

Flags days when a price makes a new `n`-day extreme that an indicator does not confirm: `-1` (bearish) when price is above its highest value of the previous `n` days while the indicator is below its highest, `+1` (bullish) for the mirror image on lows, `0` otherwise. The first `n` rows are `0`. After a flag, the same kind is not flagged again for `cooldown` days, so one episode is one event; `cooldown` 0 flags every qualifying day. The indicator may start with `NaN`, as `RSI` and `SignedVolumeRatio` do: a day whose indicator, or any of the previous `n` values of it, is `NaN` is `0`. Price must be finite, and an infinite indicator value is an error.

```go
obv, err := ta.OBV(close, volume)
flags, err := ta.PriceDivergence(close, obv, 20, 10)

rsi, err := ta.RSI(close, 14)
flags, err = ta.PriceDivergence(close, rsi, 20, 10) // the first rows of rsi are NaN
```

---

### Bias and BiasZ

```go
func Bias(prices insyra.IDataList, n int) (*insyra.DataList, error)
func BiasZ(prices insyra.IDataList, n, window int) (*insyra.DataList, error)
```

`Bias` is `price / SMA(n) - 1`, `NaN` for the first `n - 1` rows. `BiasZ` divides the bias by the standard deviation of the previous `window` biases, never including the current day's own, so nothing is seen ahead of time; it is `NaN` until enough earlier biases exist.

```go
bias, _ := ta.Bias(close, 20)
buckets, err := ta.SignalBuckets(biasTable, closes, dates, ta.BucketOptions{
    ForwardOptions: ta.ForwardOptions{Horizons: []int{1, 5, 20, 60}},
    Edges:          []float64{-0.2, -0.1, 0, 0.1, 0.2},
})
```

---

## Turning Points and Regimes

### ZigZag

```go
func ZigZag(prices insyra.IDataList, threshold float64) ([]Swing, error)

type Swing struct {
    Kind        SwingKind // SwingLow or SwingHigh
    Index       int       // row of the extreme
    Price       float64
    ConfirmedAt int       // first row on which the price was `threshold` away from the extreme
}
```

Finds the turning points of a positive price series that reverse by at least `threshold`, a fraction such as `0.10` for 10%. Trade on `ConfirmedAt`, the earliest the swing could be known, and describe the series with `Index`.

It is causal: a swing is reported at the row that confirms it and uses no later price, so recomputing on a series cut short returns exactly the swings of the full series whose `ConfirmedAt` lies before the cut (the tests check this on random series). Swings alternate between lows and highs, and the extreme still being tracked at the end is not reported. For `100, 111, 120, 105, 95, 100, 115, 130, 116` with a threshold of 0.10 the swings are low at 0 confirmed at 1, high at 2 confirmed at 3, low at 4 confirmed at 6, high at 7 confirmed at 8.

### Regimes

```go
func Regimes(swings []Swing, opts ...RegimeOptions) ([]Regime, error)
```

Turns the swings into bull stretches (low to the next high) and bear stretches (high to the next low). Each `Regime` has `Kind` (`RegimeBull` or `RegimeBear`), `Start` and `End` rows, `Length`, `Change` (the price change as a fraction) and `ConfirmedAt`. The stretch before the first swing and after the last is not reported.

`RegimeOptions.MinPhase` removes a stretch shorter than that many rows together with the two turning points around it, so its neighbours join; `MinCycle` removes a turning point that makes a low-to-low or high-to-high cycle shorter than that, with the one before it. They follow the minimum phase and cycle lengths of Pagan and Sossounov (2003). Using them looks at the whole list, so it is not causal.

---

## Oscillators

### Technical Indicators

```go
func SMA(prices insyra.IDataList, n int) (*insyra.DataList, error)
func EMA(prices insyra.IDataList, n int) (*insyra.DataList, error)
func MACD(prices insyra.IDataList, fast, slow, signal int) (*MACDResult, error)    // MACD, Signal, Histogram
func Bollinger(prices insyra.IDataList, n int, k float64) (*BollingerResult, error) // Upper, Middle, Lower
func TrueRange(high, low, close insyra.IDataList) (*insyra.DataList, error)
func ATR(high, low, close insyra.IDataList, n int) (*insyra.DataList, error)
func KD(high, low, close insyra.IDataList, n int, opts ...KDOptions) (*KDResult, error) // K, D
```

They follow TA-Lib's definitions, and the tests compare every one with TA-Lib on three seeded synthetic series (`quant/testdata/indicator_baseline.py` regenerates the values):

- A window that is not full is `NaN`, and the result has one row per price. Every returned list is named after its line (`SMA`, `MACD`, `Signal`, `Upper`, `K` and so on). A `NaN` is not accepted by the other `quant` functions that take a series, so drop the leading rows before passing an indicator on.
- `EMA` has weight `2/(n+1)` and starts from the simple average of its first `n` values. `MACD` starts both EMAs on row `slow-1`, the fast one from the simple average of the `fast` prices ending there. TA-Lib leaves the MACD line (and K of the stochastic) empty until the signal line (D) exists; here they start earlier, and the values agree wherever both have one.
- `Bollinger` uses the population standard deviation. `ATR` is Wilder's: row `n` is the mean of the first `n` true ranges and later rows are smoothed with weight `1/n`. `TrueRange` is `NaN` on row 0.
- `KD` takes where the close lies in the range of the last `n` rows (RSV, 0 to 100; a flat window counts as 50). `KDOptions.Method` chooses the smoothing: `KDRecursive` (default) is the KD of Taiwanese brokers' software, which TA-Lib and pandas-ta do not have, so their `STOCH` numbers differ from it (use `KDMovingAverage` to compare with them), where K and D start at 50 and `K = (1 − 1/SlowK) × previous K + RSV/SlowK`, then D from K the same way; `KDMovingAverage` is TA-Lib's STOCH, simple averages of RSV and then K. `SlowK` and `SlowD` default to 3. The usual setting is `n = 9`.

A missing, non-numeric or non-finite value is an error naming its row index, since a moving average over a hole is not the average of anything.

```go
macd, _ := ta.MACD(close, 12, 26, 9)
kd, _ := ta.KD(high, low, close, 9)
bands, _ := ta.Bollinger(close, 20, 2)
```

### RSI

```go
func RSI(prices insyra.IDataList, n int, opts ...RSIOptions) (*insyra.DataList, error)

type RSIOptions struct {
    Method  RSIMethod  // RSIWilder (default) or RSISimple
    Missing RSIMissing // RSIMissingError (default), RSIMissingFlat or RSIMissingBreak
}
```

Relative strength index of a price series over `n` days, `100 * avgGain / (avgGain + avgLoss)`, in [0, 100]. The averages come from the day-to-day price changes:

- `RSIWilder`: an exponential average of weight `1/n`, Wilder's original definition (1978) and the default of most software. It starts from the plain mean of the first `n` changes.
- `RSISimple`: the plain mean of the last `n` changes, Cutler's version.

The first `n` values, which lack `n` changes, are `NaN`. A stretch with no gain and no loss has no defined RSI and is `NaN`; gains only is 100 and losses only is 0. Wilder's RSI(14) on the closes of the StockCharts worked example is 70.53, 66.32, 66.55, 69.41, … and the tests pin those values.

A missing price (nil or `NaN`) is an error unless `Missing` says otherwise: `RSIMissingFlat` carries the last price over, so the missing day counts as a change of 0 (its own value is `NaN`), and `RSIMissingBreak` ends the series there and starts a new one after it, which needs `n` changes of its own. Infinities and non-numbers are always errors.

---

## Channels and Stops

```go
func Keltner(high, low, close insyra.IDataList, n, atrN int, k float64) (*ChannelResult, error)
func KeltnerClassic(high, low, close insyra.IDataList, n int) (*ChannelResult, error)
func Donchian(high, low insyra.IDataList, n int) (*ChannelResult, error)
func SuperTrend(high, low, close insyra.IDataList, n int, k float64) (*SuperTrendResult, error)
func ParabolicSAR(high, low insyra.IDataList, start, step, max float64) (*insyra.DataList, error)
func Chandelier(high, low, close insyra.IDataList, n int, k float64) (*ChandelierResult, error)

type ChannelResult struct{ Upper, Middle, Lower *insyra.DataList }
type SuperTrendResult struct{ Line, Direction *insyra.DataList } // Direction is 1 up, -1 down
type ChandelierResult struct{ Long, Short *insyra.DataList }
```

- **`Keltner`** is Raschke's form, the one trading platforms show: the `n`-row EMA of the close, plus and minus `k` times the `atrN`-row ATR (common settings 20, 10, 2). **`KeltnerClassic`** is the 1960 form: the `n`-row simple average of the typical price `(high + low + close) / 3`, plus and minus the `n`-row simple average of the day's high minus low. They are two functions because the two forms take different parameters.
- **`Donchian`** is the highest high and the lowest low of the `n` rows *before* each row, and their midpoint. The current row is left out, so a close above `Upper` is a breakout of the earlier range, not a tie with its own high. The first `n` rows are `NaN`.
- **`SuperTrend`** follows the TradingView script: bands of `(high + low) / 2` plus and minus `k` ATRs (common 10 and 3), each band only allowed to move toward the price, and a direction that flips when the close crosses the band the line is on. `Line` is the lower band while `Direction` is `1` and the upper band while it is `-1`. It starts on the first row the ATR exists (row `n`), pointing down; the ATR is TA-Lib's, whose first value is row `n`, so the first rows differ from a platform whose ATR starts one row earlier.
- **`ParabolicSAR`** is Wilder's stop and reverse. `start` is the acceleration factor a trend begins with, `step` how much it grows with each new extreme and `max` its cap (Wilder's are 0.02, 0.02 and 0.2). It matches TA-Lib's `SAR` and `SAREXT` (without the minus sign `SAREXT` puts on a short trend).
- **`Chandelier`** is the highest high of the last `n` rows (the current one included) minus `k` ATRs as the stop of a long position, and the lowest low plus `k` ATRs for a short one (common 22 and 3).

`Keltner`, `Donchian`, `SuperTrend` and `Chandelier` are not in TA-Lib; the tests compare them with implementations written from the definitions in `testdata/channels_baseline.py` (Parabolic SAR and the ATR and EMA they use come from TA-Lib). The channels are named by their lines (`Upper`, `Middle`, `Lower`, `Long`, `Short`, `Direction`, `SAR`), so a line can be added to a table as it is.

---

## Trend Indicators

```go
func Aroon(high, low insyra.IDataList, n int) (*AroonResult, error)             // Up, Down, Oscillator
func TRIX(prices insyra.IDataList, n, signal int) (*TRIXResult, error)           // TRIX, Signal
func KAMA(prices insyra.IDataList, n, fast, slow int) (*insyra.DataList, error)
func Ichimoku(high, low, close insyra.IDataList, tenkan, kijun, senkouB int, opts ...IchimokuOptions) (*IchimokuResult, error)

type IchimokuOptions struct {
    Displacement int // rows the cloud is drawn ahead of where it is computed; default 26
}
type IchimokuResult struct{ Tenkan, Kijun, SenkouA, SenkouB *insyra.DataList }
```

- **`Aroon`**: `Up` is `100 × (n − rows since the highest high of the last n+1 rows) / n`, so 100 means the high is today; `Down` is the same for the lowest low; `Oscillator` is `Up − Down`. When an extreme occurs more than once the most recent counts. The first `n` rows are `NaN`. It equals TA-Lib's `AROON` and `AROONOSC`.
- **`TRIX`** is the one-row percentage change (5 means 5%) of a triple EMA, each `n` rows long, as TA-Lib's `TRIX`; `Signal` is the `signal`-row EMA of it, the usual trading form, which TA-Lib does not have.
- **`KAMA`** is Kaufman's adaptive moving average: the efficiency ratio of the last `n` rows (net change over the sum of absolute changes) picks a smoothing constant between the fast and the slow EMA's, so the line follows a trend and barely moves sideways (usual 10, 2 and 30). It equals TA-Lib's `KAMA`, which fixes `fast` and `slow` at 2 and 30, and is reported from row `n`.
- **`Ichimoku`** returns the conversion line `Tenkan` and base line `Kijun` (midpoints of the highest high and lowest low of 9 and 26 rows, the current one included), and the two cloud lines. Charting software draws the cloud 26 rows ahead; here `SenkouA[t]` and `SenkouB[t]` are the cloud *in force on row t*, computed from row `t − Displacement`, so a close can be compared with the cloud it actually sat against and nothing is seen ahead of time. The lagging line (the close drawn 26 rows back) is just the close aligned this way and is not returned. Projecting the cloud ahead is left to charting.

Aroon, TRIX and KAMA are compared with TA-Lib; Ichimoku, and KAMA with periods other than TA-Lib's, with implementations written from the definitions (`testdata/trend_baseline.py`).

---

## Oscillators and Momentum

```go
func CCI(high, low, close insyra.IDataList, n int) (*insyra.DataList, error)
func Momentum(prices insyra.IDataList, n int) (*insyra.DataList, error)
func ROC(prices insyra.IDataList, n int) (*insyra.DataList, error)
func UltimateOscillator(high, low, close insyra.IDataList, short, medium, long int) (*insyra.DataList, error)
func StochRSI(prices insyra.IDataList, n, stochN int, opts ...KDOptions) (*KDResult, error)
func ARBR(open, high, low, close insyra.IDataList, n int) (*ARBRResult, error)            // AR, BR
func CR(high, low, close insyra.IDataList, n int, opts ...CROptions) (*insyra.DataList, error)
func ForceIndex(close, volume insyra.IDataList, n int) (*insyra.DataList, error)
func ElderRay(high, low, close insyra.IDataList, n int) (*ElderRayResult, error)          // BullPower, BearPower

type CROptions struct {
    Midpoint CRMidpoint       // CRMidHighLow (default), CRMidWeightedClose, CRMidTypical, CRMidOHLC
    Open     insyra.IDataList // required for CRMidOHLC
}
```

- **`CCI`** is the typical price `(high + low + close) / 3` minus its `n`-row average, over `0.015` times the mean absolute deviation (usual 20). A window whose typical price does not move is `NaN`.
- **`Momentum`** is the price change over `n` rows, in price units. **`ROC`** is the rate of change as a **percentage**, as TA-Lib does (5 means up 5%); like `RSI` and `KD` it is on a 0 to 100 scale, while `Bias` is a fraction.
- **`UltimateOscillator`** is Larry Williams' buying pressure over true range summed over a short, a medium and a long window and weighted 4, 2 and 1 (usual 7, 14, 28).
- **`StochRSI`** is the stochastic of the `n`-row RSI over the last `stochN` rows, smoothed as `KDOptions` say, with the same default as `KD`. `KDOptions{Method: KDMovingAverage, SlowK: 1, SlowD: 3}` equals TA-Lib's `STOCHRSI`.
- **`ARBR`** is the popularity and willingness pair common in Taiwan and China (usual 26): `AR` is `100 × Σ(high − open) / Σ(open − low)`, `BR` is `100 × Σ max(0, high − previous close) / Σ max(0, previous close − low)`; 100 is balance.
- **`CR`** is the intermediate willingness index (usual 26): `100 × Σ max(0, high − previous midpoint) / Σ max(0, previous midpoint − low)`. Platforms disagree on the midpoint, so it is an option: `(high + low) / 2` (default, the original), `(2 × close + high + low) / 4`, `(high + low + close) / 3`, or `(open + high + low + close) / 4`, which needs `CROptions.Open`.
- **`ForceIndex`** is the `n`-row EMA of `(close − previous close) × volume` (Elder; usual 13). **`ElderRay`** is the high and the low minus the `n`-row EMA of the close.

`CCI`, `Momentum`, `ROC`, `UltimateOscillator` and `StochRSI` equal TA-Lib's. `ARBR`, `CR`, `ForceIndex` and `ElderRay` are not in TA-Lib and are compared with implementations written from the definitions (`testdata/oscillators_baseline.py`).

---

## Money Flow and Volume Profile

```go
func MFI(high, low, close, volume insyra.IDataList, n int) (*insyra.DataList, error)
func AccumulationDistribution(high, low, close, volume insyra.IDataList) (*insyra.DataList, error)
func CMF(high, low, close, volume insyra.IDataList, n int) (*insyra.DataList, error)
func ChaikinOscillator(high, low, close, volume insyra.IDataList, fast, slow int) (*insyra.DataList, error)
func EaseOfMovement(high, low, volume insyra.IDataList, n int, opts ...EaseOfMovementOptions) (*insyra.DataList, error)
func NVI(close, volume insyra.IDataList) (*insyra.DataList, error)
func PVI(close, volume insyra.IDataList) (*insyra.DataList, error)
func VolumeProfile(high, low, close, volume insyra.IDataList, n int, opts ...VolumeProfileOptions) (*VolumeProfileResult, error)

type EaseOfMovementOptions struct{ Scale float64 }              // volume divisor, default 100,000,000
type VolumeProfileOptions struct{ Bins int; ValueArea float64 } // default 24 and 0.7
type VolumeProfileResult struct{ POC, ValueAreaHigh, ValueAreaLow *insyra.DataList }
```

- **`MFI`** is the money flow index (usual 14), an RSI of the typical price weighted by volume, from 0 to 100. **`AccumulationDistribution`** is the running total of the money flow volume, `((close − low) − (high − close)) / (high − low) × volume` (0 when high equals low). **`CMF`** is Chaikin money flow (usual 20): the sum of that over the sum of the volume, from −1 to 1. **`ChaikinOscillator`** is the fast EMA of the accumulation/distribution line minus the slow one (usual 3 and 10), started from its first value as TA-Lib does.
- **`EaseOfMovement`** is Arms' ease of movement (usual 14): the `n`-row simple average of the move of the range's midpoint divided by the box ratio `(volume / Scale) / (high − low)`. The scale only sets the size of the numbers; StockCharts uses 100,000,000.
- **`NVI`** and **`PVI`** are the negative and positive volume indexes: they start at 1000 and move by the price's percentage change on rows whose volume is lower (NVI) or higher (PVI) than the previous row's.
- **`VolumeProfile`** returns, for the window of the last `n` rows ending on each row, the price level with the most volume (`POC`) and the value area around it (`ValueAreaHigh`, `ValueAreaLow`). The window's range is cut into `Bins` equal bins; each bar's volume is spread over the bins it covers in proportion to the overlap (a bar with no range goes into the bin it sits in); the value area grows one bin at a time toward the side with more volume until it holds `ValueArea` of the volume. How a bar's volume is spread changes the answer, and this is the common even spread.

`MFI`, `AccumulationDistribution` and `ChaikinOscillator` equal TA-Lib's `MFI`, `AD` and `ADOSC`. The rest are compared with implementations written from the definitions (`testdata/volume_baseline.py`).

---

## Price Structure

```go
func PivotPoints(high, low, close insyra.IDataList) (*PivotLevelsResult, error)    // P, R1, R2, R3, S1, S2, S3
func HeikinAshi(open, high, low, close insyra.IDataList) (*insyra.DataTable, error) // Open, High, Low, Close
func TDSequential(close insyra.IDataList) (*TDSequentialResult, error)              // BuySetup, SellSetup, BuyCountdown, SellCountdown
func PointAndFigure(close insyra.IDataList, boxSize float64, reversal int) ([]PFColumn, error)

type PFColumn struct {
    Kind             PFKind // PFRising (X) or PFFalling (O)
    StartRow, EndRow int
    Top, Bottom      float64 // prices of the highest and lowest box
    Boxes            int
}
```

- **`PivotPoints`** are the floor-trader levels for each row, from the previous row's high `H`, low `L` and close `C`: `P = (H + L + C) / 3`, `R1 = 2P − L`, `R2 = P + (H − L)`, `R3 = H + 2(P − L)`, `S1 = 2P − H`, `S2 = P − (H − L)`, `S3 = L − 2(H − P)`. They are known at yesterday's close, so row 0 is `NaN`. They are price levels, not the turning points `ZigZag` returns (those are `Swing`s).
- **`HeikinAshi`** returns the smoothed bars as a table: the close is `(open + high + low + close) / 4`, the open is the midpoint of the previous Heikin-Ashi bar (the first is `(open + close) / 2`), and the high and low extend to cover them.
- **`TDSequential`** counts on closes only. A buy setup counts consecutive closes below the close four rows earlier (1 to 9, a broken run resets to 0, and a run that goes on past 9 starts again from 1); a sell setup is the mirror. A completed setup starts a countdown on the next row: the buy countdown counts, not necessarily in a row, the rows whose close is at or below the close two rows earlier, up to 13 (the original rule uses lows and highs); a completed opposite setup cancels it, and a second setup of the same side does not restart it. Platforms differ in the finer rules (perfection, qualifiers, recycling), and there is no reference implementation to match, so the tests use hand-built series.
- **`PointAndFigure`** builds the columns from closes: a price lies in the box `floor(price / boxSize)`, a rising column grows when the close reaches a higher box and turns into a falling one when the close falls `reversal` boxes below its top (usual 3), and the other way round. A new column starts one box beyond the old column's end.

---

## Market Breadth

```go
func AdvanceDeclineRatio(adv, dec insyra.IDataList, window int) (*insyra.DataList, error)
func AdvanceDeclineLine(adv, dec insyra.IDataList, start float64) (*insyra.DataList, error)
func OverboughtOversold(adv, dec insyra.IDataList, window int) (*insyra.DataList, error)
func NetAdvanceRatio(adv, dec insyra.IDataList, window int) (*insyra.DataList, error)
```

`adv` and `dec` are the daily numbers of advancing and declining stocks. Each function returns a list as long as its input.

- `AdvanceDeclineRatio` (ADR): advancing total divided by declining total over the last `window` days. `NaN` when the declining total is 0.
- `AdvanceDeclineLine` (ADL): `start` plus the running sum of `adv - dec`.
- `OverboughtOversold` (OBOS): the sum of `adv - dec` over the last `window` days, which is the `window`-period change of the ADL. It grows with the number of listed stocks, so its absolute value does not compare across decades.
- `NetAdvanceRatio`: `(sum adv - sum dec) / (sum adv + sum dec)` over the window, in [-1, 1] whatever the number of listed stocks. `NaN` when both totals are 0.

**Missing days.** A `nil` or `NaN` in either list marks a day with no data. That day outputs `NaN` and is skipped, not counted as 0: the window is the last `window` valid days, so a missing day does not shorten it, and the ADL carries on from its last level. Any other non-number, an infinity or a negative count is an error, as is a length mismatch, a nil list, or a `window` below 1.
