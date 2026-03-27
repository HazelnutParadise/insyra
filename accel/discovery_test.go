package accel

import "testing"

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
