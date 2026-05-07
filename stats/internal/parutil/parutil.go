// Package parutil provides lightweight parallel-for primitives used inside
// the stats package's internal compute paths. It avoids the reflection
// overhead of the public parallel.GroupUp helper, which is unsuitable for
// tight numeric inner loops.
//
// API design note: parutil performs NO auto-threshold. Every caller is
// responsible for deciding whether the workload is large enough to amortise
// goroutine launch — the right cutoff is wildly different across hot paths
// (kendall pair count breaks even at n≈192, pickClosestPair at m≈512,
// KMeans-init assignment at n·k·p≈50000, brute distance matrix at n²·p≈200K).
// A single global threshold would systematically miss wins or mis-fire.
// Each caller's decision must be backed by data — see
// stats/calibration_bench_test.go for the methodology.
package parutil

import (
	"runtime"
	"sync"
)

// MaxWorkers returns the worker count to use for n items, capped at
// GOMAXPROCS. Always ≥ 1 for n > 0. Applies NO threshold.
func MaxWorkers(n int) int {
	if n <= 0 {
		return 0
	}
	w := runtime.GOMAXPROCS(0)
	w = min(w, n)
	if w < 1 {
		w = 1
	}
	return w
}

// Run runs fn(i) for i in [0, n). When goParallel is true, splits across
// up to MaxWorkers(n) goroutines; otherwise runs a plain serial loop.
// The goParallel decision should come from calibration data.
//
// fn must be safe for concurrent invocation across distinct i's when
// goParallel is true.
func Run(n int, goParallel bool, fn func(i int)) {
	if n <= 0 {
		return
	}
	workers := runtime.GOMAXPROCS(0)
	if !goParallel || workers <= 1 {
		for i := range n {
			fn(i)
		}
		return
	}
	if workers > n {
		workers = n
	}
	chunk := (n + workers - 1) / workers
	var wg sync.WaitGroup
	for w := range workers {
		start := w * chunk
		if start >= n {
			break
		}
		end := min(start+chunk, n)
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				fn(i)
			}
		}(start, end)
	}
	wg.Wait()
}

// ChunkBounds returns the [start, end) range that worker w (0 ≤ w < workers)
// owns when [0, n) is split contiguously across `workers` workers. Mirrors
// the slicing used internally by Run — useful when the caller manages its
// own goroutines (e.g. for per-worker accumulators).
func ChunkBounds(n, workers, w int) (int, int) {
	if workers <= 0 || w < 0 || w >= workers {
		return 0, 0
	}
	chunk := (n + workers - 1) / workers
	start := min(w*chunk, n)
	end := min(start+chunk, n)
	return start, end
}
