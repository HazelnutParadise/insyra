package wgpu

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"
)

// TestShortlistKernelSmoke checks the WGSL compiles and ranks correctly before
// anything is wired above it.
func TestShortlistKernelSmoke(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	rnd := rand.New(rand.NewSource(7))
	const rows, dims, queryCount, k = 5000, 6, 40, 4

	cols := make([]Column, dims)
	for c := range cols {
		v := make([]float32, rows)
		for i := range v {
			v[i] = float32(rnd.NormFloat64())
		}
		cols[c] = Column{Name: "c", Values: v}
	}
	queries := make([][]float32, queryCount)
	for q := range queries {
		p := make([]float32, dims)
		for c := range p {
			p[c] = float32(rnd.NormFloat64())
		}
		queries[q] = p
	}

	idx, dist, boundary, cost, err := NearestShortlist(context.Background(), cols, queries, k)
	if err != nil {
		if errors.Is(err, ErrUnavailable) {
			t.Skipf("no usable GPU: %v", err)
		}
		t.Fatalf("shortlist failed: %v", err)
	}
	t.Logf("transfer=%v dispatch=%v readback=%v", cost.Transfer, cost.Dispatch, cost.Readback)

	// Reference: every distance in f32, sorted, ties to the lowest index.
	type cand struct {
		d float32
		q int
	}
	bad := 0
	for r := 0; r < rows; r++ {
		all := make([]cand, queryCount)
		for q := 0; q < queryCount; q++ {
			var acc float32
			for c := 0; c < dims; c++ {
				diff := cols[c].Values[r] - queries[q][c]
				acc = acc + diff*diff
			}
			all[q] = cand{acc, q}
		}
		sort.SliceStable(all, func(i, j int) bool { return all[i].d < all[j].d })
		byQuery := make([]float32, queryCount)
		for _, c := range all {
			byQuery[c.q] = c.d
		}
		for j := 0; j < k; j++ {
			gotIdx, gotDist := idx[r*k+j], dist[r*k+j]
			if !shortlistClose(gotDist, all[j].d) || (int(gotIdx) != all[j].q && !shortlistClose(byQuery[gotIdx], all[j].d)) {
				if bad < 5 {
					t.Errorf("row %d slot %d: got (%d, %v), want (%d, %v)",
						r, j, gotIdx, gotDist, all[j].q, all[j].d)
				}
				bad++
			}
		}
		if !shortlistClose(boundary[r], all[k].d) {
			if bad < 5 {
				t.Errorf("row %d boundary: got %v, want %v", r, boundary[r], all[k].d)
			}
			bad++
		}
		if math.IsNaN(float64(boundary[r])) {
			t.Fatalf("row %d boundary is NaN", r)
		}
	}
	if bad > 0 {
		t.Fatalf("%d mismatches", bad)
	}
}

// shortlistClose compares two float32 distances to a few ulps. The reference
// above rounds every multiply and add, while a device may fuse them into one
// rounding, and which it does depends on the platform: Metal and Go on arm64
// fuse, an NVIDIA Vulkan driver fuses, Go on amd64 does not. The kernel only
// proposes a shortlist and the CPU decides in float64 (AGENTS.md), so what the
// test must show is the ranking, not bit equality of the proposals.
func shortlistClose(got, want float32) bool {
	return math.Abs(float64(got)-float64(want)) <= 1e-6*math.Abs(float64(want))+1e-30
}
