package accel

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func distanceDataset(rows, dims int, rnd *rand.Rand) *Dataset {
	buffers := make([]Buffer, dims)
	for c := 0; c < dims; c++ {
		values := make([]float64, rows)
		for r := range values {
			values[r] = rnd.NormFloat64() * 3.7
		}
		buffers[c] = Buffer{Name: fmt.Sprintf("c%d", c), Type: DataTypeFloat64, Values: values, Len: rows}
	}
	ds := &Dataset{Name: "points", Lineage: "test", Rows: rows, Buffers: buffers}
	assignDatasetFingerprint(ds)
	return ds
}

func randomQueries(count, dims int, rnd *rand.Rand) [][]float32 {
	queries := make([][]float32, count)
	for q := range queries {
		point := make([]float32, dims)
		for c := range point {
			point[c] = float32(rnd.NormFloat64() * 3.7)
		}
		queries[q] = point
	}
	return queries
}

func TestSquaredDistancesCPUMatchesHandComputed(t *testing.T) {
	ds := &Dataset{
		Name: "pts", Lineage: "test", Rows: 3,
		Buffers: []Buffer{
			{Name: "x", Type: DataTypeFloat64, Values: []float64{0, 3, 1}, Len: 3},
			{Name: "y", Type: DataTypeFloat64, Values: []float64{0, 4, 1}, Len: 3},
		},
	}
	assignDatasetFingerprint(ds)

	got, rows, err := SquaredDistancesCPU(ds, [][]float32{{0, 0}})
	if err != nil {
		t.Fatalf("cpu distances: %v", err)
	}
	if rows != 3 {
		t.Fatalf("expected 3 rows, got %d", rows)
	}
	want := []float32{0, 25, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("distance %d: got %v want %v", i, got[i], want[i])
		}
	}
}

func TestExecuteDistancesRejectsBadInput(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	rnd := rand.New(rand.NewSource(1))
	ds := distanceDataset(8, 2, rnd)

	if _, err := session.ExecuteDistances(ds, nil, WorkloadEstimate{Precision: PrecisionFloat32}); err == nil {
		t.Fatal("expected an error with no query points")
	}
	if _, err := session.ExecuteDistances(ds, [][]float32{{1, 2, 3}}, WorkloadEstimate{Precision: PrecisionFloat32}); err == nil {
		t.Fatal("expected an error when a query's dimension does not match the column count")
	}
}

func TestExecuteDistancesRefusesWithoutPrecisionOptIn(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	rnd := rand.New(rand.NewSource(2))
	ds := distanceDataset(512, 3, rnd)

	result, err := session.ExecuteDistances(ds, randomQueries(2, 3, rnd), WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute should not error outside strict mode: %v", err)
	}
	if result.Accelerated {
		t.Fatal("expected float64 columns to be refused without a precision opt-in")
	}
	if result.FallbackReason != FallbackReasonPrecisionNotAccepted {
		t.Fatalf("expected precision-not-accepted, got %q", result.FallbackReason)
	}
}

func TestExecuteDistancesRoutesThroughTheBackend(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	rnd := rand.New(rand.NewSource(3))
	// Large enough to clear the planner's profitability floor.
	ds := distanceDataset(1024, 3, rnd)
	queries := randomQueries(2, 3, rnd)

	want, rows, err := SquaredDistancesCPU(ds, queries)
	if err != nil {
		t.Fatalf("cpu reference: %v", err)
	}

	executor := &distanceFake{}
	if err := RegisterBackendExecutor(BackendCUDA, executor); err != nil {
		t.Fatalf("register executor failed: %v", err)
	}

	result, err := session.ExecuteDistances(ds, queries, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !result.Accelerated {
		t.Fatalf("expected acceleration, got %q", result.FallbackReason)
	}
	if result.Rows != rows || result.Queries != len(queries) {
		t.Fatalf("expected %dx%d, got %dx%d", len(queries), rows, result.Queries, result.Rows)
	}
	if executor.lastReq.Op != OpSquaredDistance {
		t.Fatalf("expected the distance op to reach the backend, got %q", executor.lastReq.Op)
	}
	if len(executor.lastReq.Queries) != len(queries) {
		t.Fatalf("expected the queries to reach the backend, got %d", len(executor.lastReq.Queries))
	}
	for i := range want {
		if result.Distances[i] != want[i] {
			t.Fatalf("distance %d: got %v want %v", i, result.Distances[i], want[i])
		}
	}
}

// distanceFake answers with the same CPU reference the device is held to, so
// the routing test needs no hardware.
type distanceFake struct {
	lastReq ExecuteRequest
}

func (*distanceFake) Name() string { return "distance-fake" }

func (f *distanceFake) Execute(_ context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	f.lastReq = req
	values := make([][]float32, len(req.Columns))
	for i, column := range req.Columns {
		values[i] = column.Values
	}
	rows := 0
	if len(values) > 0 {
		rows = len(values[0])
	}
	return ExecuteResponse{
		Distances:     squaredDistancesReference(values, req.Queries, rows),
		Transfer:      time.Microsecond,
		Dispatch:      time.Microsecond,
		Readback:      time.Microsecond,
		BytesUploaded: uint64(rows * len(values) * 4),
	}, nil
}

// The parity gate. Bit-level agreement between the device and the CPU reference
// is what default-on execution is conditioned on, and it is a property of the
// platform rather than of the kernel — Go emits a fused multiply-add on arm64
// but not on amd64, so a host where the CPU stops fusing while the device keeps
// fusing would diverge. Measuring it here is the point.
func TestGPUDistancesAreBitIdenticalToTheCPUReference(t *testing.T) {
	requireGPU(t)

	rnd := rand.New(rand.NewSource(7))
	const rows, dims, queries = 20000, 16, 8
	ds := distanceDataset(rows, dims, rnd)
	points := randomQueries(queries, dims, rnd)

	want, _, err := SquaredDistancesCPU(ds, points)
	if err != nil {
		t.Fatalf("cpu reference: %v", err)
	}

	session := Default()
	if len(session.Devices()) == 0 {
		t.Skip("no GPU discovered on this host")
	}
	result, err := session.ExecuteDistances(ds, points, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !result.Accelerated {
		t.Fatalf("expected GPU execution, got %q", result.FallbackReason)
	}
	if len(result.Distances) != len(want) {
		t.Fatalf("expected %d distances, got %d", len(want), len(result.Distances))
	}

	mismatch, worstUlp := 0, 0
	for i := range want {
		gb, wb := math.Float32bits(result.Distances[i]), math.Float32bits(want[i])
		if gb == wb {
			continue
		}
		mismatch++
		ulp := int(gb) - int(wb)
		if ulp < 0 {
			ulp = -ulp
		}
		if ulp > worstUlp {
			worstUlp = ulp
		}
	}
	if mismatch != 0 {
		t.Fatalf("parity gate failed on this platform: %d of %d values differ, worst %d ulp",
			mismatch, len(want), worstUlp)
	}
	t.Logf("bit-identical across %d values; transfer=%v dispatch=%v readback=%v",
		len(want), result.Transfer, result.Dispatch, result.Readback)
}

// The query count sets both the compute and the output size, so sweeping it
// shows where the device starts winning.
func BenchmarkSquaredDistances(b *testing.B) {
	rnd := rand.New(rand.NewSource(11))
	const rows, dims = 100000, 16
	ds := distanceDataset(rows, dims, rnd)

	for _, queries := range []int{1, 4, 16, 64} {
		points := randomQueries(queries, dims, rnd)
		b.Run(fmt.Sprintf("q=%d/cpu", queries), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, _, err := SquaredDistancesCPU(ds, points); err != nil {
					b.Fatalf("cpu: %v", err)
				}
			}
		})
		b.Run(fmt.Sprintf("q=%d/gpu", queries), func(b *testing.B) {
			if !gpuTestsEnabled(b) {
				return
			}
			session := Default()
			if len(session.Devices()) == 0 {
				b.Skip("no GPU discovered on this host")
			}
			var readback int64
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := session.ExecuteDistances(ds, points, WorkloadEstimate{Precision: PrecisionFloat32})
				if err != nil {
					b.Fatalf("gpu: %v", err)
				}
				// Without this check a fallback would be timed as if the device
				// had run, which is how an earlier sweep produced a fictional 28x.
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
