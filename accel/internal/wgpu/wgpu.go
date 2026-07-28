// Package wgpu runs numeric kernels on a GPU through the pure-Go WebGPU
// implementation in github.com/gogpu/wgpu.
//
// It is deliberately ignorant of the accel runtime: it speaks in its own small
// vocabulary so that accel can import it without a cycle. The adapter that maps
// this onto accel's Device and ExecuteRequest lives in accel/backend_wgpu.go.
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

	"github.com/gogpu/gputypes"
	gowgpu "github.com/gogpu/wgpu"

	_ "github.com/gogpu/wgpu/hal/allbackends"
)

// Info describes the adapter this package found.
type Info struct {
	Name           string
	Vendor         string
	Driver         string
	IsMetal        bool
	UnifiedMemory  bool
	MaxBufferBytes uint64
}

// Cost is what one execution actually took. The durations are host-observed:
// Metal and GLES do not implement GPU timestamp queries, so device-side timing
// is not available on every backend.
type Cost struct {
	Transfer      time.Duration
	Dispatch      time.Duration
	Readback      time.Duration
	BytesUploaded uint64
}

// The three failure modes worth telling apart, plus "there is no usable GPU".
var (
	ErrUnavailable     = errors.New("wgpu: no usable GPU on this host")
	ErrShaderCompile   = errors.New("wgpu: shader compilation failed")
	ErrBufferTooLarge  = errors.New("wgpu: column exceeds device buffer limit")
	ErrReadbackTimeout = errors.New("wgpu: device readback timed out")
)

// Probe reports the adapter, or ErrUnavailable when there is no GPU worth using.
func Probe() (Info, error) {
	h, err := acquire()
	if err != nil {
		return Info{}, ErrUnavailable
	}
	return h.info0(), nil
}

func (h *handle) info0() Info {
	return Info{
		Name:           h.info.Name,
		Vendor:         strings.ToLower(h.info.Vendor),
		Driver:         h.info.Driver,
		IsMetal:        h.info.Backend == gputypes.BackendMetal,
		UnifiedMemory:  unifiedMemory(h.info),
		MaxBufferBytes: h.limits.MaxStorageBufferBindingSize,
	}
}

// submitMu serializes device submission process-wide. Every session shares one
// device handle, and gogpu's queue-concurrency guarantees are unverified;
// concurrent submissions would serialize on the hardware queue anyway.
var submitMu sync.Mutex

// Sum reduces the values on the device and returns the total. Nulls are
// expected to have been replaced by the additive identity by the caller.
func Sum(ctx context.Context, values []float32) (float64, Cost, error) {
	submitMu.Lock()
	defer submitMu.Unlock()

	h, err := acquire()
	if err != nil {
		return 0, Cost{}, err
	}
	pipeline, bgLayout, err := h.computePipeline()
	if err != nil {
		return 0, Cost{}, err
	}
	chunkSize := h.maxElementsPerChunk()
	if chunkSize <= 0 {
		return 0, Cost{}, fmt.Errorf("%w: device reports no bindable storage", ErrBufferTooLarge)
	}

	var total float64
	var cost Cost
	for start := 0; start < len(values); start += chunkSize {
		end := start + chunkSize
		if end > len(values) {
			end = len(values)
		}
		partial, chunk, err := h.sumChunk(ctx, pipeline, bgLayout, values[start:end])
		if err != nil {
			return 0, Cost{}, err
		}
		total += partial
		cost.Transfer += chunk.transfer
		cost.Dispatch += chunk.dispatch
		cost.Readback += chunk.readback
		cost.BytesUploaded += chunk.uploaded
	}
	return total, cost, nil
}

// unifiedMemory decides unified versus discrete memory from the vendor and the
// backend rather than from AdapterInfo.DeviceType, which reports an Apple M3 as
// DiscreteGPU even though Apple Silicon has unified memory.
func unifiedMemory(info gputypes.AdapterInfo) bool {
	vendor := strings.ToLower(info.Vendor)
	if info.Backend == gputypes.BackendMetal || strings.Contains(vendor, "apple") {
		return true
	}
	if strings.Contains(vendor, "intel") {
		return true
	}
	return info.DeviceType == gputypes.DeviceTypeIntegratedGPU
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
			h.pipelineErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
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
			h.pipelineErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
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
		return 0, cost, fmt.Errorf("%w: %v", ErrBufferTooLarge, err)
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
			return 0, cost, fmt.Errorf("%w: %v", ErrReadbackTimeout, err)
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
