package quant

import (
	"errors"
	"fmt"
	"math"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/stat"
	"gonum.org/v1/gonum/stat/combin"
)

// eulerMascheroni is the γ constant used by ExpectedMaxSharpe's
// extreme-value approximation of the expected maximum Sharpe ratio.
const eulerMascheroni = 0.5772156649015329

// ProbabilisticSharpeRatio returns the Probabilistic Sharpe Ratio (PSR)
// of Bailey & López de Prado: the probability that the true Sharpe ratio
// exceeds a benchmark, given the estimate's standard error under
// non-normal returns.
//
//	PSR = Φ[ (SR̂ - SR*)·√(n-1) / √(1 - γ₃·SR̂ + ((γ₄-1)/4)·SR̂²) ]
//
// observedSR (SR̂) and benchmarkSR (SR*) are per-period, NON-annualized
// Sharpe ratios. n is the number of return observations. skew (γ₃) and
// kurt (γ₄) are the skewness and the NON-excess kurtosis of the returns
// (a normal distribution has skew 0 and kurt 3).
//
// Returns an error if n < 2 or the variance term in the denominator is
// non-positive (which can happen under extreme skew/kurtosis combinations).
func ProbabilisticSharpeRatio(observedSR, benchmarkSR float64, n int, skew, kurt float64) (float64, error) {
	if n < 2 {
		return math.NaN(), errors.New("ProbabilisticSharpeRatio: need at least 2 observations")
	}
	varTerm := 1 - skew*observedSR + (kurt-1)/4*observedSR*observedSR
	if varTerm <= 0 {
		return math.NaN(), fmt.Errorf("ProbabilisticSharpeRatio: non-positive variance term %v (extreme skew/kurtosis)", varTerm)
	}
	z := (observedSR - benchmarkSR) * math.Sqrt(float64(n-1)) / math.Sqrt(varTerm)
	return stats.NormCDF(z), nil
}

// ExpectedMaxSharpe returns SR₀, the expected maximum (per-period,
// non-annualized) Sharpe ratio obtained by chance after nTrials
// independent backtests whose Sharpe ratios have variance sharpeVariance.
// This is the deflation benchmark used by DeflatedSharpeRatio:
//
//	SR₀ = √V · [ (1-γ)·Z⁻¹(1 - 1/N) + γ·Z⁻¹(1 - 1/(N·e)) ]
//
// where V = sharpeVariance, N = nTrials, γ is the Euler-Mascheroni
// constant, e is Euler's number, and Z⁻¹ is the standard-normal quantile.
//
// With nTrials ≤ 1 there is no selection bias, so SR₀ = 0. Returns an
// error if sharpeVariance is negative.
func ExpectedMaxSharpe(sharpeVariance float64, nTrials int) (float64, error) {
	if sharpeVariance < 0 {
		return math.NaN(), errors.New("ExpectedMaxSharpe: sharpeVariance must be non-negative")
	}
	if nTrials <= 1 {
		return 0, nil
	}
	n := float64(nTrials)
	// Both quantile arguments lie strictly in (0, 1) for nTrials >= 2.
	q1, err := stats.NormPPF(1 - 1/n)
	if err != nil {
		return math.NaN(), err
	}
	q2, err := stats.NormPPF(1 - 1/(n*math.E))
	if err != nil {
		return math.NaN(), err
	}
	return math.Sqrt(sharpeVariance) * ((1-eulerMascheroni)*q1 + eulerMascheroni*q2), nil
}

// DeflatedSharpeRatio returns the Deflated Sharpe Ratio (DSR): the PSR of
// the selected strategy measured against the deflation benchmark SR₀
// derived from the whole set of trial Sharpe ratios. It corrects the
// observed Sharpe for selection bias from multiple testing, non-normality,
// and sample length in one number — DSR ≈ 1 means the result survives
// deflation; DSR near 0 means it is likely a false discovery.
//
// observedSR is the selected strategy's per-period (non-annualized) Sharpe
// (typically the maximum of trialSharpes). n is its number of return
// observations; skew and kurt are that strategy's skewness and non-excess
// kurtosis. trialSharpes holds the per-period Sharpe ratios of ALL trials
// considered during the search; their count and (population) variance feed
// SR₀.
//
// Returns an error if trialSharpes is empty or any downstream computation
// fails.
func DeflatedSharpeRatio(observedSR float64, n int, skew, kurt float64, trialSharpes insyra.IDataList) (float64, error) {
	sharpes := trialSharpes.ToF64Slice()
	if len(sharpes) == 0 {
		return math.NaN(), errors.New("DeflatedSharpeRatio: trialSharpes is empty")
	}
	v := populationVariance(sharpes)
	sr0, err := ExpectedMaxSharpe(v, len(sharpes))
	if err != nil {
		return math.NaN(), err
	}
	return ProbabilisticSharpeRatio(observedSR, sr0, n, skew, kurt)
}

// PBO estimates the Probability of Backtest Overfitting via Combinatorially
// Symmetric Cross-Validation (CSCV), per Bailey, Borwein, López de Prado &
// Zhu.
//
// perf is a T×N performance DataTable: column j is candidate strategy j and
// row i is period i, so perf[i][j] is strategy j's period-i return (T rows,
// N ≥ 2 columns). nSplits (S) is the number of equal, contiguous time blocks
// the rows are cut into; it must be even. CSCV enumerates every way to split
// the S blocks into an in-sample half (IS) and an out-of-sample half (OOS).
// For each split it picks the IS-best strategy (by Sharpe ratio) and records
// its OOS rank; PBO is the fraction of splits where that strategy's OOS
// performance lands in the bottom half (logit ω ≤ 0). A high PBO means
// in-sample winners tend to be out-of-sample losers — the signature of
// overfitting.
//
// If T is not a multiple of nSplits, the trailing T mod nSplits rows are
// dropped (each block has T/nSplits rows). Per-block Sharpe uses the sample
// standard deviation; a zero-volatility series contributes a Sharpe of 0.
//
// Returns an error for an empty matrix, columns of unequal length, fewer
// than 2 strategies, an odd or non-positive nSplits, or nSplits greater
// than T.
func PBO(perf insyra.IDataTable, nSplits int) (float64, error) {
	numRows, numCols := perf.Size()
	if numRows == 0 || numCols == 0 {
		return math.NaN(), errors.New("PBO: perf is empty")
	}
	matrix := make([][]float64, numRows)
	for i := range numRows {
		matrix[i] = make([]float64, numCols)
	}
	for j := range numCols {
		col := perf.GetColByNumber(j).ToF64Slice()
		if len(col) != numRows {
			return math.NaN(), fmt.Errorf("PBO: column %d has %d rows, want %d (columns must be equal length)", j, len(col), numRows)
		}
		for i := range numRows {
			matrix[i][j] = col[i]
		}
	}
	return pboF64(matrix, nSplits)
}

// pboF64 is the CSCV core operating on a T×N [][]float64 matrix.
func pboF64(perf [][]float64, nSplits int) (float64, error) {
	t := len(perf)
	if t == 0 {
		return math.NaN(), errors.New("PBO: perf is empty")
	}
	n := len(perf[0])
	if n < 2 {
		return math.NaN(), errors.New("PBO: need at least 2 strategies (columns)")
	}
	for i, row := range perf {
		if len(row) != n {
			return math.NaN(), fmt.Errorf("PBO: row %d has %d columns, want %d (matrix must be rectangular)", i, len(row), n)
		}
	}
	if nSplits < 2 || nSplits%2 != 0 {
		return math.NaN(), errors.New("PBO: nSplits must be a positive even number")
	}
	if nSplits > t {
		return math.NaN(), fmt.Errorf("PBO: nSplits (%d) exceeds number of rows (%d)", nSplits, t)
	}

	blockSize := t / nSplits
	// blockRows[b] holds the row indices belonging to block b.
	blockRows := make([][]int, nSplits)
	for b := range nSplits {
		rows := make([]int, blockSize)
		for k := range blockSize {
			rows[k] = b*blockSize + k
		}
		blockRows[b] = rows
	}

	combos := combin.Combinations(nSplits, nSplits/2)
	overfit := 0
	for _, isBlocks := range combos {
		inIS := make([]bool, nSplits)
		for _, b := range isBlocks {
			inIS[b] = true
		}

		isSharpe := make([]float64, n)
		oosSharpe := make([]float64, n)
		for j := range n {
			var isVals, oosVals []float64
			for b := range nSplits {
				for _, i := range blockRows[b] {
					if inIS[b] {
						isVals = append(isVals, perf[i][j])
					} else {
						oosVals = append(oosVals, perf[i][j])
					}
				}
			}
			isSharpe[j] = perPeriodSharpe(isVals)
			oosSharpe[j] = perPeriodSharpe(oosVals)
		}

		nStar := argmax(isSharpe)
		// OOS rank of n* (1 = worst .. n = best), ties counted as ≤.
		rank := 0
		for j := range n {
			if oosSharpe[j] <= oosSharpe[nStar] {
				rank++
			}
		}
		omega := float64(rank) / float64(n+1)
		if omega <= 0.5 { // logit(ω) ≤ 0
			overfit++
		}
	}
	return float64(overfit) / float64(len(combos)), nil
}

// perPeriodSharpe returns mean/stddev (sample) of xs, the non-annualized
// Sharpe ratio. A series shorter than 2 or with zero volatility yields 0.
func perPeriodSharpe(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	sd := stat.StdDev(xs, nil)
	if sd == 0 {
		return 0
	}
	return stat.Mean(xs, nil) / sd
}

// argmax returns the index of the maximum element (first on ties). For an
// empty slice it returns 0.
func argmax(xs []float64) int {
	best := 0
	for i := 1; i < len(xs); i++ {
		if xs[i] > xs[best] {
			best = i
		}
	}
	return best
}

// populationVariance returns the population variance (÷ N) of xs, matching
// the convention used by López de Prado's reference implementation for the
// spread of trial Sharpe ratios.
func populationVariance(xs []float64) float64 {
	n := float64(len(xs))
	if n == 0 {
		return 0
	}
	mean := stat.Mean(xs, nil)
	ss := 0.0
	for _, x := range xs {
		d := x - mean
		ss += d * d
	}
	return ss / n
}
