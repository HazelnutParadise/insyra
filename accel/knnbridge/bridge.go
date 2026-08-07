// Package knnbridge plugs the accelerator into stats' KNN device socket.
//
// Import it for its side effect:
//
//	import _ "github.com/HazelnutParadise/insyra/accel/knnbridge"
//
// With the import, auto-algorithm KNN routes profitable shapes through the
// exact-nearest device operation — whose answers are recomputed in float64 on
// the host, so results are identical to brute force, only faster. Without it,
// stats carries no accelerator dependency and behaves exactly as before.
package knnbridge

import (
	"fmt"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/accel"
	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
	"github.com/HazelnutParadise/insyra/stats"
)

func init() {
	stats.RegisterKNNDeviceSearcher(search)
}

// minWorkPerRow mirrors the profitability floor measured in accel/exact.go:
// below ~2048 distance evaluations per row the round trip costs more than the
// work it removes. The runtime re-checks profitability itself; this pre-gate
// exists so unprofitable shapes skip the dataset construction entirely.
const minWorkPerRow = 2048

// minTestRows is the second floor, measured in the wiring's own direction
// (BenchmarkKNNTrueDirection, M3, 2026-08-01). The kernel's parallelism comes
// from dataset rows — the test set here — and its wall time is nearly flat in
// them until the device saturates: 100k×32 took ~467ms whether 1k, 2k or 4k
// test rows were asked. Below ~2k test rows that flat cost exceeds the CPU's
// whole job (1k test rows: device 469ms vs CPU 324ms — a loss), above it the
// device pulls ahead (2k: 1.4x, 4k: 2.9x, 10k: 3.7x). The transposed
// benchmark that first justified this wiring had the winning region larger
// than it really is; this floor is what the true direction supports.
const minTestRows = 2048

func search(train, test [][]float64, k int) ([][]int, [][]float64, bool) {
	if !insyra.Config.GetAccelerationEnabled() {
		return nil, nil, false
	}
	// One verification slot is reserved past the shortlist, so the device can
	// carry at most MaxShortlist-1 requested neighbours.
	if k > wgpu.MaxShortlist-1 {
		return nil, nil, false
	}
	if len(train) == 0 || len(test) == 0 {
		return nil, nil, false
	}
	if len(test) < minTestRows {
		return nil, nil, false
	}
	dims := len(test[0])
	if len(train)*dims < minWorkPerRow {
		return nil, nil, false
	}
	session := accel.Default()
	if len(session.Devices()) == 0 {
		return nil, nil, false
	}

	// Roles: each TEST row wants its k nearest TRAINING rows, and the
	// operation answers, per dataset row, the m nearest query points — so the
	// test set is the dataset and the training set is the queries.
	buffers := make([]accel.Buffer, dims)
	for c := 0; c < dims; c++ {
		values := make([]float64, len(test))
		for i, row := range test {
			if len(row) != dims {
				return nil, nil, false
			}
			values[i] = row[c]
		}
		buffers[c] = accel.Buffer{
			Name:   fmt.Sprintf("f%d", c),
			Type:   accel.DataTypeFloat64,
			Values: values,
			Len:    len(test),
		}
	}
	dataset := &accel.Dataset{
		Name:    "knn-test-rows",
		Lineage: "stats/knn",
		Rows:    len(test),
		Buffers: buffers,
	}

	result, err := accel.Default().ExecuteNearestExact(dataset, train, k, accel.WorkloadEstimate{Rows: len(test)})
	if err != nil {
		return nil, nil, false
	}
	// An answer the runtime did not accelerate is still correct — the
	// operation always answers — but taking it would silently replace stats'
	// own auto-selected searcher with a brute-force scan, changing performance
	// while claiming to change nothing. Declined means declined.
	if !result.Accelerated {
		return nil, nil, false
	}

	indices := make([][]int, len(test))
	distances := make([][]float64, len(test))
	for i := 0; i < len(test); i++ {
		indices[i] = make([]int, k)
		distances[i] = make([]float64, k)
		for j := 0; j < k; j++ {
			indices[i][j] = int(result.Index[i*result.M+j])
			distances[i][j] = result.Distance[i*result.M+j]
		}
	}
	return indices, distances, true
}
