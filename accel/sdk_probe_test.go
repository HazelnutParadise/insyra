package accel

import (
	"errors"
	"testing"
)

type fakeSDKProbe struct {
	backend Backend
	devices []Device
	err     error
	calls   int
}

func (p *fakeSDKProbe) Name() string     { return "fake-sdk" }
func (p *fakeSDKProbe) Backend() Backend { return p.backend }
func (p *fakeSDKProbe) Probe(cfg Config) ([]Device, error) {
	p.calls++
	return p.devices, p.err
}

func TestSDKProbeWinsOverNativeAndStub(t *testing.T) {
	ResetDiscoverersForTest()
	t.Cleanup(ResetDiscoverersForTest)
	resetBuiltinProbeOverridesForTest()
	t.Cleanup(resetBuiltinProbeOverridesForTest)

	nativeCalls := 0
	setBuiltinProbeOverrideForTest(
		BackendCUDA,
		func(cfg Config) ([]Device, error) {
			nativeCalls++
			return []Device{{ID: "cuda:native:0", Backend: BackendCUDA}}, nil
		},
		nil,
	)
	setBuiltinProbeOverrideForTest(
		BackendWebGPU,
		func(cfg Config) ([]Device, error) { return nil, ErrNativeProbeUnavailable },
		nil,
	)
	t.Setenv("INSYRA_ACCEL_STUB_CUDA", "1")

	probe := &fakeSDKProbe{
		backend: BackendCUDA,
		devices: []Device{
			{
				ID:                "cuda:nvml:0",
				Name:              "Fake RTX",
				Vendor:            "nvidia",
				Backend:           BackendCUDA,
				Type:              DeviceTypeDiscrete,
				MemoryClass:       MemoryClassDevice,
				BudgetBytes:       16 * 1024 * 1024 * 1024,
				DriverVersion:     "555.42",
				ComputeCapability: "8.9",
			},
		},
	}
	RegisterSDKProbe(probe)

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	if probe.calls == 0 {
		t.Fatal("expected SDK probe to be called")
	}
	if nativeCalls != 0 {
		t.Fatalf("native probe should be skipped when SDK probe wins, called %d times", nativeCalls)
	}

	devices := session.Devices()
	if len(devices) != 1 {
		t.Fatalf("expected one cuda device from SDK probe, got %d", len(devices))
	}
	device := devices[0]
	if device.ID != "cuda:nvml:0" {
		t.Fatalf("expected SDK device id, got %q", device.ID)
	}
	if device.ProbeSource != ProbeSourceSDK {
		t.Fatalf("expected sdk probe source, got %q", device.ProbeSource)
	}
	if device.DriverVersion != "555.42" {
		t.Fatalf("expected driver version to propagate, got %q", device.DriverVersion)
	}
	if device.ComputeCapability != "8.9" {
		t.Fatalf("expected compute capability to propagate, got %q", device.ComputeCapability)
	}
	if !device.CapabilitySummary["sdk_probe"] {
		t.Fatal("expected sdk_probe capability flag to be true for SDK-discovered device")
	}
	if device.CapabilitySummary["native_probe"] {
		t.Fatal("did not expect native_probe flag for SDK-discovered device")
	}

	report := session.Report()
	if report.Metrics["devices.sdk"] != 1 {
		t.Fatalf("expected 1 sdk-discovered device, got %v", report.Metrics["devices.sdk"])
	}
}

func TestSDKProbeUnavailableFallsThroughToNative(t *testing.T) {
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

	RegisterSDKProbe(&fakeSDKProbe{backend: BackendCUDA, err: ErrSDKProbeUnavailable})

	session, err := Open(Config{})
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	devices := session.Devices()
	if len(devices) != 1 {
		t.Fatalf("expected fallback to native cuda device, got %d", len(devices))
	}
	if devices[0].ProbeSource != ProbeSourceNative {
		t.Fatalf("expected native probe source after sdk probe miss, got %q", devices[0].ProbeSource)
	}
}

func TestSDKProbeRealErrorPropagatesAsDiscoveryError(t *testing.T) {
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

	RegisterSDKProbe(&fakeSDKProbe{backend: BackendCUDA, err: errors.New("nvml exploded")})

	session, err := Open(Config{})
	if err == nil {
		t.Fatal("expected discovery to surface the sdk probe error")
	}
	if session == nil {
		t.Fatal("expected a session even when sdk probe returns a non-unavailable error")
	}
	t.Cleanup(func() { _ = session.Close() })

	report := session.Report()
	if report.FallbackReason != FallbackReasonDiscoveryError {
		t.Fatalf("expected discovery-error fallback, got %q", report.FallbackReason)
	}
}

type fakeNVMLLoader struct {
	driverVersion string
	devices       []nvmlDeviceInfo
	initErr       error
}

func (l *fakeNVMLLoader) Init() error                          { return l.initErr }
func (l *fakeNVMLLoader) Shutdown() error                      { return nil }
func (l *fakeNVMLLoader) DriverVersion() (string, error)       { return l.driverVersion, nil }
func (l *fakeNVMLLoader) DeviceCount() (int, error)            { return len(l.devices), nil }
func (l *fakeNVMLLoader) Device(idx int) (nvmlDeviceInfo, error) {
	if idx < 0 || idx >= len(l.devices) {
		return nvmlDeviceInfo{}, errors.New("out of range")
	}
	return l.devices[idx], nil
}

func TestNVMLProbeMapsLoaderOutputToDevices(t *testing.T) {
	probe := newNVMLSDKProbe(func() (nvmlLoader, error) {
		return &fakeNVMLLoader{
			driverVersion: "555.42.06",
			devices: []nvmlDeviceInfo{
				{
					Name:             "NVIDIA GeForce RTX 4090",
					TotalMemoryBytes: 24 * 1024 * 1024 * 1024,
					ComputeMajor:     8,
					ComputeMinor:     9,
				},
				{
					Name:             "NVIDIA RTX A2000",
					TotalMemoryBytes: 12 * 1024 * 1024 * 1024,
					ComputeMajor:     8,
					ComputeMinor:     6,
				},
			},
		}, nil
	})

	devices, err := probe.Probe(Config{})
	if err != nil {
		t.Fatalf("nvml probe failed: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].ID != "cuda:nvml:0" {
		t.Fatalf("expected first device id cuda:nvml:0, got %q", devices[0].ID)
	}
	if devices[0].DriverVersion != "555.42.06" {
		t.Fatalf("expected driver version 555.42.06, got %q", devices[0].DriverVersion)
	}
	if devices[0].ComputeCapability != "8.9" {
		t.Fatalf("expected compute capability 8.9, got %q", devices[0].ComputeCapability)
	}
	if devices[1].ComputeCapability != "8.6" {
		t.Fatalf("expected second device compute capability 8.6, got %q", devices[1].ComputeCapability)
	}
	if devices[0].BudgetBytes != 24*1024*1024*1024 {
		t.Fatalf("unexpected budget bytes for first device: %d", devices[0].BudgetBytes)
	}
	if devices[0].ProbeSource != ProbeSourceSDK {
		t.Fatalf("expected probe source sdk, got %q", devices[0].ProbeSource)
	}
}

func TestNVMLProbeReturnsUnavailableWhenLoaderInitFails(t *testing.T) {
	probe := newNVMLSDKProbe(func() (nvmlLoader, error) {
		return &fakeNVMLLoader{initErr: errors.New("driver missing")}, nil
	})

	_, err := probe.Probe(Config{})
	if !errors.Is(err, ErrSDKProbeUnavailable) {
		t.Fatalf("expected ErrSDKProbeUnavailable when loader init fails, got %v", err)
	}
}

func TestNVMLProbeReturnsUnavailableWhenOpenFails(t *testing.T) {
	probe := newNVMLSDKProbe(func() (nvmlLoader, error) {
		return nil, ErrSDKProbeUnavailable
	})

	_, err := probe.Probe(Config{})
	if !errors.Is(err, ErrSDKProbeUnavailable) {
		t.Fatalf("expected ErrSDKProbeUnavailable when open fails, got %v", err)
	}
}

func TestParseNvidiaSMIOutputCapturesRichFields(t *testing.T) {
	raw := []byte("0, NVIDIA GeForce RTX 4090, 24564, 555.42.06, 8.9, 00000000:01:00.0\n")

	devices, err := parseNvidiaSMIOutput(raw)
	if err != nil {
		t.Fatalf("parse nvidia-smi rich output failed: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	d := devices[0]
	if d.DriverVersion != "555.42.06" {
		t.Fatalf("expected driver version, got %q", d.DriverVersion)
	}
	if d.ComputeCapability != "8.9" {
		t.Fatalf("expected compute capability 8.9, got %q", d.ComputeCapability)
	}
	if d.PCIBusID != "00000000:01:00.0" {
		t.Fatalf("expected pci bus id, got %q", d.PCIBusID)
	}
	if d.BudgetBytes != 24564*1024*1024 {
		t.Fatalf("expected budget bytes derived from MiB, got %d", d.BudgetBytes)
	}
}
