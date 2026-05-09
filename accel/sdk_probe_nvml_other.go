//go:build !windows

package accel

// On non-Windows hosts the NVML SDK seam exists but no probe is registered at
// init() yet. Linux/macOS NVML support is a future slice (see
// add-accel-backend-discovery design notes); leaving this file as a deliberate
// no-op keeps the seam observable and avoids registering anything that would
// silently fall through to ErrSDKProbeUnavailable on every Probe() call.
//
// Tests can still register a fake CUDA SDKProbe via RegisterSDKProbe to
// exercise the SDK-priority path on any platform.

var _ = newNVMLSDKProbe // keep the cross-platform seam exported for tests
