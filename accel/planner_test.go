package accel

import "testing"

func TestPlanShardableAggregatesAllAccelerators(t *testing.T) {
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
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
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
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

func TestPlanShardableBuildsWeightedAssignments(t *testing.T) {
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	RegisterDiscoverer(stubDiscoverer{
		name: "weighted-devices",
		devices: []Device{
			{
				ID:          "cuda:0",
				Backend:     BackendCUDA,
				Type:        DeviceTypeDiscrete,
				MemoryClass: MemoryClassDevice,
				BudgetBytes: 16 * 1024 * 1024 * 1024,
				Score:       120,
			},
			{
				ID:           "webgpu:0",
				Backend:      BackendWebGPU,
				Type:         DeviceTypeIntegrated,
				MemoryClass:  MemoryClassShared,
				SharedMemory: true,
				BudgetBytes:  8 * 1024 * 1024 * 1024,
				Score:        60,
			},
		},
	})

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	plan := session.PlanShardableWorkload(WorkloadEstimate{
		Class: WorkloadClassColumnar,
		Rows:  3000,
		Bytes: 3000000,
	})
	if !plan.Accelerated {
		t.Fatal("expected accelerated plan")
	}
	if len(plan.Assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(plan.Assignments))
	}
	if plan.Assignments[0].Rows <= plan.Assignments[1].Rows {
		t.Fatalf("expected stronger device to receive more rows, got %d <= %d", plan.Assignments[0].Rows, plan.Assignments[1].Rows)
	}
	if plan.Assignments[0].Bytes <= plan.Assignments[1].Bytes {
		t.Fatalf("expected stronger device to receive more bytes, got %d <= %d", plan.Assignments[0].Bytes, plan.Assignments[1].Bytes)
	}
	if plan.MergePolicy != MergePolicyCPU {
		t.Fatalf("expected cpu merge policy for heterogeneous backends, got %q", plan.MergePolicy)
	}
}

func TestPlanShardableWorkloadFallsBackWhenAutoModeNotProfitable(t *testing.T) {
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	plan := session.PlanShardableWorkload(WorkloadEstimate{
		Class: WorkloadClassColumnar,
		Rows:  32,
		Bytes: 1024,
	})
	if plan.Accelerated {
		t.Fatal("expected tiny workload to fall back in auto mode")
	}
	if plan.FallbackReason != FallbackReasonWorkloadNotProfitable {
		t.Fatalf("expected workload-not-profitable fallback, got %q", plan.FallbackReason)
	}
}

func TestPlanShardableWorkloadKeepsStrictModeOnUnsupportedWorkload(t *testing.T) {
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")

	session, err := Open(Config{Mode: ModeStrictGPU})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	plan := session.PlanShardableWorkload(WorkloadEstimate{
		Class: WorkloadClassUnknown,
		Rows:  1000,
		Bytes: 64000,
	})
	if plan.Accelerated {
		t.Fatal("expected unsupported workload to remain non-accelerated")
	}
	if plan.FallbackReason != FallbackReasonWorkloadUnsupported {
		t.Fatalf("expected workload-unsupported fallback, got %q", plan.FallbackReason)
	}
}
