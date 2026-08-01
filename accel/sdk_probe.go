package accel

import (
	"errors"
	"sync"
)

// ErrSDKProbeUnavailable signals that an SDK-backed probe could not run on
// this host (driver missing, library not loaded, unsupported platform, etc.).
// Discoverers treat this as a clean miss and fall back to native command
// probes followed by env stubs.
var ErrSDKProbeUnavailable = errors.New("accel: sdk probe unavailable")

// SDKProbe is a backend-specific probe that talks directly to the vendor SDK
// (NVML for CUDA, Metal API for Apple, wgpu-native for WebGPU, …) instead of
// shelling out to a host command. Probes that succeed should set
// Device.ProbeSource to ProbeSourceSDK; the discoverer normalizes the field if
// the probe leaves it blank.
type SDKProbe interface {
	Name() string
	Backend() Backend
	Probe(cfg Config) ([]Device, error)
}

var (
	sdkProbesMu sync.RWMutex
	sdkProbes   = map[Backend][]SDKProbe{}
)

// RegisterSDKProbe wires an SDK probe into the discoverer pipeline for its
// backend. Multiple probes per backend are tried in registration order; the
// first one that returns devices wins. The pipeline ordering is:
// SDK > native command > env stub.
func RegisterSDKProbe(probe SDKProbe) {
	if probe == nil {
		return
	}
	backend := probe.Backend()
	sdkProbesMu.Lock()
	defer sdkProbesMu.Unlock()
	sdkProbes[backend] = append(sdkProbes[backend], probe)
}

// ResetSDKProbesForTest clears every registered SDK probe. Tests use this to
// isolate themselves from probes that may have been registered by init().
func ResetSDKProbesForTest() {
	sdkProbesMu.Lock()
	defer sdkProbesMu.Unlock()
	sdkProbes = map[Backend][]SDKProbe{}
}

func sdkProbesForBackend(backend Backend) []SDKProbe {
	sdkProbesMu.RLock()
	defer sdkProbesMu.RUnlock()
	probes := sdkProbes[backend]
	if len(probes) == 0 {
		return nil
	}
	cloned := make([]SDKProbe, len(probes))
	copy(cloned, probes)
	return cloned
}

func runSDKProbes(backend Backend, cfg Config) ([]Device, error) {
	probes := sdkProbesForBackend(backend)
	if len(probes) == 0 {
		return nil, ErrSDKProbeUnavailable
	}
	var lastErr error
	for _, probe := range probes {
		devices, err := probe.Probe(cfg)
		if err == nil && len(devices) > 0 {
			for idx := range devices {
				if devices[idx].ProbeSource == "" || devices[idx].ProbeSource == ProbeSourceUnknown {
					devices[idx].ProbeSource = ProbeSourceSDK
				}
			}
			return devices, nil
		}
		if err != nil && !errors.Is(err, ErrSDKProbeUnavailable) {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrSDKProbeUnavailable
}
