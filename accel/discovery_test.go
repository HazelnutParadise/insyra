package accel

import (
	"errors"
	"testing"
	"time"
)

func disableNativeProbes(t *testing.T) {
	t.Helper()
	t.Setenv("INSYRA_ACCEL_DISABLE_NATIVE_PROBES", "1")
}

type stubDiscoverer struct {
	name    string
	devices []Device
	err     error
	delay   time.Duration
}

func (d stubDiscoverer) Name() string { return d.name }

func (d stubDiscoverer) Discover(cfg Config) ([]Device, error) {
	if d.delay > 0 {
		time.Sleep(d.delay)
	}
	return d.devices, d.err
}

func TestOpenDiscoversDevicesFromRegisteredDiscoverers(t *testing.T) {
	disableNativeProbes(t)
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	RegisterDiscoverer(stubDiscoverer{
		name: "stub-webgpu",
		devices: []Device{
			{
				ID:          "webgpu:0",
				Name:        "Stub GPU",
				Backend:     BackendWebGPU,
				Type:        DeviceTypeIntegrated,
				MemoryClass: MemoryClassShared,
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

	if len(session.Devices()) != 1 {
		t.Fatalf("expected one discovered device, got %d", len(session.Devices()))
	}
	report := session.Report()
	if !report.Accelerated {
		t.Fatal("expected report to indicate an acceleration backend is available")
	}
	if report.SelectedBackend != BackendWebGPU {
		t.Fatalf("expected selected backend %q, got %q", BackendWebGPU, report.SelectedBackend)
	}
}

func TestOpenSkipsDiscoveryInCPUMode(t *testing.T) {
	disableNativeProbes(t)
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	RegisterDiscoverer(stubDiscoverer{
		name: "stub-cuda",
		devices: []Device{
			{ID: "cuda:0", Backend: BackendCUDA, Type: DeviceTypeDiscrete, MemoryClass: MemoryClassDevice},
		},
	})

	session, err := Open(Config{Mode: ModeCPU})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	if len(session.Devices()) != 0 {
		t.Fatalf("expected cpu mode to skip discovery, got %d devices", len(session.Devices()))
	}
	if session.Report().FallbackReason != FallbackReasonCPUOnly {
		t.Fatalf("expected cpu-only fallback reason, got %q", session.Report().FallbackReason)
	}
}

func TestCurrentDiscoverersIncludesBuiltinBackends(t *testing.T) {
	disableNativeProbes(t)
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	discoverers := currentDiscoverers()
	if len(discoverers) < 3 {
		t.Fatalf("expected at least 3 discoverers, got %d", len(discoverers))
	}

	names := []string{
		discoverers[0].Name(),
		discoverers[1].Name(),
		discoverers[2].Name(),
	}
	expected := []string{"cuda", "metal", "webgpu"}
	for idx, want := range expected {
		if names[idx] != want {
			t.Fatalf("expected discoverer %d to be %q, got %q", idx, want, names[idx])
		}
	}
}

func TestCurrentDiscoverersAppendsRegisteredDiscoverersAfterBuiltins(t *testing.T) {
	disableNativeProbes(t)
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	RegisterDiscoverer(stubDiscoverer{name: "custom"})

	discoverers := currentDiscoverers()
	if got := discoverers[len(discoverers)-1].Name(); got != "custom" {
		t.Fatalf("expected custom discoverer to be appended last, got %q", got)
	}
}

func TestOpenDiscoversBuiltinStubDeviceFromEnv(t *testing.T) {
	disableNativeProbes(t)
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	resetBuiltinProbeOverridesForTest()
	t.Cleanup(resetBuiltinProbeOverridesForTest)
	setBuiltinProbeOverrideForTest(
		BackendWebGPU,
		func(cfg Config) ([]Device, error) { return nil, ErrNativeProbeUnavailable },
		nil,
	)
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	devices := session.Devices()
	if len(devices) != 1 {
		t.Fatalf("expected one builtin stub device, got %d", len(devices))
	}
	device := devices[0]
	if device.Backend != BackendWebGPU {
		t.Fatalf("expected webgpu backend, got %q", device.Backend)
	}
	if device.ID != "webgpu:stub:0" {
		t.Fatalf("expected deterministic stub device id, got %q", device.ID)
	}
	if device.ProbeSource != ProbeSourceEnvStub {
		t.Fatalf("expected env-stub probe source, got %q", device.ProbeSource)
	}
	if !device.CapabilitySummary["encoded_strings"] {
		t.Fatal("expected normalized encoded_strings capability")
	}
	if !device.CapabilitySummary["validity_bitmap"] {
		t.Fatal("expected normalized validity_bitmap capability")
	}
	if !device.CapabilitySummary["shared_memory"] {
		t.Fatal("expected normalized shared_memory capability")
	}
	if !session.Report().Accelerated {
		t.Fatal("expected builtin stub device to mark session accelerated")
	}
}

func TestBuiltinDiscovererUsesNativeProbeOverrideBeforeStub(t *testing.T) {
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	resetBuiltinProbeOverridesForTest()
	t.Cleanup(resetBuiltinProbeOverridesForTest)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")

	setBuiltinProbeOverrideForTest(
		BackendCUDA,
		func(cfg Config) ([]Device, error) {
			return []Device{
				{
					ID:          "cuda:native:0",
					Name:        "Native CUDA",
					Backend:     BackendCUDA,
					Vendor:      "nvidia",
					Type:        DeviceTypeDiscrete,
					MemoryClass: MemoryClassDevice,
					BudgetBytes: 1024,
				},
			}, nil
		},
		nil,
	)
	setBuiltinProbeOverrideForTest(
		BackendWebGPU,
		func(cfg Config) ([]Device, error) { return nil, ErrNativeProbeUnavailable },
		nil,
	)

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	devices := session.Devices()
	if len(devices) != 1 {
		t.Fatalf("expected one native device, got %d", len(devices))
	}
	if devices[0].ID != "cuda:native:0" {
		t.Fatalf("expected native cuda device, got %q", devices[0].ID)
	}
	if devices[0].ProbeSource != ProbeSourceNative {
		t.Fatalf("expected native probe source, got %q", devices[0].ProbeSource)
	}
	if !devices[0].CapabilitySummary["native_probe"] {
		t.Fatal("expected native_probe capability flag")
	}
	if devices[0].CapabilitySummary["env_stub"] {
		t.Fatal("did not expect env_stub capability flag for native probe")
	}
}

func TestOpenStrictGPUFailsWithoutAcceleratorButReturnsSessionReport(t *testing.T) {
	disableNativeProbes(t)
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	session, err := Open(Config{Mode: ModeStrictGPU})
	if err == nil {
		t.Fatal("expected strict-gpu mode to fail without accelerators")
	}
	if session == nil {
		t.Fatal("expected strict-gpu failure to still return a session")
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	report := session.Report()
	if report.Mode != ModeStrictGPU {
		t.Fatalf("expected strict-gpu report mode, got %q", report.Mode)
	}
	if report.FallbackReason != FallbackReasonStrictGPUUnavailable {
		t.Fatalf("expected strict-gpu unavailable fallback reason, got %q", report.FallbackReason)
	}
	if report.Metrics["devices.discovered"] != 0 {
		t.Fatalf("expected zero discovered devices, got %v", report.Metrics["devices.discovered"])
	}
}

func TestDiscoveryReportPopulatesCoreMetrics(t *testing.T) {
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	resetBuiltinProbeOverridesForTest()
	t.Cleanup(resetBuiltinProbeOverridesForTest)
	setBuiltinProbeOverrideForTest(
		BackendCUDA,
		func(cfg Config) ([]Device, error) {
			return []Device{
				{
					ID:          "cuda:native:0",
					Name:        "Native CUDA",
					Backend:     BackendCUDA,
					Vendor:      "nvidia",
					Type:        DeviceTypeDiscrete,
					MemoryClass: MemoryClassDevice,
					BudgetBytes: 1024,
				},
			}, nil
		},
		nil,
	)
	setBuiltinProbeOverrideForTest(
		BackendWebGPU,
		func(cfg Config) ([]Device, error) { return nil, ErrNativeProbeUnavailable },
		nil,
	)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")
	t.Setenv("INSYRA_ACCEL_STUB_WEBGPU", "1")

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	report := session.Report()
	if report.Metrics["devices.discovered"] != 2 {
		t.Fatalf("expected 2 discovered devices, got %v", report.Metrics["devices.discovered"])
	}
	if report.Metrics["devices.selected"] != 1 {
		t.Fatalf("expected 1 selected device, got %v", report.Metrics["devices.selected"])
	}
	if report.Metrics["devices.native"] != 1 {
		t.Fatalf("expected 1 native device, got %v", report.Metrics["devices.native"])
	}
	if report.Metrics["devices.env_stub"] != 1 {
		t.Fatalf("expected 1 env-stub device, got %v", report.Metrics["devices.env_stub"])
	}
	if report.Metrics["devices.device_local"] != 1 {
		t.Fatalf("expected 1 device-local device, got %v", report.Metrics["devices.device_local"])
	}
	if report.Metrics["devices.shared_memory"] != 1 {
		t.Fatalf("expected 1 shared-memory device, got %v", report.Metrics["devices.shared_memory"])
	}
	if report.Metrics["fallback.occurred"] != 0 {
		t.Fatalf("expected no fallback, got %v", report.Metrics["fallback.occurred"])
	}
	if report.Metrics["memory.budget_bytes_total"] <= 0 {
		t.Fatalf("expected positive budget total, got %v", report.Metrics["memory.budget_bytes_total"])
	}
}

func TestNormalizeBudgetBytesFallsBackForSharedMemoryDevicesWithoutNativeBudget(t *testing.T) {
	restore := setHostMemoryBytesForTest(32 * 1024 * 1024 * 1024)
	defer restore()

	device := Device{
		ID:           "webgpu:native:0",
		Backend:      BackendWebGPU,
		Type:         DeviceTypeIntegrated,
		MemoryClass:  MemoryClassShared,
		SharedMemory: true,
	}

	normalized := normalizeDiscoveredDevice(device, DefaultConfig())
	if normalized.BudgetBytes == 0 {
		t.Fatal("expected non-zero normalized budget for shared-memory device without native budget")
	}
}

func TestOpenReportsDiscoveryErrorReasonCode(t *testing.T) {
	disableNativeProbes(t)
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	RegisterDiscoverer(stubDiscoverer{
		name: "broken-backend",
		err:  errors.New("probe failed"),
	})

	session, err := Open(Config{})
	if err == nil {
		t.Fatal("expected discovery error")
	}
	if session == nil {
		t.Fatal("expected session to be returned on discovery error")
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	report := session.Report()
	if report.FallbackReason != FallbackReasonDiscoveryError {
		t.Fatalf("expected discovery-error fallback reason, got %q", report.FallbackReason)
	}
	if report.Metrics["fallback.occurred"] != 1 {
		t.Fatalf("expected fallback metric to indicate fallback, got %v", report.Metrics["fallback.occurred"])
	}
	if report.Metrics["discovery.errors"] != 1 {
		t.Fatalf("expected one discovery error, got %v", report.Metrics["discovery.errors"])
	}
}

func TestOpenHonorsDiscoveryTimeout(t *testing.T) {
	disableNativeProbes(t)
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	RegisterDiscoverer(stubDiscoverer{
		name:  "slow-backend",
		delay: 200 * time.Millisecond,
	})

	start := time.Now()
	session, err := Open(Config{DiscoveryTimeout: 20 * time.Millisecond})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout discovery error")
	}
	if elapsed >= 150*time.Millisecond {
		t.Fatalf("expected timeout to stop discovery early, took %v", elapsed)
	}
	if session == nil {
		t.Fatal("expected session even when discovery times out")
	}
	t.Cleanup(func() {
		_ = session.Close()
	})

	report := session.Report()
	if report.FallbackReason != FallbackReasonDiscoveryError {
		t.Fatalf("expected discovery-error fallback, got %q", report.FallbackReason)
	}
	if report.Metrics["discovery.errors"] != 1 {
		t.Fatalf("expected one discovery error after timeout, got %v", report.Metrics["discovery.errors"])
	}
}

func TestParseNvidiaSMIOutputBuildsNativeCUDADevices(t *testing.T) {
	raw := []byte("NVIDIA GeForce RTX 4090,24564\nNVIDIA RTX A2000,12288\n")

	devices, err := parseNvidiaSMIOutput(raw)
	if err != nil {
		t.Fatalf("parse nvidia-smi output failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].ID != "cuda:native:0" {
		t.Fatalf("expected first native cuda id, got %q", devices[0].ID)
	}
	if devices[0].ProbeSource != ProbeSourceNative {
		t.Fatalf("expected native probe source, got %q", devices[0].ProbeSource)
	}
	if devices[0].BudgetBytes != 24564*1024*1024 {
		t.Fatalf("unexpected first device budget bytes: %d", devices[0].BudgetBytes)
	}
}

func TestParseWindowsVideoControllersJSONBuildsWebGPUDevices(t *testing.T) {
	raw := `[{"Name":"Intel(R) Iris(R) Xe Graphics","AdapterCompatibility":"Intel Corporation","AdapterRAM":1073741824},{"Name":"AMD Radeon(TM) Graphics","AdapterCompatibility":"Advanced Micro Devices, Inc.","AdapterRAM":2147483648}]`

	devices, err := parseWindowsVideoControllersJSON(raw)
	if err != nil {
		t.Fatalf("parse windows video controller json failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 webgpu-eligible devices, got %d", len(devices))
	}
	if devices[0].Backend != BackendWebGPU || devices[1].Backend != BackendWebGPU {
		t.Fatal("expected webgpu backend classification for windows non-nvidia controllers")
	}
	if devices[0].ProbeSource != ProbeSourceNative || devices[1].ProbeSource != ProbeSourceNative {
		t.Fatal("expected native probe source for windows controller parsing")
	}
	if !devices[0].SharedMemory || !devices[1].SharedMemory {
		t.Fatal("expected windows intel/amd graphics to normalize as shared-memory devices")
	}
}
