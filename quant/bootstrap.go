package quant

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"

	"github.com/HazelnutParadise/insyra"
)

// BootstrapConfig configures BlockBootstrap.
type BootstrapConfig struct {
	// Horizon is the number of future periods to simulate per path (e.g.
	// 252 trading days). Must be positive.
	Horizon int
	// BlockSize is the block length in periods, 1 <= BlockSize <=
	// len(returns). With the moving block bootstrap (Stationary false)
	// every block has exactly this length; with the stationary bootstrap it
	// is the MEAN block length. A rule of thumb is roughly n^(1/3) for n
	// observations; longer blocks keep more autocorrelation and volatility
	// clustering, shorter blocks give more distinct paths.
	BlockSize int
	// Paths is the number of simulated paths. Must be positive. A few
	// thousand is typical for stable 5%/95% bands.
	Paths int
	// Seed initializes the random stream and ALWAYS applies: the same
	// returns and config reproduce bit-identical output, and the zero value
	// is simply seed 0. There is no "unset means random" mode — pass a
	// clock-derived value yourself if you want a fresh draw. The stream is
	// a PCG generator, so results do not depend on the Go release.
	Seed uint64
	// Stationary selects the resampling scheme. false (default) is the
	// moving block bootstrap (Künsch 1989): fixed-length blocks whose start
	// is uniform over [0, n-BlockSize]. true is the stationary bootstrap
	// (Politis & Romano 1994): geometrically distributed block lengths with
	// mean BlockSize, starting anywhere in the series and wrapping around
	// its end, which removes the edge effects of fixed blocks and makes the
	// resampled series stationary.
	Stationary bool
}

// BootstrapResult holds the output of BlockBootstrap.
type BootstrapResult struct {
	// Returns[p] is the p-th resampled per-period return series, length
	// Horizon. Use it for per-path statistics such as a bootstrapped Sharpe
	// distribution.
	Returns [][]float64
	// Equity[p] is Returns[p] compounded from 1.0, so Equity[p][0] == 1 and
	// len(Equity[p]) == Horizon+1 — the same convention as
	// WalkForwardResult.Equity. Use it for fan charts via PercentileBands.
	Equity [][]float64
}

// BlockBootstrap resamples a per-period return series in blocks and
// compounds each resampled series into an equity path, for probabilistic
// ("where might this configuration be in a year?") rather than point
// forecasts. Sampling whole blocks preserves the autocorrelation, volatility
// clustering, and fat tails of the observed returns without assuming any
// distribution.
//
// returns are per-period simple returns (0.012 for +1.2%); the equity
// paths assume they are >= -1. Every value must be numeric and finite —
// an unreadable, NaN, or Inf element is an error naming its row, never a
// substituted zero. Configuration is validated, not defaulted: see
// BootstrapConfig for the rules.
//
// Given identical returns and config the output is bit-identical (see
// BootstrapConfig.Seed).
func BlockBootstrap(returns insyra.IDataList, cfg BootstrapConfig) (*BootstrapResult, error) {
	values, err := numericSeries(returns, "BlockBootstrap: returns")
	if err != nil {
		return nil, err
	}
	return blockBootstrapF64(values, cfg)
}

func blockBootstrapF64(returns []float64, cfg BootstrapConfig) (*BootstrapResult, error) {
	n := len(returns)
	if n == 0 {
		return nil, errors.New("BlockBootstrap: returns is empty")
	}
	if cfg.Horizon <= 0 {
		return nil, errors.New("BlockBootstrap: Horizon must be positive")
	}
	if cfg.Paths <= 0 {
		return nil, errors.New("BlockBootstrap: Paths must be positive")
	}
	if cfg.BlockSize < 1 {
		return nil, errors.New("BlockBootstrap: BlockSize must be at least 1")
	}
	if cfg.BlockSize > n {
		return nil, fmt.Errorf("BlockBootstrap: BlockSize (%d) exceeds the number of returns (%d)", cfg.BlockSize, n)
	}

	rng := newBootstrapRNG(cfg.Seed)
	res := &BootstrapResult{
		Returns: make([][]float64, cfg.Paths),
		Equity:  make([][]float64, cfg.Paths),
	}
	for p := range cfg.Paths {
		var series []float64
		if cfg.Stationary {
			series = stationaryBootstrapSeries(returns, cfg.Horizon, cfg.BlockSize, rng)
		} else {
			series = movingBlockSeries(returns, cfg.Horizon, cfg.BlockSize, rng)
		}
		equity := make([]float64, cfg.Horizon+1)
		equity[0] = 1
		for t, r := range series {
			equity[t+1] = equity[t] * (1 + r)
		}
		res.Returns[p] = series
		res.Equity[p] = equity
	}
	return res, nil
}

// movingBlockSeries draws fixed-length blocks with a uniform start in
// [0, n-blockSize] until horizon values are collected, then truncates.
func movingBlockSeries(returns []float64, horizon, blockSize int, rng *bootstrapRNG) []float64 {
	series := make([]float64, 0, horizon+blockSize)
	starts := uint64(len(returns) - blockSize + 1)
	for len(series) < horizon {
		start := int(rng.uintN(starts))
		series = append(series, returns[start:start+blockSize]...)
	}
	return series[:horizon:horizon]
}

// stationaryBootstrapSeries draws blocks whose start is uniform over the
// whole series and whose length is geometric with mean blockSize, reading
// circularly past the end, until horizon values are collected. A block that
// would overrun the horizon is cut at the horizon.
func stationaryBootstrapSeries(returns []float64, horizon, blockSize int, rng *bootstrapRNG) []float64 {
	n := len(returns)
	series := make([]float64, 0, horizon)
	for len(series) < horizon {
		start := int(rng.uintN(uint64(n)))
		length := geometricBlockLength(rng, blockSize)
		if remaining := horizon - len(series); length > remaining {
			length = remaining
		}
		series = appendCircular(series, returns, start, length)
	}
	return series
}

// geometricBlockLength draws L >= 1 with P(L = k) = p(1-p)^(k-1),
// p = 1/mean, so E[L] = mean. mean == 1 always yields 1 and consumes no
// randomness.
func geometricBlockLength(rng *bootstrapRNG, mean int) int {
	if mean <= 1 {
		return 1
	}
	logQ := math.Log(1 - 1/float64(mean))
	return 1 + int(math.Floor(math.Log(rng.unitOpen())/logQ))
}

// appendCircular appends length values of returns starting at start,
// wrapping past the end back to index 0.
func appendCircular(series, returns []float64, start, length int) []float64 {
	n := len(returns)
	for j := range length {
		series = append(series, returns[(start+j)%n])
	}
	return series
}

// bootstrapSeedXor mirrors the core sampling seeding so one uint64 seeds
// both PCG words.
const bootstrapSeedXor uint64 = 0x9E3779B97F4A7C15

// bootstrapRNG wraps a PCG source and performs the uniform reductions
// itself, so the output depends only on the PCG sequence — a published
// algorithm — and not on the standard library's IntN/Float64 routines,
// whose stability across Go releases is not documented.
type bootstrapRNG struct {
	src *rand.PCG
}

func newBootstrapRNG(seed uint64) *bootstrapRNG {
	return &bootstrapRNG{src: rand.NewPCG(seed, seed^bootstrapSeedXor)}
}

// uintN returns a uniform integer in [0, n) by rejection sampling. n > 0.
func (r *bootstrapRNG) uintN(n uint64) uint64 {
	threshold := -n % n // (2^64 mod n): values below it would bias the modulo
	for {
		v := r.src.Uint64()
		if v >= threshold {
			return v % n
		}
	}
}

// unitOpen returns a uniform float64 in (0, 1] — never 0, so its log is
// finite.
func (r *bootstrapRNG) unitOpen() float64 {
	return float64(r.src.Uint64()>>11+1) * (1.0 / (1 << 53))
}

// PercentileBands takes, at every time step, the requested percentiles
// across all paths — the vertical slices of a fan chart. paths is a matrix
// with one row per path and one column per step (BootstrapResult.Equity or
// BootstrapResult.Returns, or any equal-length rows). percentiles are on the
// 0..100 scale, and bands[i] is the series for percentiles[i], in the
// caller's order.
//
// The quantile is R's type-7 (linear interpolation between order
// statistics), the same definition DataList.Percentile, Quartile, and
// Describe use, so bands agree with the rest of the library.
//
// Returns an error if paths is empty or ragged, or percentiles is empty or
// contains a value outside [0, 100].
func PercentileBands(paths [][]float64, percentiles []float64) ([][]float64, error) {
	if len(paths) == 0 {
		return nil, errors.New("PercentileBands: paths is empty")
	}
	width := len(paths[0])
	if width == 0 {
		return nil, errors.New("PercentileBands: paths have no steps")
	}
	for p, path := range paths {
		if len(path) != width {
			return nil, fmt.Errorf("PercentileBands: path %d has %d steps, want %d", p, len(path), width)
		}
	}
	if len(percentiles) == 0 {
		return nil, errors.New("PercentileBands: percentiles is empty")
	}
	for i, q := range percentiles {
		if math.IsNaN(q) || q < 0 || q > 100 {
			return nil, fmt.Errorf("PercentileBands: percentile %d (%v) is outside [0, 100]", i, q)
		}
	}

	bands := make([][]float64, len(percentiles))
	for i := range bands {
		bands[i] = make([]float64, width)
	}
	column := make([]float64, len(paths))
	for t := range width {
		for p := range paths {
			column[p] = paths[p][t]
		}
		sort.Float64s(column)
		for i, q := range percentiles {
			bands[i][t] = quantileType7(column, q/100)
		}
	}
	return bands, nil
}

// quantileType7 is the p-quantile (p in [0, 1]) of an ascending-sorted,
// non-empty slice by R's type-7 method: linear interpolation at position
// h = p*(n-1). It duplicates the root package's unexported helper of the
// same name; TestPercentileBandsMatchDataListPercentile pins the two
// together.
func quantileType7(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return math.NaN()
	}
	if n == 1 {
		return sorted[0]
	}
	h := p * float64(n-1)
	lower := int(math.Floor(h))
	upper := int(math.Ceil(h))
	if lower < 0 {
		lower = 0
	}
	if upper >= n {
		upper = n - 1
	}
	if lower == upper {
		return sorted[lower]
	}
	return sorted[lower] + (h-float64(lower))*(sorted[upper]-sorted[lower])
}

// numericSeries reads a DataList under its actor and converts every element
// to float64, refusing anything that is not a finite number. It exists
// because DataList.ToF64Slice has no failure channel and yields 0 for a
// value it cannot parse; a resampled zero would silently flatten the
// forecast. label prefixes the error so the caller can find the cell.
func numericSeries(dl insyra.IDataList, label string) ([]float64, error) {
	if dl == nil {
		return nil, fmt.Errorf("%s is nil", label)
	}
	var raw []any
	dl.AtomicDo(func(l *insyra.DataList) {
		raw = l.Data()
	})
	out := make([]float64, len(raw))
	for i, value := range raw {
		converted, ok := insyra.ToFloat64Safe(value)
		if !ok {
			return nil, fmt.Errorf("%s contains a non-numeric value at row %d: %v", label, i+1, value)
		}
		if math.IsNaN(converted) || math.IsInf(converted, 0) {
			return nil, fmt.Errorf("%s contains a non-finite value at row %d: %v", label, i+1, converted)
		}
		out[i] = converted
	}
	return out, nil
}
