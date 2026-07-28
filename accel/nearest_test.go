package accel

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func TestNearestQueryCPUMatchesHandComputed(t *testing.T) {
	ds := &Dataset{
		Name: "pts", Lineage: "test", Rows: 3,
		Buffers: []Buffer{
			{Name: "x", Type: DataTypeFloat64, Values: []float64{0, 3, 10}, Len: 3},
			{Name: "y", Type: DataTypeFloat64, Values: []float64{0, 4, 10}, Len: 3},
		},
	}
	assignDatasetFingerprint(ds)

	idx, dist, rows, err := NearestQueryCPU(ds, [][]float32{{0, 0}, {10, 10}})
	if err != nil {
		t.Fatalf("cpu: %v", err)
	}
	if rows != 3 {
		t.Fatalf("expected 3 rows, got %d", rows)
	}
	wantIdx := []uint32{0, 0, 1}
	wantDist := []float32{0, 25, 0}
	for i := range wantIdx {
		if idx[i] != wantIdx[i] || dist[i] != wantDist[i] {
			t.Fatalf("row %d: got (%d, %v), want (%d, %v)", i, idx[i], dist[i], wantIdx[i], wantDist[i])
		}
	}
}

func TestNearestQueryBreaksTiesOnTheLowestIndex(t *testing.T) {
	ds := &Dataset{
		Name: "pts", Lineage: "test", Rows: 1,
		Buffers: []Buffer{{Name: "x", Type: DataTypeFloat64, Values: []float64{0}, Len: 1}},
	}
	assignDatasetFingerprint(ds)

	// Three identical query points: every distance ties, so the first must win.
	idx, _, _, err := NearestQueryCPU(ds, [][]float32{{5}, {5}, {5}})
	if err != nil {
		t.Fatalf("cpu: %v", err)
	}
	if idx[0] != 0 {
		t.Fatalf("expected the lowest index on a tie, got %d", idx[0])
	}
}

func TestNearestQueryAgreesWithTheDistanceMatrix(t *testing.T) {
	rnd := rand.New(rand.NewSource(23))
	ds := distanceDataset(500, 4, rnd)
	queries := randomQueries(6, 4, rnd)

	idx, dist, rows, err := NearestQueryCPU(ds, queries)
	if err != nil {
		t.Fatalf("nearest: %v", err)
	}
	full, _, err := SquaredDistancesCPU(ds, queries)
	if err != nil {
		t.Fatalf("matrix: %v", err)
	}
	for r := 0; r < rows; r++ {
		bestQ, best := 0, full[r]
		for q := 1; q < len(queries); q++ {
			if v := full[q*rows+r]; v < best {
				best, bestQ = v, q
			}
		}
		if idx[r] != uint32(bestQ) || dist[r] != best {
			t.Fatalf("row %d: nearest says (%d, %v), matrix says (%d, %v)", r, idx[r], dist[r], bestQ, best)
		}
	}
}

// The parity gate for this kernel, on whatever platform the test runs.
func TestGPUNearestQueryIsBitIdenticalToTheCPUReference(t *testing.T) {
	requireGPU(t)

	rnd := rand.New(rand.NewSource(29))
	const rows, dims, queries = 200000, 16, 32
	ds := distanceDataset(rows, dims, rnd)
	points := randomQueries(queries, dims, rnd)

	wantIdx, wantDist, _, err := NearestQueryCPU(ds, points)
	if err != nil {
		t.Fatalf("cpu reference: %v", err)
	}

	session := Default()
	if len(session.Devices()) == 0 {
		t.Skip("no GPU discovered on this host")
	}
	result, err := session.ExecuteNearestQuery(ds, points, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !result.Accelerated {
		t.Fatalf("expected GPU execution, got %q", result.FallbackReason)
	}

	for r := 0; r < rows; r++ {
		if result.Index[r] != wantIdx[r] {
			t.Fatalf("row %d: gpu picked query %d, cpu picked %d", r, result.Index[r], wantIdx[r])
		}
		if math.Float32bits(result.Distance[r]) != math.Float32bits(wantDist[r]) {
			t.Fatalf("row %d: gpu distance %v, cpu %v", r, result.Distance[r], wantDist[r])
		}
	}
	t.Logf("bit-identical across %d rows against %d queries; transfer=%v dispatch=%v readback=%v",
		rows, queries, result.Transfer, result.Dispatch, result.Readback)
}

// The whole point of this kernel: readback stops growing with the query count.
func BenchmarkNearestQuery(b *testing.B) {
	rnd := rand.New(rand.NewSource(31))
	const rows, dims = 100000, 16
	ds := distanceDataset(rows, dims, rnd)

	for _, queries := range []int{16, 64} {
		points := randomQueries(queries, dims, rnd)
		b.Run(fmt.Sprintf("q=%d/cpu", queries), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, _, _, err := NearestQueryCPU(ds, points); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("q=%d/gpu", queries), func(b *testing.B) {
			if !gpuTestsEnabled(b) {
				return
			}
			session := Default()
			if len(session.Devices()) == 0 {
				b.Skip("no GPU")
			}
			var readback int64
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := session.ExecuteNearestQuery(ds, points, WorkloadEstimate{Precision: PrecisionFloat32})
				if err != nil {
					b.Fatal(err)
				}
				if !result.Accelerated {
					b.Fatalf("expected GPU execution, got %q", result.FallbackReason)
				}
				readback += result.Readback.Nanoseconds()
			}
			b.StopTimer()
			b.ReportMetric(float64(readback)/float64(b.N)/1e6, "readback_ms/op")
		})
	}
}
