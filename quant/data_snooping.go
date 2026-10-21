package quant

import (
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
)

// ResamplingScheme selects how RealityCheck and SPATest resample time.
type ResamplingScheme int

const (
	// ResampleStationary is the stationary bootstrap of Politis & Romano
	// (1994): blocks of geometric length with mean BlockSize, starting
	// anywhere and wrapping past the end. It is the scheme White (2000) and
	// Hansen (2005) use, and the default.
	ResampleStationary ResamplingScheme = iota
	// ResampleMovingBlock is the moving block bootstrap: blocks of exactly
	// BlockSize periods starting uniformly in [0, T-BlockSize].
	ResampleMovingBlock
)

// DataSnoopingOptions configures RealityCheck, SPATest and RomanoWolf.
// Nothing is defaulted: Replications and BlockSize must be set.
type DataSnoopingOptions struct {
	// Benchmark is the benchmark's performance in the same periods as the
	// table's rows. nil compares every column with 0, as for a table that
	// already holds excess returns. A list that is itself nil, such as the
	// one GetCol returns for a missing column, is an error rather than 0.
	Benchmark insyra.IDataList
	// Replications is the number of bootstrap replications, at least 1.
	// The p-values move in steps of 1/Replications; 1,000 to 10,000 is
	// typical.
	Replications int
	// BlockSize is the (mean, for the stationary bootstrap) block length
	// in periods, 1 <= BlockSize <= T/2 (2*BlockSize <= T). It also sets
	// q = 1/BlockSize in SPATest's variance estimate. Longer blocks keep
	// more serial dependence. Blocks longer than half the sample are
	// refused: at BlockSize = T the moving block bootstrap resamples the
	// original series every time, so the bootstrap has no variance, and as
	// q approaches 0 the kernel weights approach 1, which makes ω² shrink
	// toward the sum of all the demeaned series' autocovariances, which is
	// 0. Both tests then call pure noise significant.
	BlockSize int
	// Seed initializes the random stream and always applies: the same table
	// and options reproduce bit-identical results. It is the stream
	// BlockBootstrap uses.
	Seed uint64
	// Scheme is the resampling scheme; the zero value is the stationary
	// bootstrap.
	Scheme ResamplingScheme
}

// RealityCheckResult is the outcome of White's Reality Check.
type RealityCheckResult struct {
	// PValue is the share of replications whose statistic is at least
	// Statistic (a tie counts): the probability of a best column this good
	// when no column beats the benchmark. It is 1 when Statistic is 0.
	PValue float64
	// Statistic is max(max_k sqrt(T)·dbar_k, 0), dbar_k being column k's
	// mean performance over the benchmark.
	Statistic float64
	// Best is the zero-based number of the column with the largest mean
	// over the benchmark, and BestName its name ("" if it has none).
	Best     int
	BestName string
	// Replications is the number of bootstrap replications used.
	Replications int
}

// SPAResult is the outcome of Hansen's test for Superior Predictive
// Ability.
type SPAResult struct {
	// PValueLower, PValueConsistent and PValueUpper are the p-values with
	// the lower, consistent and upper centering of Hansen (2005). Lower <=
	// Consistent <= Upper. The consistent one is the test; the other two
	// bound it. PValueUpper centers every column at zero as the Reality
	// Check does, but on studentized means.
	PValueLower      float64
	PValueConsistent float64
	PValueUpper      float64
	// Statistic is max(max_k sqrt(T)·dbar_k/ω_k, 0).
	Statistic float64
	// Best is the zero-based number of the column with the largest
	// studentized mean sqrt(T)·dbar_k/ω_k, and BestName its name.
	Best     int
	BestName string
	// Replications is the number of bootstrap replications used.
	Replications int
}

// RealityCheck runs White's (2000) Reality Check: whether the best of N
// strategies beats a benchmark, allowing for having picked the best after
// trying them all.
//
// perf is a T×N table like PBO's: column k is strategy k and row t is
// period t, with higher values better (returns, or negated losses). Each
// column is compared with opts.Benchmark (0 when nil). Every replication
// draws one sequence of time positions and applies it to all columns, so
// the correlation between strategies is kept.
//
// The statistic is V = max(max_k sqrt(T)·dbar_k, 0); each replication
// computes V* = max(0, max_k sqrt(T)·(dbar*_k − dbar_k)), and the p-value
// is the share of replications with V* >= V. Hansen and White write a strict
// inequality, which differs only on exact ties; counting a tie keeps the
// p-value at 1, not 0, when no column beats the benchmark and both
// statistics are 0. A small p-value means the best
// strategy's edge is unlikely to be luck. SPATest is the same question with
// a statistic that a poor or very noisy strategy cannot dilute.
func RealityCheck(perf insyra.IDataTable, opts DataSnoopingOptions) (*RealityCheckResult, error) {
	d, names, err := relativePerformance("RealityCheck", perf, opts)
	if err != nil {
		return nil, err
	}
	res, err := realityCheckF64(d, opts.Replications, jointDraws(len(d), opts))
	if err != nil {
		return nil, err
	}
	res.BestName = names[res.Best]
	return res, nil
}

// jointDraws returns the source of each replication's time positions: one
// seeded stream, drawn exactly as BlockBootstrap draws a path of length n.
func jointDraws(n int, opts DataSnoopingOptions) func() []int {
	rng := newBootstrapRNG(opts.Seed)
	stationary := opts.Scheme == ResampleStationary
	return func() []int {
		return blockIndices(n, n, opts.BlockSize, stationary, rng)
	}
}

// realityCheckF64 is RealityCheck on validated relative performance, with
// the time positions of each replication supplied by draw.
func realityCheckF64(d [][]float64, replications int, draw func() []int) (*RealityCheckResult, error) {
	n := len(d)
	k := len(d[0])
	means := columnMeans(d)
	rootN := math.Sqrt(float64(n))

	best := 0
	for j := 1; j < k; j++ {
		if means[j] > means[best] {
			best = j
		}
	}
	statistic := math.Max(rootN*means[best], 0)

	resampled := make([]float64, k)
	exceed := 0
	for range replications {
		if !resampledMeans(d, draw(), resampled) {
			return nil, errResampledOverflow("RealityCheck")
		}
		vStar := 0.0
		for j := range k {
			vStar = math.Max(vStar, rootN*(resampled[j]-means[j]))
		}
		if vStar >= statistic {
			exceed++
		}
	}
	return &RealityCheckResult{
		PValue:       float64(exceed) / float64(replications),
		Statistic:    statistic,
		Best:         best,
		Replications: replications,
	}, nil
}

// SPATest runs Hansen's (2005) test for Superior Predictive Ability on the
// same input as RealityCheck. Each column's mean over the benchmark is
// divided by its long-run standard deviation ω_k, so a strategy that is
// merely noisy cannot hide a good one, and a column far below the benchmark
// is set aside by the consistent centering.
//
// The statistic is T_SPA = max(max_k sqrt(T)·dbar_k/ω_k, 0). Each
// replication computes T* = max(0, max_k sqrt(T)·(dbar*_k − g(dbar_k))/ω_k)
// with the same ω_k, and a p-value is the share of replications with
// T* >= T_SPA. The lower p-value uses g(x) = max(0, x), the upper g(x) = x,
// and the consistent g(x) = x when x >= −sqrt(ω_k²/T·2·log log T) and 0
// otherwise.
//
// ω_k² = γ_0 + 2·Σ_{i=1}^{T-1} κ(T,i)·γ_i, with γ_i the lag-i
// autocovariance of column k's relative performance divided by T, and
// κ(T,i) = ((T−i)/T)(1−q)^i + (i/T)(1−q)^(T−i), q = 1/BlockSize. A column
// whose ω_k² is not positive, such as one equal to the benchmark, is an
// error, and so is one whose ω_k² is within rounding noise of zero
// relative to the column's scale, or whose relative performance is
// constant.
func SPATest(perf insyra.IDataTable, opts DataSnoopingOptions) (*SPAResult, error) {
	d, names, err := relativePerformance("SPATest", perf, opts)
	if err != nil {
		return nil, err
	}
	res, err := spaF64(d, opts.BlockSize, opts.Replications, jointDraws(len(d), opts))
	if err != nil {
		return nil, err
	}
	res.BestName = names[res.Best]
	return res, nil
}

// spaF64 is SPATest on validated relative performance, with the time
// positions of each replication supplied by draw.
func spaF64(d [][]float64, blockSize, replications int, draw func() []int) (*SPAResult, error) {
	n := len(d)
	k := len(d[0])
	means := columnMeans(d)
	omegas := make([]float64, k)
	for j := range k {
		lo, hi, meanSq := d[0][j], d[0][j], 0.0
		for t := range n {
			lo, hi = math.Min(lo, d[t][j]), math.Max(hi, d[t][j])
			meanSq += d[t][j] * d[t][j] / float64(n)
		}
		omega2 := spaLongRunVariance(d, j, means[j], blockSize)
		if math.IsNaN(omega2) || math.IsInf(omega2, 0) || math.IsInf(meanSq, 0) {
			return nil, fmt.Errorf("SPATest: column %d has a long-run variance that is not finite (ω² = %v)", j, omega2)
		}
		// Rounding leaves the centered values of a constant column at about
		// eps times its size, so ω² of such a column is about eps² (5e-32)
		// times its mean square. 1e-24 sits seven orders of magnitude above
		// that and far below the ω² of any series whose spread is visible
		// at 12 significant digits.
		if lo == hi || !(omega2 > 1e-24*meanSq) {
			return nil, fmt.Errorf("SPATest: column %d has no variance relative to the benchmark (ω² = %v)", j, omega2)
		}
		omegas[j] = math.Sqrt(omega2)
	}
	rootN := math.Sqrt(float64(n))
	logLog := 2 * math.Log(math.Log(float64(n)))

	best := 0
	bestT := math.Inf(-1)
	for j := range k {
		if t := rootN * means[j] / omegas[j]; t > bestT {
			best, bestT = j, t
		}
	}
	statistic := math.Max(bestT, 0)

	lower := make([]float64, k)
	consistent := make([]float64, k)
	for j := range k {
		lower[j] = math.Max(means[j], 0)
		if means[j] >= -math.Sqrt(omegas[j]*omegas[j]/float64(n)*logLog) {
			consistent[j] = means[j]
		}
	}

	resampled := make([]float64, k)
	var exceedLower, exceedConsistent, exceedUpper int
	for range replications {
		if !resampledMeans(d, draw(), resampled) {
			return nil, errResampledOverflow("SPATest")
		}
		var tLower, tConsistent, tUpper float64
		for j := range k {
			scale := rootN / omegas[j]
			tLower = math.Max(tLower, scale*(resampled[j]-lower[j]))
			tConsistent = math.Max(tConsistent, scale*(resampled[j]-consistent[j]))
			tUpper = math.Max(tUpper, scale*(resampled[j]-means[j]))
		}
		if tLower >= statistic {
			exceedLower++
		}
		if tConsistent >= statistic {
			exceedConsistent++
		}
		if tUpper >= statistic {
			exceedUpper++
		}
	}
	b := float64(replications)
	return &SPAResult{
		PValueLower:      float64(exceedLower) / b,
		PValueConsistent: float64(exceedConsistent) / b,
		PValueUpper:      float64(exceedUpper) / b,
		Statistic:        statistic,
		Best:             best,
		Replications:     replications,
	}, nil
}

// relativePerformance validates the input and returns d[t][k], column k
// minus the benchmark in period t, with the column names.
//
// The table and the benchmark are read inside one AtomicDoAll, so a
// concurrent writer cannot leave the result with columns or a benchmark
// from different versions.
func relativePerformance(fn string, perf insyra.IDataTable, opts DataSnoopingOptions) (d [][]float64, names []string, err error) {
	if insyra.IsNilTable(perf) {
		return nil, nil, fmt.Errorf("%s: perf is nil", fn)
	}
	// AtomicDoAll locks only the core types, so a wrapper such as
	// *isr.DataTable is unwrapped to the core object it hands to AtomicDo.
	var lockBenchmark insyra.Lockable
	if !insyra.IsNilList(opts.Benchmark) {
		opts.Benchmark.AtomicDo(func(l *insyra.DataList) {
			if l != nil {
				lockBenchmark = l
			}
		})
	}
	var lockPerf insyra.Lockable
	perf.AtomicDo(func(tb *insyra.DataTable) {
		if tb != nil {
			lockPerf = tb
		}
	})
	insyra.AtomicDoAll(func() {
		d, names, err = snapshotRelativePerformance(fn, perf, opts)
	}, lockPerf, lockBenchmark)
	return d, names, err
}

// snapshotRelativePerformance is relativePerformance's body, run while the
// table and the benchmark are locked.
func snapshotRelativePerformance(fn string, perf insyra.IDataTable, opts DataSnoopingOptions) ([][]float64, []string, error) {
	rows, cols := perf.Size()
	if cols == 0 {
		return nil, nil, fmt.Errorf("%s: perf has no columns", fn)
	}
	if rows < 3 {
		return nil, nil, fmt.Errorf("%s: perf has %d rows, need at least 3", fn, rows)
	}
	if opts.Replications < 1 {
		return nil, nil, fmt.Errorf("%s: Replications must be at least 1, got %d", fn, opts.Replications)
	}
	if opts.BlockSize < 1 || opts.BlockSize > rows/2 {
		return nil, nil, fmt.Errorf("%s: BlockSize must be at least 1 and at most half the number of rows (%d rows, so at most %d), got %d", fn, rows, rows/2, opts.BlockSize)
	}
	if opts.Scheme != ResampleStationary && opts.Scheme != ResampleMovingBlock {
		return nil, nil, fmt.Errorf("%s: unknown Scheme %v", fn, opts.Scheme)
	}

	benchmark := make([]float64, rows)
	if opts.Benchmark != nil {
		values, err := numericSeries(opts.Benchmark, fn+": Benchmark")
		if err != nil {
			return nil, nil, err
		}
		if len(values) != rows {
			return nil, nil, fmt.Errorf("%s: Benchmark has %d values, want %d (one per row of perf)", fn, len(values), rows)
		}
		benchmark = values
	}

	d := make([][]float64, rows)
	for t := range rows {
		d[t] = make([]float64, cols)
	}
	names := make([]string, cols)
	for j := range cols {
		col, err := numericSeries(perf.GetColByNumber(j), fmt.Sprintf("%s: perf column %d", fn, j))
		if err != nil {
			return nil, nil, err
		}
		if len(col) != rows {
			return nil, nil, fmt.Errorf("%s: perf column %d has %d rows, want %d", fn, j, len(col), rows)
		}
		for t := range rows {
			d[t][j] = col[t] - benchmark[t]
			if math.IsInf(d[t][j], 0) {
				return nil, nil, fmt.Errorf("%s: perf column %d minus the benchmark overflows at row index %d", fn, j, t)
			}
		}
		names[j] = perf.GetColNameByNumber(j)
	}
	for j, m := range columnMeans(d) {
		if math.IsNaN(m) || math.IsInf(m, 0) {
			return nil, nil, fmt.Errorf("%s: the mean of perf column %d over the benchmark is not finite (the values are too large to sum)", fn, j)
		}
	}
	return d, names, nil
}

// columnMeans returns the mean of every column of d.
func columnMeans(d [][]float64) []float64 {
	means := make([]float64, len(d[0]))
	for _, row := range d {
		for j, v := range row {
			means[j] += v
		}
	}
	for j := range means {
		means[j] /= float64(len(d))
	}
	return means
}

// resampledMeans writes into out the mean of every column of d over the
// rows indices names, the same rows for every column.
//
// It returns false when a resampled mean is not finite: finite values that
// cancel in the original order can overflow once a draw repeats same-sign
// rows, and an infinite or NaN mean must not be counted as an exceedance.
func resampledMeans(d [][]float64, indices []int, out []float64) bool {
	clear(out)
	for _, t := range indices {
		for j, v := range d[t] {
			out[j] += v
		}
	}
	finite := true
	for j := range out {
		out[j] /= float64(len(indices))
		if math.IsNaN(out[j]) || math.IsInf(out[j], 0) {
			finite = false
		}
	}
	return finite
}

func errResampledOverflow(fn string) error {
	return fmt.Errorf("%s: a resampled mean is not finite (the values are too large to sum when a draw repeats same-sign rows)", fn)
}

// spaLongRunVariance is Hansen's (2005) ω_k² for column j: the kernel
// estimate implied by the stationary bootstrap with q = 1/blockSize.
func spaLongRunVariance(d [][]float64, j int, mean float64, blockSize int) float64 {
	n := len(d)
	nf := float64(n)
	q := 1 / float64(blockSize)
	centered := make([]float64, n)
	for t := range n {
		centered[t] = d[t][j] - mean
	}
	autocov := func(lag int) float64 {
		s := 0.0
		for t := 0; t+lag < n; t++ {
			s += centered[t] * centered[t+lag]
		}
		return s / nf
	}
	omega2 := autocov(0)
	for i := 1; i < n; i++ {
		fi := float64(i)
		kappa := (nf-fi)/nf*math.Pow(1-q, fi) + fi/nf*math.Pow(1-q, nf-fi)
		if kappa == 0 {
			continue
		}
		omega2 += 2 * kappa * autocov(i)
	}
	return omega2
}
