package accel

import (
	"context"
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

// failingExecutor reaches the device and then does not deliver.
type failingExecutor struct {
	calls int
	err   error
}

func (*failingExecutor) Name() string { return "failing" }

func (e *failingExecutor) Execute(context.Context, ExecuteRequest) (ExecuteResponse, error) {
	e.calls++
	return ExecuteResponse{}, e.err
}

// TestDeviceFailureStillAnswers covers the device that exists but does not
// deliver. The reason must still name the failure — falling back is not the same
// as pretending it worked.
func TestDeviceFailureStillAnswers(t *testing.T) {
	exerciseDeviceRegardlessOfProfit(t)
	session := singleDeviceSession(t, Config{})
	failing := &failingExecutor{err: errors.New("device fell over")}
	if err := RegisterBackendExecutor(BackendCUDA, failing); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	rnd := rand.New(rand.NewSource(12))
	ds := exactDataset(1024, 3, rnd)
	queries := exactQueries(40, 3, rnd)

	result, err := session.ExecuteNearestExact(ds, queries, 2, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("a device failure should not surface as an error outside strict mode: %v", err)
	}
	if failing.calls == 0 {
		t.Fatal("the executor was never reached, so this is not testing a device failure")
	}
	if result.Accelerated {
		t.Fatal("a failed execution is not an accelerated one")
	}
	if result.FallbackReason != FallbackReasonExecutionFailed {
		t.Fatalf("expected execution-failed, got %q", result.FallbackReason)
	}
	assertMatchesReference(t, ds, queries, 2, result.Index, result.Distance)
	if result.Transfer != 0 || result.Dispatch != 0 || result.Readback != 0 || result.BytesUploaded != 0 {
		t.Fatal("a failed device run must not report device cost")
	}
}

// TestIneligibleDatasetReturnsNothing guards the other half of the rule: a
// dataset no kernel and no reference can read produces no answer, and says why.
func TestIneligibleDatasetReturnsNothing(t *testing.T) {
	session := noDeviceSession(t, Config{})
	const rows = 64
	values := make([]string, rows)
	for i := range values {
		values[i] = "x"
	}
	ds := &Dataset{Name: "strings", Lineage: "test", Rows: rows, Buffers: []Buffer{
		{Name: "s", Type: DataTypeString, Values: values, Len: rows},
	}}

	// Outside strict mode nothing errors — the reason carries the explanation,
	// the same way every other path in the package reports one.
	result, err := session.ExecuteNearestExact(ds, [][]float64{{1}, {2}}, 1, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute should not error outside strict mode: %v", err)
	}
	if result.FallbackReason != FallbackReasonDTypeNotEligible {
		t.Fatalf("expected dtype-not-eligible, got %q", result.FallbackReason)
	}
	if result.Index != nil || result.Distance != nil {
		t.Fatal("no answer should be produced for a dataset nothing can read")
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
	ds := exactDataset(1024, 3, rnd)
	queries := exactQueries(40, 3, rnd)

	if _, err := session.ExecuteNearestExact(ds, queries, 2, WorkloadEstimate{}); err == nil {
		t.Fatal("strict GPU mode should report that it could not run on a device")
	}
}
