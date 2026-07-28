package accel

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"
)

// shortlistExecutor computes the shortlist the way the kernel does — single
// precision, ties to the lowest query index — so the float64 decision logic can
// be tested on machines with no GPU.
type shortlistExecutor struct{ calls int }

func (*shortlistExecutor) Name() string { return "cpu-shortlist" }

func (e *shortlistExecutor) Execute(_ context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	e.calls++
	rows := len(req.Columns[0].Values)
	k := req.Shortlist
	idx := make([]uint32, rows*k)
	dist := make([]float32, rows*k)
	boundary := make([]float32, rows)

	type cand struct {
		d float32
		q uint32
	}
	all := make([]cand, len(req.Queries))
	for r := 0; r < rows; r++ {
		for q, query := range req.Queries {
			var acc float32
			for c, column := range req.Columns {
				diff := column.Values[r] - query[c]
				acc = acc + diff*diff
			}
			all[q] = cand{acc, uint32(q)}
		}
		sort.SliceStable(all, func(i, j int) bool { return all[i].d < all[j].d })
		for j := 0; j < k; j++ {
			idx[r*k+j] = all[j].q
			dist[r*k+j] = all[j].d
		}
		if k < len(all) {
			boundary[r] = all[k].d
		} else {
			boundary[r] = math.MaxFloat32
		}
	}
	return ExecuteResponse{ShortlistIndex: idx, ShortlistDistance: dist, ShortlistBoundary: boundary}, nil
}

// exerciseDeviceRegardlessOfProfit lowers the profitability floor for tests that
// are about correctness rather than about when the device is worth using. The
// floor has its own test; making every correctness test carry enough work to
// clear it would only make them slower and no more convincing.
func exerciseDeviceRegardlessOfProfit(t *testing.T) {
	t.Helper()
	previous := minWorkPerRowForDevice
	minWorkPerRowForDevice = 0
	t.Cleanup(func() { minWorkPerRowForDevice = previous })
}

func exactDataset(rows, dims int, rnd *rand.Rand) *Dataset {
	buffers := make([]Buffer, dims)
	for c := 0; c < dims; c++ {
		values := make([]float64, rows)
		for i := range values {
			values[i] = rnd.NormFloat64()
		}
		buffers[c] = Buffer{Name: string(rune('a' + c)), Type: DataTypeFloat64, Values: values, Len: rows}
	}
	return &Dataset{Name: "exact", Lineage: "test", Rows: rows, Buffers: buffers}
}

func exactQueries(count, dims int, rnd *rand.Rand) [][]float64 {
	queries := make([][]float64, count)
	for q := range queries {
		point := make([]float64, dims)
		for c := range point {
			point[c] = rnd.NormFloat64()
		}
		queries[q] = point
	}
	return queries
}

func assertMatchesReference(t *testing.T, ds *Dataset, queries [][]float64, m int, gotIdx []uint32, gotDist []float64) {
	t.Helper()
	wantIdx, wantDist, rows, err := NearestExactCPU(ds, queries, m)
	if err != nil {
		t.Fatalf("reference failed: %v", err)
	}
	if len(gotIdx) != rows*m || len(gotDist) != rows*m {
		t.Fatalf("expected %d entries, got %d indices and %d distances", rows*m, len(gotIdx), len(gotDist))
	}
	for i := range wantIdx {
		if gotIdx[i] != wantIdx[i] || gotDist[i] != wantDist[i] {
			t.Fatalf("row %d slot %d: got (%d, %v), want (%d, %v)",
				i/m, i%m, gotIdx[i], gotDist[i], wantIdx[i], wantDist[i])
		}
	}
}

func TestExactNearestMatchesTheReference(t *testing.T) {
	exerciseDeviceRegardlessOfProfit(t)
	session := singleDeviceSession(t, Config{})
	shortlist := &shortlistExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, shortlist); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	rnd := rand.New(rand.NewSource(21))
	ds := exactDataset(4096, 5, rnd)
	queries := exactQueries(40, 5, rnd)

	for _, m := range []int{1, 2, 4} {
		result, err := session.ExecuteNearestExact(ds, queries, m, WorkloadEstimate{})
		if err != nil {
			t.Fatalf("m=%d: %v", m, err)
		}
		if !result.Accelerated {
			t.Fatalf("m=%d: expected the shortlist backend to run, got %q", m, result.FallbackReason)
		}
		if result.Precision != PrecisionExact {
			t.Fatalf("m=%d: the answer is exact, so it should say so, got %q", m, result.Precision)
		}
		assertMatchesReference(t, ds, queries, m, result.Index, result.Distance)
		t.Logf("m=%d rechecked %d of %d rows", m, result.Rechecked, result.Rows)
	}
	if shortlist.calls == 0 {
		t.Fatal("the shortlist backend was never reached")
	}
}

// TestExactNearestOnAdversarialData aims at the boundary the shortlist has to
// get right: duplicated rows, query points that are exactly equidistant, and
// magnitudes far enough apart that single precision loses the difference.
func TestExactNearestOnAdversarialData(t *testing.T) {
	exerciseDeviceRegardlessOfProfit(t)
	session := singleDeviceSession(t, Config{})
	if err := RegisterBackendExecutor(BackendCUDA, &shortlistExecutor{}); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Two columns. Query points sit on a circle so that many rows are very
	// nearly equidistant from several of them, and every row is duplicated.
	const rows, queryCount = 2048, 40
	rnd := rand.New(rand.NewSource(22))
	x := make([]float64, rows)
	y := make([]float64, rows)
	for i := 0; i < rows; i += 2 {
		// A magnitude wide enough that x+d and x are the same float32.
		scale := math.Pow(10, float64(rnd.Intn(9)-4))
		x[i] = rnd.NormFloat64() * scale
		y[i] = rnd.NormFloat64() * scale
		x[i+1] = x[i]
		y[i+1] = y[i]
	}
	ds := &Dataset{Name: "adversarial", Lineage: "test", Rows: rows, Buffers: []Buffer{
		{Name: "x", Type: DataTypeFloat64, Values: x, Len: rows},
		{Name: "y", Type: DataTypeFloat64, Values: y, Len: rows},
	}}
	queries := make([][]float64, queryCount)
	for q := range queries {
		angle := 2 * math.Pi * float64(q) / float64(queryCount)
		queries[q] = []float64{math.Cos(angle), math.Sin(angle)}
	}

	for _, m := range []int{1, 2} {
		result, err := session.ExecuteNearestExact(ds, queries, m, WorkloadEstimate{})
		if err != nil {
			t.Fatalf("m=%d: %v", m, err)
		}
		assertMatchesReference(t, ds, queries, m, result.Index, result.Distance)
		t.Logf("m=%d rechecked %d of %d rows (%.2f%%)",
			m, result.Rechecked, result.Rows, 100*float64(result.Rechecked)/float64(result.Rows))
	}
}

// TestExactNearestTiesGoToTheLowestIndex pins the tie rule with query points
// placed so the tie is exact in float64, not merely close.
func TestExactNearestTiesGoToTheLowestIndex(t *testing.T) {
	exerciseDeviceRegardlessOfProfit(t)
	session := singleDeviceSession(t, Config{})
	if err := RegisterBackendExecutor(BackendCUDA, &shortlistExecutor{}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	const rows = 1024
	values := make([]float64, rows)
	for i := range values {
		values[i] = 0.5
	}
	ds := &Dataset{Name: "ties", Lineage: "test", Rows: rows, Buffers: []Buffer{
		{Name: "x", Type: DataTypeFloat64, Values: values, Len: rows},
	}}
	// Every query is the same distance from 0.5, repeated past the
	// profitability floor so the device leg is the one under test.
	queries := make([][]float64, 40)
	for q := range queries {
		queries[q] = []float64{float64(q % 2)}
	}

	result, err := session.ExecuteNearestExact(ds, queries, 2, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	for r := 0; r < rows; r++ {
		if result.Index[r*2] != 0 || result.Index[r*2+1] != 1 {
			t.Fatalf("row %d: got (%d, %d), want (0, 1)", r, result.Index[r*2], result.Index[r*2+1])
		}
	}
	assertMatchesReference(t, ds, queries, 2, result.Index, result.Distance)
}

func TestExactNearestWithoutADevice(t *testing.T) {
	session := noDeviceSession(t, Config{})
	rnd := rand.New(rand.NewSource(23))
	ds := exactDataset(2048, 4, rnd)
	queries := exactQueries(16, 4, rnd)

	result, err := session.ExecuteNearestExact(ds, queries, 2, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute should not error outside strict mode: %v", err)
	}
	if result.Accelerated {
		t.Fatal("nothing was discovered, so nothing can have been accelerated")
	}
	if result.Rechecked != result.Rows {
		t.Fatalf("with no device every row takes the full path, got %d of %d", result.Rechecked, result.Rows)
	}
	assertMatchesReference(t, ds, queries, 2, result.Index, result.Distance)
}

func TestExactNearestRejectsBadInput(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	rnd := rand.New(rand.NewSource(24))
	ds := exactDataset(64, 3, rnd)
	queries := exactQueries(2, 3, rnd)

	if _, err := session.ExecuteNearestExact(ds, queries, 3, WorkloadEstimate{}); err == nil {
		t.Fatal("expected an error when asking for more neighbours than query points")
	}
	if _, err := session.ExecuteNearestExact(ds, queries, 0, WorkloadEstimate{}); err == nil {
		t.Fatal("expected an error when asking for no neighbours")
	}
	if _, err := session.ExecuteNearestExact(ds, [][]float64{{1, 2}}, 1, WorkloadEstimate{}); err == nil {
		t.Fatal("expected an error when a query's dimension does not match the column count")
	}
}

// TestExactNearestOnGPU is the one that matters: a real device narrowing the
// field, and the answer still equal to pure float64.
func TestExactNearestOnGPU(t *testing.T) {
	requireGPU(t)
	exerciseDeviceRegardlessOfProfit(t)
	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	defer func() { _ = session.Close() }()

	rnd := rand.New(rand.NewSource(25))
	ds := exactDataset(100_000, 8, rnd)
	queries := exactQueries(48, 8, rnd)

	for _, m := range []int{1, 2} {
		result, err := session.ExecuteNearestExact(ds, queries, m, WorkloadEstimate{})
		if err != nil {
			t.Fatalf("m=%d: %v", m, err)
		}
		if !result.Accelerated {
			t.Fatalf("m=%d: expected device execution, got %q", m, result.FallbackReason)
		}
		assertMatchesReference(t, ds, queries, m, result.Index, result.Distance)
		t.Logf("m=%d rechecked %d of %d rows; transfer=%v dispatch=%v readback=%v",
			m, result.Rechecked, result.Rows, result.Transfer, result.Dispatch, result.Readback)
	}
}

// BenchmarkExactNearest compares the exact answer through a device against the
// exact answer without one, across query counts. The device's share of the work
// grows with the query count while the data it uploads does not, so the shape of
// the sweep says whether the operation is compute-bound or transfer-bound.
func BenchmarkExactNearest(b *testing.B) {
	if !gpuTestsEnabled(b) {
		b.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	session, err := Open(Config{})
	if err != nil {
		b.Fatalf("open failed: %v", err)
	}
	defer func() { _ = session.Close() }()

	rnd := rand.New(rand.NewSource(26))
	ds := exactDataset(200_000, 16, rnd)

	for _, queryCount := range []int{16, 64, 256} {
		queries := exactQueries(queryCount, 16, rnd)
		b.Run(fmt.Sprintf("q=%d/cpu-float64", queryCount), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, _, _, err := NearestExactCPU(ds, queries, 2); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("q=%d/gpu-shortlist", queryCount), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				result, err := session.ExecuteNearestExact(ds, queries, 2, WorkloadEstimate{})
				if err != nil {
					b.Fatal(err)
				}
				b.ReportMetric(float64(result.Rechecked), "rechecked")
				if !result.Accelerated {
					b.ReportMetric(0, "ondevice")
				} else {
					b.ReportMetric(1, "ondevice")
				}
			}
		})
	}
}

// TestExactNearestDeclinesTinyQuerySets pins the profitability floor. Below it
// the round trip costs more than the work it removes, and the contract says not
// to use the device for work it loses.
func TestExactNearestDeclinesTinyQuerySets(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	backend := &shortlistExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, backend); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	rnd := rand.New(rand.NewSource(27))
	ds := exactDataset(4096, 4, rnd)
	queries := exactQueries(minWorkPerRowForDevice/4-1, 4, rnd)

	result, err := session.ExecuteNearestExact(ds, queries, 2, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if result.Accelerated {
		t.Fatal("expected the runtime to decline a query set this small")
	}
	if result.FallbackReason != FallbackReasonWorkloadNotProfitable {
		t.Fatalf("expected workload-not-profitable, got %q", result.FallbackReason)
	}
	if backend.calls != 0 {
		t.Fatal("the device should not have been reached at all")
	}
	assertMatchesReference(t, ds, queries, 2, result.Index, result.Distance)
}

// TestExactNearestRechecksWhenTheCutIsTooClose proves the safety net engages.
// Forty query points sit at the same distance from every row, so a shortlist of
// three carries no information about which of the other thirty-seven might tie
// it, and the boundary test has to send every row down the full path.
func TestExactNearestRechecksWhenTheCutIsTooClose(t *testing.T) {
	exerciseDeviceRegardlessOfProfit(t)
	session := singleDeviceSession(t, Config{})
	if err := RegisterBackendExecutor(BackendCUDA, &shortlistExecutor{}); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	const rows, queryCount = 512, 40
	x := make([]float64, rows)
	y := make([]float64, rows)
	ds := &Dataset{Name: "equidistant", Lineage: "test", Rows: rows, Buffers: []Buffer{
		{Name: "x", Type: DataTypeFloat64, Values: x, Len: rows},
		{Name: "y", Type: DataTypeFloat64, Values: y, Len: rows},
	}}
	queries := make([][]float64, queryCount)
	for q := range queries {
		angle := 2 * math.Pi * float64(q) / float64(queryCount)
		queries[q] = []float64{math.Cos(angle), math.Sin(angle)}
	}

	result, err := session.ExecuteNearestExact(ds, queries, 1, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !result.Accelerated {
		t.Fatalf("expected the shortlist backend to run, got %q", result.FallbackReason)
	}
	if result.Rechecked != rows {
		t.Fatalf("every row's cut is a tie, so every row should be rechecked; got %d of %d",
			result.Rechecked, rows)
	}
	assertMatchesReference(t, ds, queries, 1, result.Index, result.Distance)
}
