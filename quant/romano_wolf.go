package quant

import (
	"fmt"
	"math"
	"runtime"
	"slices"
	"sync"

	"gonum.org/v1/gonum/dsp/fourier"

	"github.com/HazelnutParadise/insyra"
)

// RomanoWolfOptions configures RomanoWolf. Benchmark, Replications,
// BlockSize, Seed and Scheme mean what they do for RealityCheck and are
// validated the same way. Nothing is defaulted.
type RomanoWolfOptions struct {
	DataSnoopingOptions
	// Alpha is the familywise level in (0, 1): the chance of declaring even
	// one strategy better than the benchmark when none is. It must be set;
	// 0 is an error.
	Alpha float64
}

// RomanoWolfResult is the outcome of the Romano-Wolf stepdown.
type RomanoWolfResult struct {
	// Alpha is the familywise level used.
	Alpha float64
	// Names are the column names in table order, "" for a column without one.
	Names []string
	// RejectedAt[k] is the 1-based step at which column k was declared to
	// beat the benchmark, or 0 if it was not.
	RejectedAt []int
	// Statistics[k] is w_k = sqrt(T)·dbar_k/σ_k for every column.
	Statistics []float64
	// CriticalValues[s-1] is the critical value used at step s. Its length
	// is the number of steps run: the last step either rejected nothing or
	// rejected the last remaining columns.
	CriticalValues []float64
	// Replications is the number of bootstrap replications used.
	Replications int
}

// RomanoWolf runs Romano and Wolf's (2005) stepdown: given many strategies
// (the columns of a T×N performance table, as for RealityCheck) and a
// benchmark, it names the strategies that beat the benchmark while keeping
// the chance of even one false claim at or below opts.Alpha (the familywise
// error rate). RealityCheck and SPATest only judge whether the best one is
// real.
//
// With d_k,t the column minus the benchmark, dbar_k its mean and σ_k its
// long-run standard deviation, w_k = sqrt(T)·dbar_k/σ_k. Every replication
// draws one sequence of time positions for all columns, as RealityCheck
// does, and computes w*_k = sqrt(T)·(dbar*_k − dbar_k)/σ*_k, with σ*_k
// estimated again on that replication's resampled series. At step s the
// critical value is the ceil(Alpha·B)-th largest, over the B replications,
// of the maximum of w* over the columns not rejected so far, and every such
// column with w_k above it is rejected at step s. The steps repeat on the
// remaining columns until one rejects nothing or none is left. A tie with the
// critical value is not a rejection, so when everything is tied nothing is
// rejected.
//
// The test is one-sided: a column is rejected only for beating the
// benchmark. It gives no adjusted p-values: the stepdown yields a
// rejection decision at the chosen level, not a p-value per column.
//
// σ_k and σ*_k are SPATest's ω_k, Hansen's (2005) stationary-bootstrap
// kernel estimate with q = 1/BlockSize, computed by FFT. Re-estimating it on
// every resample follows Götze and Künsch (1996) and Python arch. A column
// whose σ² is not finite or is at most 1e-24 times its mean square, in the
// sample or in any replication, is an error, as is a resampled mean that is
// not finite.
func RomanoWolf(perf insyra.IDataTable, opts RomanoWolfOptions) (*RomanoWolfResult, error) {
	if math.IsNaN(opts.Alpha) || opts.Alpha <= 0 || opts.Alpha >= 1 {
		return nil, fmt.Errorf("RomanoWolf: Alpha must be in (0, 1) and is not defaulted, got %v", opts.Alpha)
	}
	d, names, err := relativePerformance("RomanoWolf", perf, opts.DataSnoopingOptions)
	if err != nil {
		return nil, err
	}
	res, err := romanoWolfF64(d, opts.BlockSize, opts.Alpha, opts.Replications, jointDraws(len(d), opts.DataSnoopingOptions))
	if err != nil {
		return nil, err
	}
	res.Names = names
	return res, nil
}

// lrvKernel holds the weights 2·κ(n,i), i = 1..n−1, of SPATest's ω² for a
// series of n values with q = 1/blockSize:
// κ(n,i) = ((n−i)/n)(1−q)^i + (i/n)(1−q)^(n−i).
type lrvKernel struct {
	n      int
	size   int // FFT length: the smallest 5-smooth number of at least 2n
	weight []float64
}

func newLRVKernel(n, blockSize int) *lrvKernel {
	q := 1 / float64(blockSize)
	k := &lrvKernel{n: n, size: 1, weight: make([]float64, n)}
	for k.size = 2 * n; !fiveSmooth(k.size); k.size++ {
	}
	nf := float64(n)
	for i := 1; i < n; i++ {
		fi := float64(i)
		kappa := (nf-fi)/nf*math.Pow(1-q, fi) + fi/nf*math.Pow(1-q, nf-fi)
		k.weight[i] = 2 * kappa
	}
	return k
}

// fiveSmooth reports whether m has no prime factor above 5, the lengths the
// FFT handles without a slow general-radix pass.
func fiveSmooth(m int) bool {
	for _, p := range []int{2, 3, 5} {
		for m%p == 0 {
			m /= p
		}
	}
	return m == 1
}

// lrvWorkspace holds the FFT plan and buffers one goroutine reuses for every
// series; it is not safe for concurrent use.
type lrvWorkspace struct {
	fft    *fourier.FFT
	padded []float64
	coeff  []complex128
	auto   []float64
}

func (k *lrvKernel) workspace() *lrvWorkspace {
	return &lrvWorkspace{
		fft:    fourier.NewFFT(k.size),
		padded: make([]float64, k.size),
		coeff:  make([]complex128, k.size/2+1),
		auto:   make([]float64, k.size),
	}
}

// variance is ω² of the centered series c: γ_0 + Σ_{i=1}^{n−1} 2κ(n,i)·γ_i,
// γ_i being the lag-i autocovariance divided by n. It is SPATest's formula
// with every γ_i obtained at once: the inverse transform of the power
// spectrum of c, zero-padded to at least 2n so the circular autocorrelation
// equals the linear one, is Σ_t c_t·c_{t+i} times the transform length.
func (k *lrvKernel) variance(c []float64, w *lrvWorkspace) float64 {
	copy(w.padded, c)
	clear(w.padded[len(c):])
	w.fft.Coefficients(w.coeff, w.padded)
	for i, v := range w.coeff {
		w.coeff[i] = complex(real(v)*real(v)+imag(v)*imag(v), 0)
	}
	w.fft.Sequence(w.auto, w.coeff)
	scale := 1 / (float64(k.size) * float64(k.n))
	total := w.auto[0] * scale
	for i := 1; i < k.n; i++ {
		total += k.weight[i] * (w.auto[i] * scale)
	}
	return total
}

// longRunVarianceFFT is the variance RomanoWolf uses for the series y with
// mean mean: SPATest's ω² computed by FFT.
func longRunVarianceFFT(y []float64, mean float64, blockSize int) float64 {
	c := make([]float64, len(y))
	for i, v := range y {
		c[i] = v - mean
	}
	k := newLRVKernel(len(y), blockSize)
	return k.variance(c, k.workspace())
}

// stepCriticalValue returns the ceil(alpha·B)-th largest of maxima (sorting
// it): a statistic above it has fewer than alpha·B replications at or above
// it, which is a p-value below alpha counted with ties as RealityCheck does.
// The 1e-9 keeps an alpha·B that is an integer, such as 0.1·120, from
// rounding up to the next one.
func stepCriticalValue(maxima []float64, alpha float64) float64 {
	b := len(maxima)
	j := min(max(int(math.Ceil(alpha*float64(b)-1e-9)), 1), b)
	slices.Sort(maxima)
	return maxima[b-j]
}

// romanoWolfF64 is RomanoWolf on validated relative performance, with the
// time positions of each replication supplied by draw. Names are left to
// the caller.
func romanoWolfF64(d [][]float64, blockSize int, alpha float64, replications int, draw func() []int) (*RomanoWolfResult, error) {
	n := len(d)
	k := len(d[0])
	means := columnMeans(d)
	rootN := math.Sqrt(float64(n))
	kernel := newLRVKernel(n, blockSize)

	// Column-major copy, so a column of a resample is gathered from one slice.
	cols := make([][]float64, k)
	for j := range k {
		cols[j] = make([]float64, n)
		for t := range n {
			cols[j][t] = d[t][j]
		}
	}

	statistics := make([]float64, k)
	centered := make([]float64, n)
	ws := kernel.workspace()
	for j := range k {
		lo, hi, meanSq := cols[j][0], cols[j][0], 0.0
		for t, v := range cols[j] {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
			meanSq += v * v / float64(n)
			centered[t] = v - means[j]
		}
		sigma2 := kernel.variance(centered, ws)
		if math.IsNaN(sigma2) || math.IsInf(sigma2, 0) || math.IsInf(meanSq, 0) {
			return nil, fmt.Errorf("RomanoWolf: column %d has a long-run variance that is not finite (σ² = %v)", j, sigma2)
		}
		// The floor SPATest uses: rounding leaves a constant column at about
		// eps² times its mean square.
		if lo == hi || !(sigma2 > 1e-24*meanSq) {
			return nil, fmt.Errorf("RomanoWolf: column %d has no variance relative to the benchmark (σ² = %v)", j, sigma2)
		}
		statistics[j] = rootN * means[j] / math.Sqrt(sigma2)
	}

	wStar, err := resampledStatistics(cols, d, means, kernel, replications, draw)
	if err != nil {
		return nil, err
	}

	rejectedAt := make([]int, k)
	var critical []float64
	active := make([]int, k)
	for j := range active {
		active[j] = j
	}
	for step := 1; len(active) > 0; step++ {
		maxima := make([]float64, replications)
		for b, row := range wStar {
			m := math.Inf(-1)
			for _, j := range active {
				m = math.Max(m, row[j])
			}
			maxima[b] = m
		}
		c := stepCriticalValue(maxima, alpha)
		critical = append(critical, c)
		rejected := false
		remaining := active[:0]
		for _, j := range active {
			if statistics[j] > c {
				rejectedAt[j] = step
				rejected = true
			} else {
				remaining = append(remaining, j)
			}
		}
		if !rejected {
			break
		}
		active = remaining
	}
	return &RomanoWolfResult{
		Alpha:          alpha,
		RejectedAt:     rejectedAt,
		Statistics:     statistics,
		CriticalValues: critical,
		Replications:   replications,
	}, nil
}

// resampledStatistics returns w*[b][j] = sqrt(T)·(dbar*_j − dbar_j)/σ*_j for
// every replication b. The positions are drawn one replication at a time
// from the seeded stream, so the stream does not depend on the number of
// goroutines; the replications themselves are independent and are computed
// in parallel, each into its own row.
func resampledStatistics(cols, d [][]float64, means []float64, kernel *lrvKernel, replications int, draw func() []int) ([][]float64, error) {
	n, k := kernel.n, len(cols)
	rootN := math.Sqrt(float64(n))
	wStar := make([][]float64, replications)
	workers := min(runtime.GOMAXPROCS(0), replications)
	batch := make([][]int, 0, 4*workers)
	errs := make([]error, 4*workers)

	for start := 0; start < replications; start += len(batch) {
		batch = batch[:0]
		for b := start; b < min(start+4*workers, replications); b++ {
			batch = append(batch, draw())
		}
		var wg sync.WaitGroup
		for w := range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				out := make([]float64, k)
				c := make([]float64, n)
				ws := kernel.workspace()
				for r := w; r < len(batch); r += workers {
					b := start + r
					errs[r] = nil
					if !resampledMeans(d, batch[r], out) {
						errs[r] = errResampledOverflow("RomanoWolf")
						continue
					}
					row := make([]float64, k)
					for j := range k {
						meanSq := 0.0
						for t, i := range batch[r] {
							v := cols[j][i]
							meanSq += v * v / float64(n)
							c[t] = v - out[j]
						}
						sigma2 := kernel.variance(c, ws)
						if math.IsNaN(sigma2) || math.IsInf(sigma2, 0) || math.IsInf(meanSq, 0) || !(sigma2 > 1e-24*meanSq) {
							errs[r] = fmt.Errorf("RomanoWolf: column %d has no usable variance in the resampled series of replication %d (σ*² = %v)", j, b, sigma2)
							break
						}
						row[j] = rootN * (out[j] - means[j]) / math.Sqrt(sigma2)
					}
					wStar[b] = row
				}
			}()
		}
		wg.Wait()
		for r := range batch {
			if errs[r] != nil {
				return nil, errs[r]
			}
		}
	}
	return wStar, nil
}
