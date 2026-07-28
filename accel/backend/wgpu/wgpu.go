// Package wgpu runs Insyra accel workloads on a real GPU through the pure-Go
// WebGPU implementation in github.com/gogpu/wgpu.
//
// Import it for its side effect to make GPU execution available:
//
//	import _ "github.com/HazelnutParadise/insyra/accel/backend/wgpu"
//
// One WGSL source reaches Metal on macOS, Vulkan on Linux and Windows, and
// DirectX 12 on Windows, and it builds with CGO_ENABLED=0.
//
// WGSL has no f64 and Apple GPUs have no double-precision hardware, so the
// runtime narrows columns to float32 and only does so when the caller has
// explicitly accepted single precision. See accel.Precision.
package wgpu

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/HazelnutParadise/insyra/accel"
	"github.com/gogpu/gputypes"
	gowgpu "github.com/gogpu/wgpu"

	_ "github.com/gogpu/wgpu/hal/allbackends"
)

func init() {
	// gogpu reports Metal on Apple and Vulkan or DX12 elsewhere. Register one
	// probe per accel backend kind so a device is only ever offered under the
	// backend it actually belongs to, and never enumerated twice.
	accel.RegisterSDKProbe(probe{backend: accel.BackendMetal})
	accel.RegisterSDKProbe(probe{backend: accel.BackendWebGPU})
	_ = accel.RegisterBackendExecutor(accel.BackendMetal, executor{})
	_ = accel.RegisterBackendExecutor(accel.BackendWebGPU, executor{})
}

const (
	workgroupSize = 256
	// Each thread folds this many strided elements before the tree reduction,
	// which keeps the dispatch under the workgroup-count limit.
	elemsPerThread = 8
	elemsPerGroup  = workgroupSize * elemsPerThread
	// Every backend gogpu targets reports 65535 as the maximum workgroup count
	// per dimension.
	maxWorkgroups       = 65535
	maxElemsPerDispatch = maxWorkgroups * elemsPerGroup
)

// sumWGSL reduces one chunk of a float32 column to one partial per workgroup.
// The host folds the partials in float64, so only the within-workgroup tree
// runs at single precision.
const sumWGSL = `
@group(0) @binding(0) var<storage, read> input: array<f32>;
@group(0) @binding(1) var<storage, read_write> partials: array<f32>;

struct Params { count: u32, }
@group(0) @binding(2) var<uniform> params: Params;

var<workgroup> scratch: array<f32, 256>;

@compute @workgroup_size(256)
fn main(@builtin(local_invocation_id) lid: vec3<u32>,
        @builtin(workgroup_id)        wid: vec3<u32>) {
    let base = wid.x * 256u * 8u + lid.x;
    var v: f32 = 0.0;
    for (var k: u32 = 0u; k < 8u; k = k + 1u) {
        let idx = base + k * 256u;
        if (idx < params.count) { v = v + input[idx]; }
    }
    scratch[lid.x] = v;
    workgroupBarrier();

    var stride: u32 = 128u;
    loop {
        if (stride == 0u) { break; }
        if (lid.x < stride) { scratch[lid.x] = scratch[lid.x] + scratch[lid.x + stride]; }
        workgroupBarrier();
        stride = stride / 2u;
    }
    if (lid.x == 0u) { partials[wid.x] = scratch[0]; }
}
`

// readbackTimeout bounds the wait for a mapped staging buffer. A stuck queue
// otherwise hangs the caller with no way out.
const readbackTimeout = 30 * time.Second

// -----------------------------------------------------------------------------
// Device handle
// -----------------------------------------------------------------------------

// handle owns the one instance/adapter/device triple this process uses. Creating
// a device per call would pay adapter enumeration and shader compilation on
// every operation.
type handle struct {
	instance *gowgpu.Instance
	adapter  *gowgpu.Adapter
	device   *gowgpu.Device
	info     gputypes.AdapterInfo
	limits   gputypes.Limits

	pipelineOnce sync.Once
	shader       *gowgpu.ShaderModule
	bgLayout     *gowgpu.BindGroupLayout
	plLayout     *gowgpu.PipelineLayout
	pipeline     *gowgpu.ComputePipeline
	pipelineErr  error
}

var (
	sharedOnce sync.Once
	shared     *handle
	sharedErr  error
)

// errSoftwareAdapter is returned when the only adapter on offer is gogpu's
// SPIR-V interpreter. Reporting that as an acceleration device would mean
// calling a CPU interpreter a GPU, which is the failure this backend exists to
// avoid.
var errSoftwareAdapter = errors.New("wgpu: only a software (CPU) adapter is available")

func acquire() (*handle, error) {
	sharedOnce.Do(func() {
		shared, sharedErr = open()
	})
	return shared, sharedErr
}

func open() (*handle, error) {
	instance, err := gowgpu.CreateInstance(nil)
	if err != nil {
		return nil, fmt.Errorf("wgpu: create instance: %w", err)
	}
	adapter, err := instance.RequestAdapter(nil)
	if err != nil {
		instance.Release()
		return nil, fmt.Errorf("wgpu: request adapter: %w", err)
	}
	info := adapter.Info()
	if isSoftwareAdapter(info) {
		adapter.Release()
		instance.Release()
		return nil, errSoftwareAdapter
	}
	device, err := adapter.RequestDevice(nil)
	if err != nil {
		adapter.Release()
		instance.Release()
		return nil, fmt.Errorf("wgpu: request device: %w", err)
	}
	return &handle{
		instance: instance,
		adapter:  adapter,
		device:   device,
		info:     info,
		limits:   adapter.Limits(),
	}, nil
}

func isSoftwareAdapter(info gputypes.AdapterInfo) bool {
	if info.DeviceType == gputypes.DeviceTypeCPU {
		return true
	}
	return strings.Contains(strings.ToLower(info.Name), "software")
}

func (h *handle) computePipeline() (*gowgpu.ComputePipeline, *gowgpu.BindGroupLayout, error) {
	h.pipelineOnce.Do(func() {
		shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
			Label: "accel-sum", WGSL: sumWGSL,
		})
		if err != nil {
			h.pipelineErr = fmt.Errorf("%w: %v", accel.ErrShaderCompile, err)
			return
		}
		h.shader = shader

		bgLayout, err := h.device.CreateBindGroupLayout(&gowgpu.BindGroupLayoutDescriptor{
			Label: "accel-sum-bgl",
			Entries: []gowgpu.BindGroupLayoutEntry{
				{Binding: 0, Visibility: gowgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}},
				{Binding: 1, Visibility: gowgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}},
				{Binding: 2, Visibility: gowgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}},
			},
		})
		if err != nil {
			h.pipelineErr = fmt.Errorf("wgpu: bind group layout: %w", err)
			return
		}
		h.bgLayout = bgLayout

		plLayout, err := h.device.CreatePipelineLayout(&gowgpu.PipelineLayoutDescriptor{
			Label: "accel-sum-pl", BindGroupLayouts: []*gowgpu.BindGroupLayout{bgLayout},
		})
		if err != nil {
			h.pipelineErr = fmt.Errorf("wgpu: pipeline layout: %w", err)
			return
		}
		h.plLayout = plLayout

		pipeline, err := h.device.CreateComputePipeline(&gowgpu.ComputePipelineDescriptor{
			Label: "accel-sum-pipeline", Layout: plLayout, Module: shader, EntryPoint: "main",
		})
		if err != nil {
			h.pipelineErr = fmt.Errorf("%w: %v", accel.ErrShaderCompile, err)
			return
		}
		h.pipeline = pipeline
	})
	return h.pipeline, h.bgLayout, h.pipelineErr
}

// maxElementsPerChunk is the largest number of float32 values one dispatch can
// take, bounded by both the storage-buffer limit and the workgroup-count limit.
func (h *handle) maxElementsPerChunk() int {
	byLimit := maxElemsPerDispatch
	if bindable := int(h.limits.MaxStorageBufferBindingSize / 4); bindable > 0 && bindable < byLimit {
		byLimit = bindable
	}
	// Keep chunks aligned to whole workgroups so the last group is never partial
	// for reasons other than the column ending.
	return (byLimit / elemsPerGroup) * elemsPerGroup
}

// -----------------------------------------------------------------------------
// Discovery
// -----------------------------------------------------------------------------

type probe struct {
	backend accel.Backend
}

func (p probe) Name() string { return "gogpu-wgpu-" + string(p.backend) }

func (p probe) Backend() accel.Backend { return p.backend }

func (p probe) Probe(_ accel.Config) ([]accel.Device, error) {
	h, err := acquire()
	if err != nil {
		// A missing driver, a headless host, or a software-only adapter all mean
		// the same thing to discovery: this backend has nothing to offer.
		return nil, accel.ErrSDKProbeUnavailable
	}
	device := h.device0()
	if device.Backend != p.backend {
		return nil, accel.ErrSDKProbeUnavailable
	}
	return []accel.Device{device}, nil
}

func (h *handle) device0() accel.Device {
	backend := backendFor(h.info)
	shared := sharedMemory(h.info)
	memoryClass := accel.MemoryClassDevice
	deviceType := accel.DeviceTypeDiscrete
	if shared {
		memoryClass = accel.MemoryClassShared
		deviceType = accel.DeviceTypeIntegrated
	}
	return accel.Device{
		ID:            fmt.Sprintf("%s:wgpu:0", backend),
		Name:          h.info.Name,
		Vendor:        strings.ToLower(h.info.Vendor),
		Backend:       backend,
		ProbeSource:   accel.ProbeSourceSDK,
		Type:          deviceType,
		MemoryClass:   memoryClass,
		SharedMemory:  shared,
		BudgetBytes:   h.limits.MaxStorageBufferBindingSize,
		DriverVersion: h.info.Driver,
		CapabilitySummary: map[string]bool{
			"compute_shaders": true,
			"float32":         true,
			"float64":         false,
			"shared_memory":   shared,
		},
		Score: scoreFor(shared),
	}
}

func backendFor(info gputypes.AdapterInfo) accel.Backend {
	if info.Backend == gputypes.BackendMetal {
		return accel.BackendMetal
	}
	return accel.BackendWebGPU
}

// sharedMemory decides unified versus discrete memory from the vendor and the
// backend rather than from AdapterInfo.DeviceType, which reports an Apple M3 as
// DiscreteGPU even though Apple Silicon has unified memory.
func sharedMemory(info gputypes.AdapterInfo) bool {
	vendor := strings.ToLower(info.Vendor)
	if info.Backend == gputypes.BackendMetal || strings.Contains(vendor, "apple") {
		return true
	}
	if strings.Contains(vendor, "intel") {
		return true
	}
	return info.DeviceType == gputypes.DeviceTypeIntegratedGPU
}

func scoreFor(shared bool) float64 {
	if shared {
		return 70
	}
	return 90
}

// -----------------------------------------------------------------------------
// Execution
// -----------------------------------------------------------------------------

type executor struct{}

func (executor) Name() string { return "gogpu-wgpu" }

func (executor) Execute(ctx context.Context, req accel.ExecuteRequest) (accel.ExecuteResponse, error) {
	if req.Op != accel.OpSum {
		return accel.ExecuteResponse{}, fmt.Errorf("wgpu: unsupported operation %q", req.Op)
	}
	h, err := acquire()
	if err != nil {
		return accel.ExecuteResponse{}, err
	}
	pipeline, bgLayout, err := h.computePipeline()
	if err != nil {
		return accel.ExecuteResponse{}, err
	}

	chunkSize := h.maxElementsPerChunk()
	if chunkSize <= 0 {
		return accel.ExecuteResponse{}, fmt.Errorf("%w: device reports no bindable storage", accel.ErrBufferTooLarge)
	}

	var response accel.ExecuteResponse
	for start := 0; start < len(req.Values); start += chunkSize {
		end := start + chunkSize
		if end > len(req.Values) {
			end = len(req.Values)
		}
		partial, chunkCost, err := h.sumChunk(ctx, pipeline, bgLayout, req.Values[start:end])
		if err != nil {
			return accel.ExecuteResponse{}, err
		}
		response.Value += partial
		response.Transfer += chunkCost.transfer
		response.Dispatch += chunkCost.dispatch
		response.Readback += chunkCost.readback
		response.BytesUploaded += chunkCost.uploaded
	}
	return response, nil
}

type chunkCost struct {
	transfer time.Duration
	dispatch time.Duration
	readback time.Duration
	uploaded uint64
}

func (h *handle) sumChunk(
	ctx context.Context,
	pipeline *gowgpu.ComputePipeline,
	bgLayout *gowgpu.BindGroupLayout,
	values []float32,
) (float64, chunkCost, error) {
	var cost chunkCost
	groups := (len(values) + elemsPerGroup - 1) / elemsPerGroup
	inputBytes := uint64(len(values) * 4)
	partialBytes := uint64(groups * 4)

	inputBuf, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-input", Size: inputBytes,
		Usage: gowgpu.BufferUsageStorage | gowgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return 0, cost, fmt.Errorf("%w: %v", accel.ErrBufferTooLarge, err)
	}
	defer inputBuf.Release()

	partialBuf, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-partials", Size: partialBytes,
		Usage: gowgpu.BufferUsageStorage | gowgpu.BufferUsageCopySrc,
	})
	if err != nil {
		return 0, cost, fmt.Errorf("wgpu: partial buffer: %w", err)
	}
	defer partialBuf.Release()

	stagingBuf, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-staging", Size: partialBytes,
		Usage: gowgpu.BufferUsageCopyDst | gowgpu.BufferUsageMapRead,
	})
	if err != nil {
		return 0, cost, fmt.Errorf("wgpu: staging buffer: %w", err)
	}
	defer stagingBuf.Release()

	paramsBuf, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-params", Size: 4,
		Usage: gowgpu.BufferUsageUniform | gowgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return 0, cost, fmt.Errorf("wgpu: params buffer: %w", err)
	}
	defer paramsBuf.Release()

	params := make([]byte, 4)
	binary.LittleEndian.PutUint32(params, uint32(len(values)))

	transferStart := time.Now()
	if err := h.device.Queue().WriteBuffer(inputBuf, 0, float32Bytes(values)); err != nil {
		return 0, cost, fmt.Errorf("wgpu: upload column: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(paramsBuf, 0, params); err != nil {
		return 0, cost, fmt.Errorf("wgpu: upload params: %w", err)
	}
	cost.transfer = time.Since(transferStart)
	cost.uploaded = inputBytes

	bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
		Label: "accel-sum-bg", Layout: bgLayout,
		Entries: []gowgpu.BindGroupEntry{
			{Binding: 0, Buffer: inputBuf, Size: inputBytes},
			{Binding: 1, Buffer: partialBuf, Size: partialBytes},
			{Binding: 2, Buffer: paramsBuf, Size: 4},
		},
	})
	if err != nil {
		return 0, cost, fmt.Errorf("wgpu: bind group: %w", err)
	}
	defer bindGroup.Release()

	// Encode and submit. The queue is asynchronous, so most of the device time
	// lands in the map wait below rather than here.
	dispatchStart := time.Now()
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return 0, cost, fmt.Errorf("wgpu: command encoder: %w", err)
	}
	pass, err := encoder.BeginComputePass(nil)
	if err != nil {
		return 0, cost, fmt.Errorf("wgpu: compute pass: %w", err)
	}
	pass.SetPipeline(pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Dispatch(uint32(groups), 1, 1)
	if err := pass.End(); err != nil {
		return 0, cost, fmt.Errorf("wgpu: end compute pass: %w", err)
	}
	encoder.CopyBufferToBuffer(partialBuf, 0, stagingBuf, 0, partialBytes)
	commands, err := encoder.Finish()
	if err != nil {
		return 0, cost, fmt.Errorf("wgpu: finish encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return 0, cost, fmt.Errorf("wgpu: submit: %w", err)
	}
	cost.dispatch = time.Since(dispatchStart)

	readbackStart := time.Now()
	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := stagingBuf.Map(mapCtx, gowgpu.MapModeRead, 0, partialBytes); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(mapCtx.Err(), context.DeadlineExceeded) {
			return 0, cost, fmt.Errorf("%w: %v", accel.ErrReadbackTimeout, err)
		}
		return 0, cost, fmt.Errorf("wgpu: map staging buffer: %w", err)
	}
	mapped, err := stagingBuf.MappedRange(0, partialBytes)
	if err != nil {
		_ = stagingBuf.Unmap()
		return 0, cost, fmt.Errorf("wgpu: mapped range: %w", err)
	}
	raw := mapped.Bytes()
	// Fold the per-workgroup partials in float64: only the reduction inside a
	// workgroup runs at single precision.
	sum := 0.0
	for i := 0; i < groups; i++ {
		sum += float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:])))
	}
	if err := stagingBuf.Unmap(); err != nil {
		return 0, cost, fmt.Errorf("wgpu: unmap staging buffer: %w", err)
	}
	cost.readback = time.Since(readbackStart)

	return sum, cost, nil
}

func float32Bytes(values []float32) []byte {
	if len(values) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&values[0])), len(values)*4)
}
