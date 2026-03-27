package accel

import "testing"

func TestPlanShardableAggregatesAllAccelerators(t *testing.T) {
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	plan := session.PlanShardable()
	if !plan.Accelerated {
		t.Fatal("expected shardable plan to be accelerated")
	}
	if len(plan.DeviceIDs) != 2 {
		t.Fatalf("expected 2 planned devices, got %d", len(plan.DeviceIDs))
	}
	if !plan.Heterogeneous {
		t.Fatal("expected heterogeneous plan when multiple backends participate")
	}
	if plan.TotalBudgetBytes == 0 {
		t.Fatal("expected positive aggregated budget")
	}
	if plan.DeviceIDs[0] != "cuda:stub:0" {
		t.Fatalf("expected highest-scoring device first, got %q", plan.DeviceIDs[0])
	}
}

func TestPlanShardableFallsBackWithoutAccelerators(t *testing.T) {
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	plan := session.PlanShardable()
	if plan.Accelerated {
		t.Fatal("expected non-accelerated shardable plan without devices")
	}
	if plan.FallbackReason != FallbackReasonNoAccelerator {
		t.Fatalf("expected no-accelerator fallback, got %q", plan.FallbackReason)
	}
}
