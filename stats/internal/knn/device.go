package knn

import "sync/atomic"

// DeviceSearcher answers every test row's k nearest training rows in one
// batch: indices[i] and squared distances dist2[i] hold row i's k neighbours,
// nearest first, ties broken by ascending training index — the same order the
// CPU searchers report. ok=false means the searcher declines this shape and
// the CPU path runs instead.
//
// This is a socket, not an import: `stats` knows nothing about what fills it.
// The accelerator bridge registers itself on blank import, which keeps the
// device stack — 41 packages and 1.9s of cold build — off every stats user
// who did not ask for it.
type DeviceSearcher func(train, test [][]float64, k int) (indices [][]int, dist2 [][]float64, ok bool)

var deviceSearcher atomic.Value // of DeviceSearcher

// RegisterDeviceSearcher installs the batch searcher consulted by
// auto-algorithm queries. Passing nil uninstalls it.
func RegisterDeviceSearcher(fn DeviceSearcher) {
	deviceSearcher.Store(fn)
}

// deviceBatch consults the registered searcher and converts its answer into
// per-row neighbour lists. Only the auto algorithm consults it: a caller who
// named brute force or a tree asked for that specific machine, and quietly
// substituting another would make the option a lie.
//
// The answer is validated defensively. The searcher is registered from
// outside this package, so a bug there must degrade to the CPU path, not
// panic a stats call.
func deviceBatch(train, test [][]float64, k int, opts Options) [][]neighbor {
	if opts.Algorithm != AutoAlgorithm {
		return nil
	}
	stored := deviceSearcher.Load()
	if stored == nil {
		return nil
	}
	fn, _ := stored.(DeviceSearcher)
	if fn == nil {
		return nil
	}
	indices, dist2, ok := fn(train, test, k)
	if !ok {
		return nil
	}
	if len(indices) != len(test) || len(dist2) != len(test) {
		return nil
	}
	batch := make([][]neighbor, len(test))
	for i := range indices {
		if len(indices[i]) != k || len(dist2[i]) != k {
			return nil
		}
		row := make([]neighbor, k)
		for j := 0; j < k; j++ {
			index := indices[i][j]
			if index < 0 || index >= len(train) {
				return nil
			}
			row[j] = neighbor{index: index, dist2: dist2[i][j]}
		}
		batch[i] = row
	}
	return batch
}
