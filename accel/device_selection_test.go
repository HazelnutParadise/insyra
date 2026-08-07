package accel

import (
	"strings"
	"testing"
)

func selectionTestDevices() []Device {
	return []Device{
		{ID: "cuda:test:0", Name: "Selection Test 0", Backend: BackendCUDA, Type: DeviceTypeDiscrete, MemoryClass: MemoryClassDevice, BudgetBytes: 1024, Score: 100},
		{ID: "cuda:test:1", Name: "Selection Test 1", Backend: BackendCUDA, Type: DeviceTypeDiscrete, MemoryClass: MemoryClassDevice, BudgetBytes: 1024, Score: 90},
		{ID: "cuda:test:2", Name: "Selection Test 2", Backend: BackendCUDA, Type: DeviceTypeDiscrete, MemoryClass: MemoryClassDevice, BudgetBytes: 1024, Score: 80},
	}
}

func openSelectionTestSession(t *testing.T, cfg Config) (*Session, error) {
	t.Helper()
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	isolateBuiltinProbes(t)
	setBuiltinProbeOverrideForTest(
		BackendCUDA,
		func(Config) ([]Device, error) { return nil, ErrNativeProbeUnavailable },
		func(Config) ([]Device, error) { return selectionTestDevices(), nil },
	)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")

	session, err := Open(cfg)
	if err != nil && session == nil {
		t.Fatalf("open returned no session: %v", err)
	}
	if session != nil {
		t.Cleanup(func() { _ = session.Close() })
	}
	return session, err
}

func TestDeviceSelectionEnvironmentMaskHidesDevicesAtDiscoveryBoundary(t *testing.T) {
	t.Setenv(accelDevicesEnv, "cuda:test:1")
	session, err := openSelectionTestSession(t, Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	if got := session.Devices(); len(got) != 1 || got[0].ID != "cuda:test:1" {
		t.Fatalf("eligible devices = %+v, want only cuda:test:1", got)
	}
	report := session.Report()
	if len(report.DiscoveredDeviceIDs) != 1 || report.DiscoveredDeviceIDs[0] != "cuda:test:1" {
		t.Fatalf("report discovered IDs = %v, want only cuda:test:1", report.DiscoveredDeviceIDs)
	}
	plan := session.PlanShardableWorkload(WorkloadEstimate{Class: WorkloadClassColumnar, Rows: 1000})
	if len(plan.DeviceIDs) != 1 || plan.DeviceIDs[0] != "cuda:test:1" {
		t.Fatalf("planner devices = %v, want only cuda:test:1", plan.DeviceIDs)
	}
}

func TestDeviceSelectionConfigAllowlistWinsWithinEnvironmentAndPreference(t *testing.T) {
	t.Setenv(accelDevicesEnv, "0,1,2")
	session, err := openSelectionTestSession(t, Config{
		Devices:          []string{"cuda:test:2"},
		PreferredDevices: []string{"cuda:test:0"},
	})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	devices := session.Devices()
	if len(devices) != 1 || devices[0].ID != "cuda:test:2" {
		t.Fatalf("eligible devices = %+v, want only cuda:test:2", devices)
	}
	if got := session.Report().SelectedDeviceIDs; len(got) != 1 || got[0] != "cuda:test:2" {
		t.Fatalf("selected device IDs = %v, want only cuda:test:2", got)
	}
}

func TestDeviceSelectionStrictEmptyNamesEnvironmentBound(t *testing.T) {
	t.Setenv(accelDevicesEnv, "missing-device")
	strictSession, err := openSelectionTestSession(t, Config{Mode: ModeStrictGPU})
	if err == nil || !strings.Contains(err.Error(), accelDevicesEnv) {
		t.Fatalf("strict open error = %v, want one naming %s", err, accelDevicesEnv)
	}
	if strictSession == nil {
		t.Fatal("expected strict failure to return a session")
	}
	t.Cleanup(func() { _ = strictSession.Close() })
	if got := strictSession.Report().FallbackReason; got != FallbackReasonDeviceSelectionEmpty {
		t.Fatalf("fallback reason = %q, want %q", got, FallbackReasonDeviceSelectionEmpty)
	}
}

func TestDeviceSelectionAutomaticEmptyFallsBackWithDedicatedReason(t *testing.T) {
	t.Setenv(accelDevicesEnv, "missing-device")
	session, err := openSelectionTestSession(t, Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	result, err := session.ExecuteNearestExact(
		&Dataset{Rows: 1, Buffers: []Buffer{{Name: "x", Type: DataTypeFloat64, Values: []float64{0}, Len: 1}}},
		[][]float64{{0}}, 1, WorkloadEstimate{},
	)
	if err != nil {
		t.Fatalf("CPU fallback failed: %v", err)
	}
	if result.Accelerated || result.FallbackReason != FallbackReasonDeviceSelectionEmpty {
		t.Fatalf("result = accelerated:%t reason:%q, want CPU device-selection fallback", result.Accelerated, result.FallbackReason)
	}
	if len(result.Index) != 1 || result.Index[0] != 0 || len(result.Distance) != 1 || result.Distance[0] != 0 {
		t.Fatalf("CPU result = index:%v distance:%v, want [0] and [0]", result.Index, result.Distance)
	}
}

func TestDeviceSelectionReportsUnmatchedEntries(t *testing.T) {
	t.Setenv(accelDevicesEnv, "cuda:test:0,missing-env")
	session, err := openSelectionTestSession(t, Config{Devices: []string{"cuda:test:0", "missing-config"}})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	got := session.Report().UnmatchedDeviceSelectors
	if len(got) != 2 {
		t.Fatalf("unmatched selectors = %+v, want two entries", got)
	}
	if got[0] != (UnmatchedDeviceSelector{Bound: accelDevicesEnv, Selector: "missing-env"}) ||
		got[1] != (UnmatchedDeviceSelector{Bound: "Config.Devices", Selector: "missing-config"}) {
		t.Fatalf("unmatched selectors = %+v, want env and config entries", got)
	}
}

func TestDeviceSelectionIntersectsBoundsUsingOriginalDiscoveryIndices(t *testing.T) {
	t.Setenv(accelDevicesEnv, "0, 2")
	session, err := openSelectionTestSession(t, Config{Devices: []string{"2"}})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	if got := session.Devices(); len(got) != 1 || got[0].ID != "cuda:test:2" {
		t.Fatalf("eligible devices = %+v, want index 2 only", got)
	}

	t.Setenv(accelDevicesEnv, "0")
	other, err := openSelectionTestSession(t, Config{Devices: []string{"1"}})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	if got := other.Devices(); len(got) != 0 {
		t.Fatalf("eligible devices = %+v, want empty intersection", got)
	}
	if got := other.Report().FallbackReason; got != FallbackReasonDeviceSelectionEmpty {
		t.Fatalf("fallback reason = %q, want %q", got, FallbackReasonDeviceSelectionEmpty)
	}
}

func TestDeviceSelectionPrefersExactIDBeforeIndex(t *testing.T) {
	devices := []Device{
		{ID: "0", Name: "numeric ID", Backend: BackendCUDA},
		{ID: "other", Name: "other", Backend: BackendCUDA},
	}
	matched, unmatched := resolveDeviceSelectors([]string{"0"}, devices, "test")
	if len(unmatched) != 0 {
		t.Fatalf("unmatched = %+v, want none", unmatched)
	}
	if _, ok := matched[0]; !ok {
		t.Fatalf("matched = %v, want exact numeric ID at index 0", matched)
	}
}
