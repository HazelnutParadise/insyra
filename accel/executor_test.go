package accel

import "testing"

type testBackendAllocator struct {
	name string
}

func (a testBackendAllocator) Name() string { return a.name }

func (a testBackendAllocator) Materialize(dataset *Dataset, plan ShardPlan) AllocationRecord {
	record := AllocationRecord{
		DeviceIDs:         append([]string(nil), plan.DeviceIDs...),
		DeviceResidentMap: map[string]map[string]uint64{},
	}
	for _, buffer := range dataset.Buffers {
		bytes := estimateBufferResidentBytes(buffer)
		record.DeviceResidentMap[buffer.Name] = map[string]uint64{
			plan.DeviceIDs[0]: bytes,
		}
		record.BytesMoved += bytes
	}
	return record
}

func TestExecuteProjectedDatasetMaterializesDeviceResidency(t *testing.T) {
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

	dataset := &Dataset{
		Name:        "numbers",
		Fingerprint: "numbers-fp",
		Lineage:     "project:datalist",
		Rows:        1024,
		Buffers: []Buffer{
			{
				Name:   "numbers",
				Type:   DataTypeInt64,
				Values: []int64{1, 2, 3, 4},
				Len:    1024,
			},
		},
	}

	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute projected dataset failed: %v", err)
	}
	if !result.Accelerated {
		t.Fatal("expected accelerated execution result")
	}
	if len(result.Assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(result.Assignments))
	}

	snapshot := session.CacheSnapshot()
	if len(snapshot.Entries) != 1 {
		t.Fatalf("expected 1 cache entry, got %d", len(snapshot.Entries))
	}
	if len(snapshot.Entries[0].DeviceIDs) != 2 {
		t.Fatalf("expected executed cache entry to record 2 devices, got %d", len(snapshot.Entries[0].DeviceIDs))
	}
	if len(snapshot.Entries[0].DeviceResidentBytes) != 2 {
		t.Fatalf("expected per-device residency bytes, got %d", len(snapshot.Entries[0].DeviceResidentBytes))
	}
	if snapshot.DeviceUsage[0].ResidentBytes == 0 && snapshot.DeviceUsage[1].ResidentBytes == 0 {
		t.Fatal("expected per-device resident bytes after execution")
	}
	report := session.Report()
	if report.Metrics["execution.device_participants"] != 2 {
		t.Fatalf("expected 2 execution device participants, got %v", report.Metrics["execution.device_participants"])
	}
	if report.Metrics["execution.bytes_moved"] <= 0 {
		t.Fatalf("expected bytes moved metric, got %v", report.Metrics["execution.bytes_moved"])
	}
}

func TestExecuteProjectedDatasetUsesRegisteredAllocatorForSingleBackendPlan(t *testing.T) {
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	resetBackendAllocatorsForTest()
	t.Cleanup(resetBackendAllocatorsForTest)

	if err := RegisterBackendAllocator(BackendCUDA, testBackendAllocator{name: "cuda-test"}); err != nil {
		t.Fatalf("register backend allocator failed: %v", err)
	}

	RegisterDiscoverer(stubDiscoverer{
		name: "cuda-pair",
		devices: []Device{
			{ID: "cuda:0", Backend: BackendCUDA, Type: DeviceTypeDiscrete, MemoryClass: MemoryClassDevice, BudgetBytes: 1 << 20, Score: 100},
			{ID: "cuda:1", Backend: BackendCUDA, Type: DeviceTypeDiscrete, MemoryClass: MemoryClassDevice, BudgetBytes: 1 << 20, Score: 90},
		},
	})

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	dataset := &Dataset{
		Name:        "numbers",
		Fingerprint: "numbers-fp",
		Lineage:     "project:datalist",
		Rows:        1024,
		Buffers:     []Buffer{{Name: "numbers", Type: DataTypeInt64, Values: []int64{1, 2, 3, 4}, Len: 1024}},
	}

	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute projected dataset failed: %v", err)
	}
	if result.Allocator != "cuda-test" {
		t.Fatalf("expected registered allocator, got %q", result.Allocator)
	}
	if result.AllocatorKind != AllocatorKindRegistered {
		t.Fatalf("expected registered allocator kind, got %q", result.AllocatorKind)
	}
	if session.Report().Metrics["execution.allocator_registered"] != 1 {
		t.Fatalf("expected registered allocator metric, got %v", session.Report().Metrics["execution.allocator_registered"])
	}
}

func TestExecuteProjectedDatasetFallsBackToLedgerForHeterogeneousPlan(t *testing.T) {
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	resetBackendAllocatorsForTest()
	t.Cleanup(resetBackendAllocatorsForTest)

	if err := RegisterBackendAllocator(BackendCUDA, testBackendAllocator{name: "cuda-test"}); err != nil {
		t.Fatalf("register backend allocator failed: %v", err)
	}

	RegisterDiscoverer(stubDiscoverer{
		name: "mixed-devices",
		devices: []Device{
			{ID: "cuda:0", Backend: BackendCUDA, Type: DeviceTypeDiscrete, MemoryClass: MemoryClassDevice, BudgetBytes: 1 << 20, Score: 100},
			{ID: "webgpu:0", Backend: BackendWebGPU, Type: DeviceTypeIntegrated, MemoryClass: MemoryClassShared, SharedMemory: true, BudgetBytes: 1 << 20, Score: 80},
		},
	})

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	dataset := &Dataset{
		Name:        "numbers",
		Fingerprint: "numbers-fp",
		Lineage:     "project:datalist",
		Rows:        1024,
		Buffers:     []Buffer{{Name: "numbers", Type: DataTypeInt64, Values: []int64{1, 2, 3, 4}, Len: 1024}},
	}

	result, err := session.ExecuteProjectedDataset(dataset, WorkloadEstimate{})
	if err != nil {
		t.Fatalf("execute projected dataset failed: %v", err)
	}
	if result.AllocatorKind != AllocatorKindLedger {
		t.Fatalf("expected ledger allocator kind, got %q", result.AllocatorKind)
	}
	if session.Report().Metrics["execution.allocator_ledger"] != 1 {
		t.Fatalf("expected ledger allocator metric, got %v", session.Report().Metrics["execution.allocator_ledger"])
	}
}

func TestExecuteProjectedDatasetStrictUnsupportedReturnsError(t *testing.T) {
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

	dataset := &Dataset{
		Name:        "unsupported",
		Fingerprint: "unsupported-fp",
		Lineage:     "project:datalist",
		Rows:        256,
		Buffers: []Buffer{
			{Name: "unsupported", Type: DataTypeAny, Values: []any{1, "x"}, Len: 256},
		},
	}

	_, err = session.ExecuteProjectedDataset(dataset, WorkloadEstimate{Class: WorkloadClassUnknown})
	if err == nil {
		t.Fatal("expected strict mode unsupported execution to return error")
	}
	if session.Report().Metrics["execution.fallback"] != 1 {
		t.Fatalf("expected execution fallback metric, got %v", session.Report().Metrics["execution.fallback"])
	}
}
