package stats

import (
	"errors"
	"sync"

	"github.com/HazelnutParadise/insyra/stats/internal/parutil"
)

type oneWayANOVAStats struct {
	SSB float64
	SSW float64
	DFB int
	DFW int
	F   float64
	P   float64
	Eta float64
}

// oneWayANOVAFromSlices computes the one-way ANOVA SSB / SSW / F / p / η²
// from a flat (value, group-label) input pair.
//
// Two parallel reduction phases for n ≥ 4096 (smaller n stays serial — at
// that size the goroutine launch dwarfs the per-iter work):
//
//  1. Group-sum / total-sum / group-count pass: each worker maintains
//     private (groupSums[k], groupCounts[k], totalSum) accumulators and
//     we reduce serially worker-by-worker. Order is fixed (worker 0 →
//     workers-1) so the result is deterministic for a given input.
//  2. SSW pass: per-worker partial Σ(x − μ_g)² then summed serially.
//
// Both reductions sacrifice bit-for-bit equality with the strict
// left-fold serial implementation due to FP non-associativity, but the
// magnitude of the perturbation is O(n·ε) — well below the 1e-9 / 1e-12
// tolerance the existing tests use, and orders of magnitude below the
// statistical sampling noise the test fixtures absorb.
func oneWayANOVAFromSlices(values []float64, labels []int, k int) (*oneWayANOVAStats, error) {
	if k < 2 || len(values) == 0 || len(values) != len(labels) {
		return nil, errors.New("invalid group count or input lengths")
	}

	n := len(values)
	groupSums := make([]float64, k)
	groupCounts := make([]int, k)
	totalSum := 0.0

	const parallelN = 4096

	if n >= parallelN {
		workers := parutil.MaxWorkers(n)
		if workers > 1 {
			partialSums := make([][]float64, workers)
			partialCounts := make([][]int, workers)
			partialTotals := make([]float64, workers)
			rangeErrs := make([]bool, workers)
			var wg sync.WaitGroup
			for w := range workers {
				start, end := parutil.ChunkBounds(n, workers, w)
				if start >= end {
					continue
				}
				wg.Add(1)
				go func(w, start, end int) {
					defer wg.Done()
					ls := make([]float64, k)
					lc := make([]int, k)
					var lt float64
					for i := start; i < end; i++ {
						g := labels[i]
						if g < 0 || g >= k {
							rangeErrs[w] = true
							return
						}
						v := values[i]
						ls[g] += v
						lc[g]++
						lt += v
					}
					partialSums[w] = ls
					partialCounts[w] = lc
					partialTotals[w] = lt
				}(w, start, end)
			}
			wg.Wait()
			for w := range workers {
				if rangeErrs[w] {
					return nil, errors.New("group label out of range")
				}
				if partialSums[w] == nil {
					continue
				}
				for g := range k {
					groupSums[g] += partialSums[w][g]
					groupCounts[g] += partialCounts[w][g]
				}
				totalSum += partialTotals[w]
			}
		} else {
			for i, v := range values {
				g := labels[i]
				if g < 0 || g >= k {
					return nil, errors.New("group label out of range")
				}
				groupSums[g] += v
				groupCounts[g]++
				totalSum += v
			}
		}
	} else {
		for i, v := range values {
			g := labels[i]
			if g < 0 || g >= k {
				return nil, errors.New("group label out of range")
			}
			groupSums[g] += v
			groupCounts[g]++
			totalSum += v
		}
	}

	for _, count := range groupCounts {
		if count == 0 {
			return nil, errors.New("at least one group is empty")
		}
	}

	totalMean := totalSum / float64(n)

	ssb := 0.0
	for i := 0; i < k; i++ {
		mean := groupSums[i] / float64(groupCounts[i])
		ssb += float64(groupCounts[i]) * (mean - totalMean) * (mean - totalMean)
	}

	groupMeans := make([]float64, k)
	for i := range k {
		groupMeans[i] = groupSums[i] / float64(groupCounts[i])
	}

	ssw := 0.0
	if n >= parallelN {
		workers := parutil.MaxWorkers(n)
		if workers > 1 {
			partials := make([]float64, workers)
			var wg sync.WaitGroup
			for w := range workers {
				start, end := parutil.ChunkBounds(n, workers, w)
				if start >= end {
					continue
				}
				wg.Add(1)
				go func(w, start, end int) {
					defer wg.Done()
					var local float64
					for i := start; i < end; i++ {
						d := values[i] - groupMeans[labels[i]]
						local += d * d
					}
					partials[w] = local
				}(w, start, end)
			}
			wg.Wait()
			for _, p := range partials {
				ssw += p
			}
		} else {
			for i, v := range values {
				d := v - groupMeans[labels[i]]
				ssw += d * d
			}
		}
	} else {
		for i, v := range values {
			d := v - groupMeans[labels[i]]
			ssw += d * d
		}
	}

	dfb := k - 1
	dfw := n - k
	if dfb <= 0 || dfw <= 0 {
		return nil, errors.New("invalid degrees of freedom")
	}

	f := fRatio(ssb, dfb, ssw, dfw)
	return &oneWayANOVAStats{
		SSB: ssb,
		SSW: ssw,
		DFB: dfb,
		DFW: dfw,
		F:   f,
		P:   fOneTailedPValue(f, float64(dfb), float64(dfw)),
		Eta: etaSquared(ssb, ssw),
	}, nil
}
