package accel

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
)

// wgpuDisableEnv turns the builtin GPU backend off without changing code. Set
// it when a host has a GPU that should not be used, or in tests that assert
// behaviour on a machine with no accelerator. INSYRA_ACCEL_DISABLE_NATIVE_PROBES
// also disables it, because that flag means "do not look at the real machine"
// and a test that sets it must not then discover the host's GPU.
const wgpuDisableEnv = "INSYRA_ACCEL_DISABLE_WGPU"

func wgpuDisabled() bool {
	if nativeProbesDisabled() {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv(wgpuDisableEnv))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// The GPU backend registers itself when the accel package is initialised, so a
// consumer needs nothing beyond importing accel. Probing is lazy — registration
// only records the seams, and no device is opened until discovery runs.
func init() {
	// gogpu reports Metal on Apple and Vulkan or DX12 elsewhere. Register one
	// probe per accel backend kind so a device is only ever offered under the
	// backend it belongs to, and never enumerated twice.
	RegisterSDKProbe(wgpuProbe{backend: BackendMetal})
	RegisterSDKProbe(wgpuProbe{backend: BackendWebGPU})
	_ = RegisterBackendExecutor(BackendMetal, wgpuExecutor{})
	_ = RegisterBackendExecutor(BackendWebGPU, wgpuExecutor{})
}

type wgpuProbe struct {
	backend Backend
}

func (p wgpuProbe) Name() string { return "gogpu-wgpu-" + string(p.backend) }

func (p wgpuProbe) Backend() Backend { return p.backend }

func (p wgpuProbe) Probe(_ Config) ([]Device, error) {
	if wgpuDisabled() {
		return nil, ErrSDKProbeUnavailable
	}
	info, err := wgpu.Probe()
	if err != nil {
		// A missing driver, a headless host, or a software-only adapter all mean
		// the same thing to discovery: this backend has nothing to offer.
		return nil, ErrSDKProbeUnavailable
	}
	device := wgpuDevice(info)
	if device.Backend != p.backend {
		return nil, ErrSDKProbeUnavailable
	}
	return []Device{device}, nil
}

func wgpuDevice(info wgpu.Info) Device {
	backend := BackendWebGPU
	if info.IsMetal {
		backend = BackendMetal
	}
	memoryClass, deviceType, score := MemoryClassDevice, DeviceTypeDiscrete, 90.0
	if info.UnifiedMemory {
		memoryClass, deviceType, score = MemoryClassShared, DeviceTypeIntegrated, 70.0
	}
	return Device{
		ID:            fmt.Sprintf("%s:wgpu:0", backend),
		Name:          info.Name,
		Vendor:        info.Vendor,
		Backend:       backend,
		ProbeSource:   ProbeSourceSDK,
		Type:          deviceType,
		MemoryClass:   memoryClass,
		SharedMemory:  info.UnifiedMemory,
		BudgetBytes:   info.MaxBufferBytes,
		DriverVersion: info.Driver,
		CapabilitySummary: map[string]bool{
			"compute_shaders": true,
			"float32":         true,
			"float64":         false,
			"shared_memory":   info.UnifiedMemory,
		},
		Score: score,
	}
}

type wgpuExecutor struct{}

func (wgpuExecutor) Name() string { return "gogpu-wgpu" }

func (wgpuExecutor) Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	switch req.Op {
	case OpSum:
		return wgpuSum(ctx, req)
	case OpSquaredDistance:
		return wgpuDistances(ctx, req)
	case OpNearestQuery:
		return wgpuNearest(ctx, req)
	default:
		return ExecuteResponse{}, fmt.Errorf("accel: wgpu backend does not support operation %q", req.Op)
	}
}

func wgpuDistances(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	columns := make([]wgpu.Column, len(req.Columns))
	for i, column := range req.Columns {
		columns[i] = wgpu.Column{Name: column.Name, Values: column.Values}
	}
	distances, cost, err := wgpu.SquaredDistances(ctx, columns, req.Queries)
	if err != nil {
		return ExecuteResponse{}, translateWGPUError(err)
	}
	return ExecuteResponse{
		Distances:     distances,
		Transfer:      cost.Transfer,
		Dispatch:      cost.Dispatch,
		Readback:      cost.Readback,
		BytesUploaded: cost.BytesUploaded,
	}, nil
}

func wgpuNearest(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	columns := make([]wgpu.Column, len(req.Columns))
	for i, column := range req.Columns {
		columns[i] = wgpu.Column{Name: column.Name, Values: column.Values}
	}
	indices, distances, cost, err := wgpu.NearestQuery(ctx, columns, req.Queries)
	if err != nil {
		return ExecuteResponse{}, translateWGPUError(err)
	}
	return ExecuteResponse{
		NearestIndex:  indices,
		Distances:     distances,
		Transfer:      cost.Transfer,
		Dispatch:      cost.Dispatch,
		Readback:      cost.Readback,
		BytesUploaded: cost.BytesUploaded,
	}, nil
}

func wgpuSum(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error) {
	if len(req.Columns) == 0 {
		return ExecuteResponse{}, fmt.Errorf("accel: wgpu backend received no columns")
	}
	columns := make([]wgpu.Column, len(req.Columns))
	for i, column := range req.Columns {
		columns[i] = wgpu.Column{Name: column.Name, Values: column.Values}
	}
	sums, cost, err := wgpu.SumColumns(ctx, columns)
	if err != nil {
		return ExecuteResponse{}, translateWGPUError(err)
	}
	return ExecuteResponse{
		Reductions:    sums,
		Transfer:      cost.Transfer,
		Dispatch:      cost.Dispatch,
		Readback:      cost.Readback,
		BytesUploaded: cost.BytesUploaded,
	}, nil
}

// translateWGPUError maps the backend's own sentinels onto the runtime's, so
// the fallback reason a user sees does not depend on which backend ran.
func translateWGPUError(err error) error {
	switch {
	case errors.Is(err, wgpu.ErrShaderCompile):
		return fmt.Errorf("%w: %v", ErrShaderCompile, err)
	case errors.Is(err, wgpu.ErrBufferTooLarge):
		return fmt.Errorf("%w: %v", ErrBufferTooLarge, err)
	case errors.Is(err, wgpu.ErrReadbackTimeout):
		return fmt.Errorf("%w: %v", ErrReadbackTimeout, err)
	default:
		return err
	}
}
