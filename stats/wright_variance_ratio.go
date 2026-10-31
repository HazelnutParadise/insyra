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
	// PR1, PR2 and PS1 are their equal-tailed two-sided p-values from the
	// simulated null distribution: twice the smaller of (1 + b) / (B + 1) over
	// the draws at or below and at or above the statistic, capped at 1.
	PR1, PR2, PS1 float64
	// Zeros is the number of returns equal to 0, which S1 counts as falls;
	// many of them push S1 up and make it reject a true random walk too often.
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
// and its p-values hold for any sample size, up to simulation error, rather
// than relying on a large sample.
//
// returns are single-period changes, as for VarianceRatio. R1 and R2 assume
// the returns are independent and identically distributed under the null;
// S1 assumes only a martingale difference with median 0, and counts a
// return of exactly 0 as a fall, as Wright does. Zeros then make the signs
// lean negative, which biases S1 upward and makes it reject a true random
// walk too often; Zeros reports how many there were, and with many of them
// R1 and R2 are the ones to read. Tied returns get average ranks.
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
	// below[k][j] and above[k][j] count draws at or below and at or above
	// statistic j for holding period k.
	below := make([][3]int, len(qs))
	above := make([][3]int, len(qs))
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
			sims := [3]float64{wrightPrefixStat(c1, ss1, q), wrightPrefixStat(c2, ss2, q), wrightPrefixStat(cs, nf, q)}
			obs := [3]float64{out[k].R1, out[k].R2, out[k].S1}
			for j := range sims {
				lo, hi := tailSides(sims[j], obs[j])
				if lo {
					below[k][j]++
				}
				if hi {
					above[k][j]++
				}
			}
		}
	}
	for k := range out {
		out[k].PR1 = equalTailP(below[k][0], above[k][0], reps)
		out[k].PR2 = equalTailP(below[k][1], above[k][1], reps)
		out[k].PS1 = equalTailP(below[k][2], above[k][2], reps)
	}
	return out, nil
}
