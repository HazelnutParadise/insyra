package accel

import "testing"

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
