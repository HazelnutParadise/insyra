package stats

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/HazelnutParadise/insyra"
)

// WrightVarianceRatioOptions are the settings of WrightVarianceRatio.
type WrightVarianceRatioOptions struct {
	// Replications is the number of simulated draws under the null that the
	// p-values come from; 0 means 10,000, the library's choice (it resolves
	// p-values to about 1e-4).
	Replications int
	// Seed starts the random stream and always applies: the zero value is
	// seed 0, so the same input always gives the same p-values.
	Seed uint64
}

// WrightVarianceRatioResult is the result of WrightVarianceRatio for one
// holding period q.
type WrightVarianceRatioResult struct {
	Q int // the holding period, in observations
	// R1 is the statistic on the ranks, R2 on their normal scores and S1 on
	// the signs of the returns. Each is 0 in expectation under the null; a
	// positive value means returns trend, a negative one that they reverse.
	R1, R2, S1 float64
	// PR1, PR2 and PS1 are their two-sided p-values from the simulated null
	// distribution, (1 + b) / (B + 1) for b of B draws at least as extreme.
	PR1, PR2, PS1 float64
	// Zeros is the number of returns equal to 0, which S1 counts as falls.
	Zeros int
}

// WrightVarianceRatioResults is the result of WrightVarianceRatio, one entry
// per holding period in the order given. Show prints it as a table.
type WrightVarianceRatioResults []WrightVarianceRatioResult

const wrightDefaultReplications = 10000

// WrightVarianceRatio tests whether returns follow a random walk with
// Wright's (2000) rank and sign variance ratios. It asks the question of
// VarianceRatio, but on the ranks of the returns (R1), their normal scores
// (R2) and their signs (S1), so outliers and heavy tails do not distort it,
// and its p-values are exact for any sample size rather than asymptotic.
//
// returns are single-period changes, as for VarianceRatio. R1 and R2 assume
// the returns are independent and identically distributed under the null;
// S1 assumes only a martingale difference with median 0, and counts a
// return of exactly 0 as a fall, as Wright does, which biases S1 downward
// when there are many (Zeros reports how many). Tied returns get average
// ranks.
//
// The p-values come from simulating the null: each replication permutes the
// observed R1 and R2 scores, and draws independent signs for S1. Each q must
// be between 2 and the number of returns minus 1. Compare R's
// vrtest::Wright, which gives the same statistics.
func WrightVarianceRatio(returns insyra.IDataList, qs []int, opts ...WrightVarianceRatioOptions) (WrightVarianceRatioResults, error) {
	if len(opts) > 1 {
		return nil, fmt.Errorf("at most one WrightVarianceRatioOptions may be given, got %d", len(opts))
	}
	var o WrightVarianceRatioOptions
	if len(opts) == 1 {
		o = opts[0]
	}
	if o.Replications < 0 {
		return nil, fmt.Errorf("the number of replications must not be negative, got %d", o.Replications)
	}
	reps := o.Replications
	if reps == 0 {
		reps = wrightDefaultReplications
	}
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
	if allEqualF64(x) {
		return nil, errors.New("returns have no variance")
	}

	nf := float64(n)
	ranks, _ := rankWithTies(x)
	r1 := make([]float64, n)
	r2 := make([]float64, n)
	s := make([]float64, n)
	zeros := 0
	scale := math.Sqrt((nf - 1) * (nf + 1) / 12)
	for i, r := range ranks {
		r1[i] = (r - (nf+1)/2) / scale
		r2[i] = zQuantile(r / (nf + 1))
		if x[i] > 0 {
			s[i] = 1
		} else {
			s[i] = -1
			if x[i] == 0 {
				zeros++
			}
		}
	}

	out := make(WrightVarianceRatioResults, len(qs))
	cum1, ss1 := prefixSums(r1)
	cum2, ss2 := prefixSums(r2)
	cumS, ssS := prefixSums(s)
	for k, q := range qs {
		out[k] = WrightVarianceRatioResult{
			Q:     q,
			R1:    wrightPrefixStat(cum1, ss1, q),
			R2:    wrightPrefixStat(cum2, ss2, q),
			S1:    wrightPrefixStat(cumS, ssS, q),
			Zeros: zeros,
		}
	}

	// Simulate the null: one permutation feeds R1 and R2, and S1 gets fresh
	// independent signs. Permuting keeps the sums of squares, so only the
	// prefix sums change.
	rng := rand.New(rand.NewPCG(o.Seed, o.Seed^0x9E3779B97F4A7C15))
	perm := make([]int, n)
	for i := range perm {
		perm[i] = i
	}
	p1 := make([]float64, n)
	p2 := make([]float64, n)
	ps := make([]float64, n)
	exceed := make([][3]int, len(qs))
	for b := 0; b < reps; b++ {
		rng.Shuffle(n, func(i, j int) { perm[i], perm[j] = perm[j], perm[i] })
		for i, src := range perm {
			p1[i] = r1[src]
			p2[i] = r2[src]
			if rng.Uint64()&1 == 1 {
				ps[i] = 1
			} else {
				ps[i] = -1
			}
		}
		c1, _ := prefixSums(p1)
		c2, _ := prefixSums(p2)
		cs, _ := prefixSums(ps)
		for k, q := range qs {
			if atLeastAsExtreme(wrightPrefixStat(c1, ss1, q), out[k].R1) {
				exceed[k][0]++
			}
			if atLeastAsExtreme(wrightPrefixStat(c2, ss2, q), out[k].R2) {
				exceed[k][1]++
			}
			if atLeastAsExtreme(wrightPrefixStat(cs, nf, q), out[k].S1) {
				exceed[k][2]++
			}
		}
	}
	denom := float64(reps + 1)
	for k := range out {
		out[k].PR1 = float64(1+exceed[k][0]) / denom
		out[k].PR2 = float64(1+exceed[k][1]) / denom
		out[k].PS1 = float64(1+exceed[k][2]) / denom
	}
	return out, nil
}
