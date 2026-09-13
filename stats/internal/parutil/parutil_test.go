package parutil

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

// This package had no test. Every parallel path in stats/internal/clustering
// and stats/internal/knn splits its work through Run and ChunkBounds, so an
// off-by-one here is a wrong statistic, not a crash.

func TestMaxWorkers(t *testing.T) {
	procs := runtime.GOMAXPROCS(0)

	for _, n := range []int{0, -1, -100} {
		if got := MaxWorkers(n); got != 0 {
			t.Errorf("MaxWorkers(%d) = %d, want 0", n, got)
		}
	}
	if got := MaxWorkers(1); got != 1 {
		t.Errorf("MaxWorkers(1) = %d, want 1", got)
	}
	// Never more workers than items, never more than GOMAXPROCS.
	for _, n := range []int{1, 2, 3, 100, 100000} {
		got := MaxWorkers(n)
		if got > n {
			t.Errorf("MaxWorkers(%d) = %d, more workers than items", n, got)
		}
		if got > procs {
			t.Errorf("MaxWorkers(%d) = %d, more than GOMAXPROCS (%d)", n, got, procs)
		}
		if got < 1 {
			t.Errorf("MaxWorkers(%d) = %d, want at least 1 for a positive n", n, got)
		}
	}
}

// The contract ChunkBounds has to satisfy: the ranges of workers 0..workers-1
// tile [0, n) exactly — no gap, no overlap, nothing past the end.
func TestChunkBounds_TilesTheRange(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, 7, 8, 9, 100, 1000} {
		for _, workers := range []int{1, 2, 3, 4, 7, 8, 16} {
			covered := make([]int, n)
			prevEnd := 0
			for w := 0; w < workers; w++ {
				start, end := ChunkBounds(n, workers, w)
				if start > end {
					t.Fatalf("n=%d workers=%d w=%d: start %d > end %d", n, workers, w, start, end)
				}
				if end > n {
					t.Fatalf("n=%d workers=%d w=%d: end %d past n", n, workers, w, end)
				}
				if start != prevEnd {
					t.Fatalf("n=%d workers=%d w=%d: starts at %d, previous worker ended at %d", n, workers, w, start, prevEnd)
				}
				for i := start; i < end; i++ {
					covered[i]++
				}
				prevEnd = end
			}
			if prevEnd != n {
				t.Fatalf("n=%d workers=%d: the last worker ended at %d, want %d", n, workers, prevEnd, n)
			}
			for i, c := range covered {
				if c != 1 {
					t.Fatalf("n=%d workers=%d: index %d covered %d times, want once", n, workers, i, c)
				}
			}
		}
	}
}

// Out-of-range worker numbers and a non-positive worker count give an empty
// range rather than a panic or a slice of someone else's work.
func TestChunkBounds_OutOfRange(t *testing.T) {
	tests := []struct{ n, workers, w int }{
		{n: 10, workers: 4, w: -1},
		{n: 10, workers: 4, w: 4},
		{n: 10, workers: 4, w: 100},
		{n: 10, workers: 0, w: 0},
		{n: 10, workers: -1, w: 0},
	}
	for _, tt := range tests {
		start, end := ChunkBounds(tt.n, tt.workers, tt.w)
		if start != 0 || end != 0 {
			t.Errorf("ChunkBounds(%d, %d, %d) = (%d, %d), want (0, 0)", tt.n, tt.workers, tt.w, start, end)
		}
	}
}

// Run calls fn exactly once for every index, whichever path it takes.
func TestRun_CoversEveryIndexOnce(t *testing.T) {
	for _, goParallel := range []bool{false, true} {
		for _, n := range []int{0, 1, 2, 3, 17, 1000} {
			counts := make([]int32, max(n, 1))
			Run(n, goParallel, func(i int) {
				atomic.AddInt32(&counts[i], 1)
			})
			for i := 0; i < n; i++ {
				if counts[i] != 1 {
					t.Fatalf("goParallel=%v n=%d: index %d called %d times, want once", goParallel, n, i, counts[i])
				}
			}
		}
	}
}

func TestRun_NonPositiveNDoesNothing(t *testing.T) {
	for _, n := range []int{0, -1, -100} {
		called := false
		Run(n, true, func(int) { called = true })
		if called {
			t.Errorf("Run(%d, …) called fn", n)
		}
	}
}

// The serial and parallel paths produce the same answer.
func TestRun_ParallelMatchesSerial(t *testing.T) {
	const n = 5000
	serial := make([]int, n)
	Run(n, false, func(i int) { serial[i] = i * i })

	parallel := make([]int, n)
	Run(n, true, func(i int) { parallel[i] = i * i })

	for i := range serial {
		if serial[i] != parallel[i] {
			t.Fatalf("index %d: serial %d, parallel %d", i, serial[i], parallel[i])
		}
	}
}

// A caller that manages its own goroutines slices the range with ChunkBounds
// and must land on the same partition Run uses. This is the documented reason
// ChunkBounds exists, and nothing checked the two agree.
func TestChunkBounds_MatchesRunsPartition(t *testing.T) {
	const n = 997
	workers := runtime.GOMAXPROCS(0)
	if workers > n {
		workers = n
	}

	var mu sync.Mutex
	byRun := make([]int, 0, n)
	Run(n, true, func(i int) {
		mu.Lock()
		byRun = append(byRun, i)
		mu.Unlock()
	})

	byBounds := make([]int, 0, n)
	for w := 0; w < workers; w++ {
		start, end := ChunkBounds(n, workers, w)
		for i := start; i < end; i++ {
			byBounds = append(byBounds, i)
		}
	}

	if len(byRun) != len(byBounds) {
		t.Fatalf("Run covered %d indices, ChunkBounds covered %d", len(byRun), len(byBounds))
	}
	seen := make(map[int]bool, n)
	for _, i := range byBounds {
		seen[i] = true
	}
	for _, i := range byRun {
		if !seen[i] {
			t.Fatalf("Run visited index %d, which no ChunkBounds range covers", i)
		}
	}
}
