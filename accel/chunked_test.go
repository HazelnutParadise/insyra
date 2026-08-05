package accel

import (
	"context"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
)

func TestExactNearestChunkRanges(t *testing.T) {
	tests := []struct {
		name    string
		rows    int
		maxRows int
		want    []nearestRowRange
	}{
		{name: "empty", rows: 0, maxRows: exactNearestChunkRows},
		{name: "one", rows: 1, maxRows: exactNearestChunkRows, want: []nearestRowRange{{0, 1}}},
		{name: "remainder", rows: 10, maxRows: 4, want: []nearestRowRange{{0, 4}, {4, 8}, {8, 10}}},
		{name: "bound-minus-one", rows: exactNearestChunkRows - 1, maxRows: exactNearestChunkRows, want: []nearestRowRange{{0, exactNearestChunkRows - 1}}},
		{name: "bound", rows: exactNearestChunkRows, maxRows: exactNearestChunkRows, want: []nearestRowRange{{0, exactNearestChunkRows}}},
		{name: "bound-plus-one", rows: exactNearestChunkRows + 1, maxRows: exactNearestChunkRows, want: []nearestRowRange{{0, exactNearestChunkRows}, {exactNearestChunkRows, exactNearestChunkRows + 1}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := exactNearestChunkRanges(test.rows, test.maxRows); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ranges(%d, %d) = %#v, want %#v", test.rows, test.maxRows, got, test.want)
			}
		})
	}
}

func TestExactNearestUsesOneSubmissionAtTheBound(t *testing.T) {
	exerciseDeviceRegardlessOfProfit(t)
	session := singleDeviceSession(t, Config{})
	backend := &shortlistExecutor{}
	if err := RegisterBackendExecutor(BackendCUDA, backend); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	rnd := rand.New(rand.NewSource(20260805))
	queries := exactQueries(4, 1, rnd)

	for _, rows := range []int{exactNearestChunkRows, exactNearestChunkRows + 1} {
		dataset := exactDataset(rows, 1, rnd)
		before := backend.calls
		result, err := session.ExecuteNearestExact(dataset, queries, 1, WorkloadEstimate{})
		if err != nil {
			t.Fatalf("rows=%d: execute failed: %v", rows, err)
		}
		wantChunks := 1
		if rows > exactNearestChunkRows {
			wantChunks = 2
		}
		if result.Chunks != wantChunks {
			t.Fatalf("rows=%d: got %d chunks, want %d", rows, result.Chunks, wantChunks)
		}
		if backend.calls-before != wantChunks {
			t.Fatalf("rows=%d: got %d backend calls, want %d", rows, backend.calls-before, wantChunks)
		}
		assertMatchesReference(t, dataset, queries, 1, result.Index, result.Distance)
	}
}

func TestExactNearestChunkMergePreservesOrderAndParity(t *testing.T) {
	rnd := rand.New(rand.NewSource(20260805))
	dataset := exactDataset(7, 3, rnd)
	queries := exactQueries(9, 3, rnd)
	columns, _, reason := narrowColumns(dataset)
	if reason != FallbackReasonNone {
		t.Fatalf("narrow columns failed: %s", reason)
	}
	narrowed := narrowQueries(queries)
	device := Device{ID: "test-device", Backend: BackendCUDA}
	request := ExecuteRequest{
		Op:        OpNearestShortlist,
		Device:    device,
		Columns:   columns,
		Queries:   narrowed,
		Precision: PrecisionFloat32,
		Shortlist: 4,
	}

	singleExecutor := &shortlistExecutor{}
	unchunked, err := singleExecutor.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("single submission failed: %v", err)
	}
	chunkedExecutor := &shortlistExecutor{}
	chunked, chunks, err := executeNearestShortlistChunks(
		context.Background(), chunkedExecutor, device, columns, narrowed, PrecisionFloat32, 4, 3)
	if err != nil {
		t.Fatalf("chunked submission failed: %v", err)
	}
	if chunks != 3 || chunkedExecutor.calls != 3 {
		t.Fatalf("got %d chunks and %d calls, want 3", chunks, chunkedExecutor.calls)
	}
	if !reflect.DeepEqual(chunked.ShortlistIndex, unchunked.ShortlistIndex) ||
		!reflect.DeepEqual(chunked.ShortlistDistance, unchunked.ShortlistDistance) ||
		!reflect.DeepEqual(chunked.ShortlistBoundary, unchunked.ShortlistBoundary) {
		t.Fatal("chunked shortlist did not preserve the unchunked row order")
	}

	host, rows, reason := hostColumns(dataset)
	if reason != FallbackReasonNone {
		t.Fatalf("host columns failed: %s", reason)
	}
	singleIndex, singleDistance, singleRechecked := decideFromShortlist(
		host, queries, rows, 2, 4, unchunked.ShortlistIndex, unchunked.ShortlistDistance, unchunked.ShortlistBoundary)
	chunkedIndex, chunkedDistance, chunkedRechecked := decideFromShortlist(
		host, queries, rows, 2, 4, chunked.ShortlistIndex, chunked.ShortlistDistance, chunked.ShortlistBoundary)
	if !reflect.DeepEqual(chunkedIndex, singleIndex) || !reflect.DeepEqual(chunkedDistance, singleDistance) || chunkedRechecked != singleRechecked {
		t.Fatal("chunked exact decision did not match the unchunked decision")
	}
}

func TestExactNearestChunked64K128Hardware(t *testing.T) {
	requireHardwareGPU(t)
	session, err := Open(Config{})
	if err != nil && (session == nil || len(session.Devices()) == 0) {
		t.Skipf("no usable GPU backend: %v", err)
	}
	if session == nil {
		t.Skip("no session")
	}
	t.Cleanup(func() { _ = session.Close() })
	device, ok := saturationDevice(session.Devices())
	if !ok {
		t.Skip("only a software, virtual, or environment-stub adapter was discovered")
	}
	if reason, ok := saturationMemoryAbort(64_000, 128, device); ok {
		t.Skipf("resource preflight: %s", reason)
	}

	rnd := rand.New(rand.NewSource(20260805 + 128 + 64_000))
	dataset := exactDataset(64_000, 128, rnd)
	queries := exactQueries(100_000, 128, rnd)
	start := time.Now()
	result, err := session.ExecuteNearestExact(dataset, queries, 5, WorkloadEstimate{Rows: 64_000})
	wall := time.Since(start)
	if err != nil {
		t.Fatalf("chunked 64k×128 execution failed: %v", err)
	}
	if !result.Accelerated {
		t.Fatalf("expected device execution, got fallback %q", result.FallbackReason)
	}
	if result.Chunks != 4 {
		t.Fatalf("got %d chunks, want 4", result.Chunks)
	}
	wantIndex, wantDistance, _, err := NearestExactCPU(dataset, queries, 5)
	if err != nil {
		t.Fatalf("brute-force reference failed: %v", err)
	}
	if !reflect.DeepEqual(result.Index, wantIndex) || !reflect.DeepEqual(result.Distance, wantDistance) {
		t.Fatal("chunked hardware result differs from brute force")
	}
	t.Logf("chunked 64k×128 wall_ms=%.3f chunks=%d", float64(wall)/float64(time.Millisecond), result.Chunks)
}

func TestExactNearestChunkedOverheadHardware(t *testing.T) {
	requireHardwareGPU(t)
	session, err := Open(Config{})
	if err != nil && (session == nil || len(session.Devices()) == 0) {
		t.Skipf("no usable GPU backend: %v", err)
	}
	if session == nil {
		t.Skip("no session")
	}
	t.Cleanup(func() { _ = session.Close() })
	devices := session.Devices()
	device, ok := saturationDevice(devices)
	if !ok {
		t.Skip("only a software, virtual, or environment-stub adapter was discovered")
	}
	if reason, ok := saturationMemoryAbort(32_000, 32, device); ok {
		t.Skipf("resource preflight: %s", reason)
	}

	rnd := rand.New(rand.NewSource(20260805 + 32 + 32_000))
	dataset := exactDataset(32_000, 32, rnd)
	queries := exactQueries(100_000, 32, rnd)
	productionStart := time.Now()
	production, err := session.ExecuteNearestExact(dataset, queries, 5, WorkloadEstimate{Rows: 32_000})
	productionWall := time.Since(productionStart)
	if err != nil {
		t.Fatalf("production chunked execution failed: %v", err)
	}
	if !production.Accelerated || production.Chunks != 2 {
		t.Fatalf("expected two chunked device submissions, got accelerated=%t chunks=%d fallback=%q", production.Accelerated, production.Chunks, production.FallbackReason)
	}

	columns, _, reason := narrowColumns(dataset)
	if reason != FallbackReasonNone {
		t.Fatalf("narrow columns failed: %s", reason)
	}
	executor, ok := lookupBackendExecutor(device.Backend)
	if !ok {
		t.Fatalf("no executor for %q", device.Backend)
	}
	narrowed := narrowQueries(queries)
	chunkStart := time.Now()
	_, chunks, err := executeNearestShortlistChunks(
		context.Background(), executor, device, columns, narrowed, PrecisionFloat32, 7, exactNearestChunkRows)
	chunkWall := time.Since(chunkStart)
	if err != nil {
		t.Fatalf("direct chunked submission failed: %v", err)
	}
	singleStart := time.Now()
	_, singleChunks, err := executeNearestShortlistChunks(
		context.Background(), executor, device, columns, narrowed, PrecisionFloat32, 7, dataset.Rows)
	singleWall := time.Since(singleStart)
	if err != nil {
		t.Fatalf("single submission failed: %v", err)
	}
	if chunks != 2 || singleChunks != 1 {
		t.Fatalf("got chunk counts %d and %d, want 2 and 1", chunks, singleChunks)
	}
	t.Logf("shape=32k×32 production_chunked_ms=%.3f chunked_submission_ms=%.3f single_submission_ms=%.3f fixed_cost_delta_ms=%.3f",
		float64(productionWall)/float64(time.Millisecond),
		float64(chunkWall)/float64(time.Millisecond),
		float64(singleWall)/float64(time.Millisecond),
		float64(chunkWall-singleWall)/float64(time.Millisecond))
}

func requireHardwareGPU(t *testing.T) {
	t.Helper()
	requireGPU(t)
	info, err := wgpu.Probe()
	if err != nil {
		t.Skipf("no usable GPU backend: %v", err)
	}
	if info.Name == "" {
		t.Skip("GPU backend returned no hardware adapter information")
	}
}
