package accel

import (
	"math"
	"math/rand"
	"testing"
)

// 6.4M outputs need ~100k workgroups, past the 65535 limit, so this only passes
// once the dispatch is split.
func TestGPUDistancesBeyondOneDispatch(t *testing.T) {
	requireGPU(t)
	rnd := rand.New(rand.NewSource(17))
	const rows, dims, queries = 100000, 8, 64
	ds := distanceDataset(rows, dims, rnd)
	points := randomQueries(queries, dims, rnd)

	session := Default()
	if len(session.Devices()) == 0 {
		t.Skip("no GPU")
	}
	result, err := session.ExecuteDistances(ds, points, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.Accelerated {
		t.Fatalf("expected GPU execution, got %q", result.FallbackReason)
	}
	want, _, err := SquaredDistancesCPU(ds, points)
	if err != nil {
		t.Fatalf("cpu reference: %v", err)
	}
	for i := range want {
		if math.Float32bits(result.Distances[i]) != math.Float32bits(want[i]) {
			t.Fatalf("value %d differs: gpu %v cpu %v", i, result.Distances[i], want[i])
		}
	}
	t.Logf("bit-identical across %d values spanning multiple dispatches", len(want))
}
