package accel

import (
	"errors"
	"fmt"
)

// nvmlDeviceInfo is the subset of NVML device data we surface in v1. Future
// slices can extend this struct; consumers should not assume any field is
// populated (e.g. PCIBusID is currently filled by the nvidia-smi enriched
// probe rather than NVML).
type nvmlDeviceInfo struct {
	Name             string
	TotalMemoryBytes uint64
	ComputeMajor     int
	ComputeMinor     int
}

// nvmlLoader abstracts the platform-specific NVML library binding so the
// generic NVML probe stays testable. Implementations live in
// sdk_probe_nvml_<os>.go.
type nvmlLoader interface {
	Init() error
	DriverVersion() (string, error)
	DeviceCount() (int, error)
	Device(index int) (nvmlDeviceInfo, error)
	Shutdown() error
}

// nvmlOpenFunc resolves a fresh nvmlLoader at probe time. Returning
// ErrSDKProbeUnavailable indicates the host does not have NVML installed and
// the discoverer should fall through to native command probes / env stubs.
type nvmlOpenFunc func() (nvmlLoader, error)

type nvmlSDKProbe struct {
	open nvmlOpenFunc
}

func newNVMLSDKProbe(open nvmlOpenFunc) SDKProbe {
	return nvmlSDKProbe{open: open}
}

func (p nvmlSDKProbe) Name() string     { return "nvml" }
func (p nvmlSDKProbe) Backend() Backend { return BackendCUDA }

func (p nvmlSDKProbe) Probe(cfg Config) ([]Device, error) {
	if p.open == nil {
		return nil, ErrSDKProbeUnavailable
	}
	loader, err := p.open()
	if err != nil {
		if errors.Is(err, ErrSDKProbeUnavailable) {
			return nil, err
		}
		return nil, ErrSDKProbeUnavailable
	}
	if err := loader.Init(); err != nil {
		return nil, ErrSDKProbeUnavailable
	}
	defer func() { _ = loader.Shutdown() }()

	driver, _ := loader.DriverVersion()

	count, err := loader.DeviceCount()
	if err != nil {
		return nil, ErrSDKProbeUnavailable
	}
	if count <= 0 {
		return nil, ErrSDKProbeUnavailable
	}

	devices := make([]Device, 0, count)
	for idx := 0; idx < count; idx++ {
		info, err := loader.Device(idx)
		if err != nil {
			continue
		}
		compute := ""
		if info.ComputeMajor > 0 || info.ComputeMinor > 0 {
			compute = fmt.Sprintf("%d.%d", info.ComputeMajor, info.ComputeMinor)
		}
		device := Device{
			ID:                fmt.Sprintf("cuda:nvml:%d", idx),
			Name:              info.Name,
			Vendor:            "nvidia",
			Backend:           BackendCUDA,
			ProbeSource:       ProbeSourceSDK,
			Type:              DeviceTypeDiscrete,
			MemoryClass:       MemoryClassDevice,
			BudgetBytes:       info.TotalMemoryBytes,
			DriverVersion:     driver,
			ComputeCapability: compute,
			CapabilitySummary: map[string]bool{
				"multi_gpu":           true,
				"device_local_memory": true,
				"sdk_probe":           true,
			},
		}
		devices = append(devices, device)
	}
	if len(devices) == 0 {
		return nil, ErrSDKProbeUnavailable
	}
	return devices, nil
}
