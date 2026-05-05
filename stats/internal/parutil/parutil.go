// Package parutil provides lightweight parallel-for primitives used inside
// the stats package's internal compute paths. It avoids the reflection
// overhead of the public parallel.GroupUp helper, which is unsuitable for
// tight numeric inner loops.
package parutil

import (
	"runtime"
	"sync"
)

// minParallelN is the smallest n for which parallel splitting is preferred.
// Below this threshold the goroutine launch + sync overhead exceeds any gain.
// Tuned conservatively — caller should still gate with a domain-specific
// "is the work per i actually expensive" check when items are tiny.
const minParallelN = 256

// For runs fn(i) for i in [0, n) using up to GOMAXPROCS workers, contiguous
// chunking. fn must be safe for concurrent invocation across distinct i's.
//
// Falls back to a serial loop when n is small or GOMAXPROCS == 1.
func For(n int, fn func(i int)) {
	if n <= 0 {
		return
	}
	workers := runtime.GOMAXPROCS(0)
	if workers <= 1 || n < minParallelN {
		for i := range n {
			fn(i)
		}
		return
	}
	workers = min(workers, n)
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

// Chunk splits [0, n) into up to GOMAXPROCS contiguous chunks and calls
// fn(start, end) once per chunk in parallel. Useful for per-chunk reductions
// where the caller maintains a worker-local accumulator.
//
// Falls back to a single in-line call for small n / single-CPU contexts.
func Chunk(n int, fn func(start, end int)) {
	if n <= 0 {
		return
	}
	workers := runtime.GOMAXPROCS(0)
	if workers <= 1 || n < minParallelN {
		fn(0, n)
		return
	}
	workers = min(workers, n)
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
			fn(start, end)
		}(start, end)
	}
	wg.Wait()
}

// NumWorkers returns the worker count For/Chunk would use for n items.
// Callers can size per-worker slices accordingly.
func NumWorkers(n int) int {
	if n <= 0 {
		return 0
	}
	workers := runtime.GOMAXPROCS(0)
	if workers <= 1 || n < minParallelN {
		return 1
	}
	return min(workers, n)
}

// ChunkBounds returns the [start, end) range that worker w (0 ≤ w < workers)
// owns when [0, n) is split contiguously across `workers` workers. Mirrors
// Chunk's slicing exactly so the caller can index a per-worker accumulator
// slice consistently.
func ChunkBounds(n, workers, w int) (int, int) {
	if workers <= 0 || w < 0 || w >= workers {
		return 0, 0
	}
	chunk := (n + workers - 1) / workers
	start := min(w*chunk, n)
	end := min(start+chunk, n)
	return start, end
}
