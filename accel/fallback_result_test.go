package accel

import (
	"errors"
	"math/rand"
	"testing"
)

// noDeviceSession opens a session on a host where nothing is discoverable — the
// situation of anyone running Insyra without a GPU.
func noDeviceSession(t *testing.T, cfg Config) *Session {
	t.Helper()
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	isolateBuiltinProbes(t)
	resetBackendExecutorsForTest()
	t.Cleanup(resetBackendExecutorsForTest)

	session, err := Open(cfg)
	if err != nil {
		t.Fatalf("open should succeed with no devices: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if len(session.Devices()) != 0 {
		t.Fatalf("expected no devices, got %d", len(session.Devices()))
	}
	return session
}

// TestDistanceOpsAnswerWithoutADevice is the whole point of the fallback: a host
// with no GPU gets the numbers, not an empty slice.
func TestDistanceOpsAnswerWithoutADevice(t *testing.T) {
	session := noDeviceSession(t, Config{})
	rnd := rand.New(rand.NewSource(11))
	ds := distanceDataset(1024, 3, rnd)
	queries := randomQueries(4, 3, rnd)

	wantDist, rows, err := SquaredDistancesCPU(ds, queries)
	if err != nil {
		t.Fatalf("cpu reference failed: %v", err)
	}

	dist, err := session.ExecuteDistances(ds, queries, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("distances should not error outside strict mode: %v", err)
	}
	if dist.Accelerated {
		t.Fatal("nothing was discovered, so nothing can have been accelerated")
	}
	if dist.FallbackReason != FallbackReasonNoAccelerator {
		t.Fatalf("expected no-accelerator, got %q", dist.FallbackReason)
	}
	if len(dist.Distances) != len(wantDist) {
		t.Fatalf("expected %d distances, got %d", len(wantDist), len(dist.Distances))
	}
	for i, want := range wantDist {
		if dist.Distances[i] != want {
			t.Fatalf("distance %d: got %v, want %v", i, dist.Distances[i], want)
		}
	}
	if dist.Rows != rows {
		t.Fatalf("expected %d rows, got %d", rows, dist.Rows)
	}

	wantIdx, wantNear, _, err := NearestQueryCPU(ds, queries)
	if err != nil {
		t.Fatalf("cpu reference failed: %v", err)
	}
	near, err := session.ExecuteNearestQuery(ds, queries, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("nearest query should not error outside strict mode: %v", err)
	}
	if near.Accelerated || near.FallbackReason != FallbackReasonNoAccelerator {
		t.Fatalf("expected an unaccelerated no-accelerator result, got %v/%q", near.Accelerated, near.FallbackReason)
	}
	if len(near.Index) != len(wantIdx) {
		t.Fatalf("expected %d indices, got %d", len(wantIdx), len(near.Index))
	}
	for i := range wantIdx {
		if near.Index[i] != wantIdx[i] || near.Distance[i] != wantNear[i] {
			t.Fatalf("row %d: got (%d, %v), want (%d, %v)",
				i, near.Index[i], near.Distance[i], wantIdx[i], wantNear[i])
		}
	}
}

// TestDeviceFailureStillAnswers covers the device that exists but does not
// deliver. The reason must still name the failure — falling back is not the same
// as pretending it worked.
func TestDeviceFailureStillAnswers(t *testing.T) {
	session := singleDeviceSession(t, Config{})
	failing := &fakeExecutor{name: "failing", err: errors.New("device fell over")}
	if err := RegisterBackendExecutor(BackendCUDA, failing); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	rnd := rand.New(rand.NewSource(12))
	ds := distanceDataset(1024, 3, rnd)
	queries := randomQueries(4, 3, rnd)
	wantDist, _, err := SquaredDistancesCPU(ds, queries)
	if err != nil {
		t.Fatalf("cpu reference failed: %v", err)
	}

	dist, err := session.ExecuteDistances(ds, queries, WorkloadEstimate{Precision: PrecisionFloat32})
	if err != nil {
		t.Fatalf("a device failure should not surface as an error outside strict mode: %v", err)
	}
	if failing.calls == 0 {
		t.Fatal("the executor was never reached, so this is not testing a device failure")
	}
	if dist.Accelerated {
		t.Fatal("a failed execution is not an accelerated one")
	}
	if dist.FallbackReason != FallbackReasonExecutionFailed {
		t.Fatalf("expected execution-failed, got %q", dist.FallbackReason)
	}
	if len(dist.Distances) != len(wantDist) {
		t.Fatalf("expected the CPU answer after the device failed, got %d distances", len(dist.Distances))
	}
	for i, want := range wantDist {
		if dist.Distances[i] != want {
			t.Fatalf("distance %d: got %v, want %v", i, dist.Distances[i], want)
		}
	}
	if dist.Transfer != 0 || dist.Dispatch != 0 || dist.Readback != 0 || dist.BytesUploaded != 0 {
		t.Fatal("a failed device run must not report device cost")
	}
}

// TestIneligibleRequestStillReturnsNothing guards the other half of the rule:
// the CPU must not deliver, by a side door, what the caller refused.
func TestIneligibleRequestStillReturnsNothing(t *testing.T) {
	session := noDeviceSession(t, Config{})
	rnd := rand.New(rand.NewSource(13))
	ds := distanceDataset(1024, 3, rnd)
	queries := randomQueries(2, 3, rnd)

	dist, err := session.ExecuteDistances(ds, queries, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute should not error outside strict mode: %v", err)
	}
	if dist.FallbackReason != FallbackReasonPrecisionNotAccepted {
		t.Fatalf("expected precision-not-accepted, got %q", dist.FallbackReason)
	}
	if dist.Distances != nil {
		t.Fatal("a caller who refused reduced precision must not receive a narrowed answer anyway")
	}

	near, err := session.ExecuteNearestQuery(ds, queries, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute should not error outside strict mode: %v", err)
	}
	if near.Index != nil || near.Distance != nil {
		t.Fatal("a caller who refused reduced precision must not receive a narrowed answer anyway")
	}
}

// TestStrictGPUStillFails pins the opt-out. Strict mode exists so that a missing
// device is loud, and the fallback must not quietly satisfy it.
func TestStrictGPUStillFails(t *testing.T) {
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	isolateBuiltinProbes(t)
	resetBackendExecutorsForTest()
	t.Cleanup(resetBackendExecutorsForTest)

	// Open itself reports the missing accelerator in strict mode, and hands back
	// a session anyway so the reason stays inspectable.
	session, openErr := Open(Config{Mode: ModeStrictGPU, Strict: true})
	if openErr == nil {
		t.Fatal("strict GPU mode should refuse to open with no accelerator")
	}
	if session == nil {
		t.Fatal("a strict-mode failure should still return an inspectable session")
	}
	t.Cleanup(func() { _ = session.Close() })

	rnd := rand.New(rand.NewSource(14))
	ds := distanceDataset(1024, 3, rnd)
	queries := randomQueries(2, 3, rnd)

	dist, err := session.ExecuteDistances(ds, queries, WorkloadEstimate{Precision: PrecisionFloat32})
	if err == nil {
		t.Fatal("strict GPU mode should report that it could not run on a device")
	}
	if dist.Distances != nil {
		t.Fatal("strict GPU mode should not hand back a CPU result")
	}

	near, err := session.ExecuteNearestQuery(ds, queries, WorkloadEstimate{Precision: PrecisionFloat32})
	if err == nil {
		t.Fatal("strict GPU mode should report that it could not run on a device")
	}
	if near.Index != nil {
		t.Fatal("strict GPU mode should not hand back a CPU result")
	}
}
