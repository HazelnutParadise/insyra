package accel

import (
	"errors"
	"testing"
)

type stubDiscoverer struct {
	name    string
	devices []Device
	err     error
}

func (d stubDiscoverer) Name() string { return d.name }

func (d stubDiscoverer) Discover(cfg Config) ([]Device, error) {
	return d.devices, d.err
}

func TestOpenDiscoversDevicesFromRegisteredDiscoverers(t *testing.T) {
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
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)

	RegisterDiscoverer(stubDiscoverer{name: "custom"})

	discoverers := currentDiscoverers()
	if got := discoverers[len(discoverers)-1].Name(); got != "custom" {
		t.Fatalf("expected custom discoverer to be appended last, got %q", got)
	}
}

func TestOpenDiscoversBuiltinStubDeviceFromEnv(t *testing.T) {
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
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
	if !session.Report().Accelerated {
		t.Fatal("expected builtin stub device to mark session accelerated")
	}
}

func TestOpenStrictGPUFailsWithoutAcceleratorButReturnsSessionReport(t *testing.T) {
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
	if report.Metrics["fallback.occurred"] != 0 {
		t.Fatalf("expected no fallback, got %v", report.Metrics["fallback.occurred"])
	}
	if report.Metrics["memory.budget_bytes_total"] <= 0 {
		t.Fatalf("expected positive budget total, got %v", report.Metrics["memory.budget_bytes_total"])
	}
}

func TestOpenReportsDiscoveryErrorReasonCode(t *testing.T) {
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
