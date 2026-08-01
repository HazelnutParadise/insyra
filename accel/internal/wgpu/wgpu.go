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

// Column is one named slice of values to reduce.
type Column struct {
	Name   string
	Values []float32
}

// MaxShortlist caps how many candidates the shortlist kernel keeps. The slots
// live in per-thread registers, so a wider list costs occupancy on every row
// whether or not the extra candidates are ever looked at.
const MaxShortlist = 8

// nearestShortlistWGSL keeps the k smallest distances per row instead of only
// the smallest, so the host can settle the ranking in float64 over a handful of
// candidates rather than over every query point.
//
// It actually keeps k+1. The extra one is never returned as a candidate: it is
// the best rejected distance, which is what tells the host whether the cut
// between candidate k and everything else is wide enough to trust in f32.
//
// Insertion is strict, and the bubble-up stops on equality, so equal distances
// keep the order they arrived in and ties resolve to the lowest query index —
// the same rule the CPU reference and the f32 kernel follow.
const nearestShortlistWGSL = `
@group(0) @binding(0) var<storage, read> data: array<f32>;
@group(0) @binding(1) var<storage, read> queries: array<f32>;
@group(0) @binding(2) var<storage, read_write> outDist: array<f32>;
@group(0) @binding(3) var<storage, read_write> outIdx: array<u32>;
@group(0) @binding(4) var<storage, read_write> outBoundary: array<f32>;

struct Params {
    rows: u32, dims: u32, queryCount: u32, base: u32,
    k: u32, pad0: u32, pad1: u32, pad2: u32,
}
@group(0) @binding(5) var<uniform> params: Params;

const SLOTS: u32 = 9u;
const FAR: f32 = 3.4028235e38;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let r = params.base + gid.x;
    if (r >= params.rows) { return; }

    var d: array<f32, 9>;
    var ix: array<u32, 9>;
    var keep: u32 = params.k + 1u;
    if (keep > SLOTS) { keep = SLOTS; }
    var filled: u32 = 0u;

    for (var q: u32 = 0u; q < params.queryCount; q = q + 1u) {
        var acc: f32 = 0.0;
        for (var c: u32 = 0u; c < params.dims; c = c + 1u) {
            let diff = data[c * params.rows + r] - queries[q * params.dims + c];
            acc = acc + diff * diff;
        }

        var pos: u32 = 0u;
        if (filled < keep) {
            pos = filled;
            filled = filled + 1u;
        } else if (acc < d[keep - 1u]) {
            pos = keep - 1u;
        } else {
            continue;
        }
        d[pos] = acc;
        ix[pos] = q;
        for (var p: u32 = pos; p > 0u; p = p - 1u) {
            if (d[p - 1u] <= d[p]) { break; }
            let td = d[p - 1u]; d[p - 1u] = d[p]; d[p] = td;
            let ti = ix[p - 1u]; ix[p - 1u] = ix[p]; ix[p] = ti;
        }
    }

    for (var j: u32 = 0u; j < params.k; j = j + 1u) {
        if (j < filled) {
            outDist[r * params.k + j] = d[j];
            outIdx[r * params.k + j] = ix[j];
        } else {
            outDist[r * params.k + j] = FAR;
            outIdx[r * params.k + j] = 0xffffffffu;
        }
    }
    if (filled > params.k) {
        outBoundary[r] = d[params.k];
    } else {
        outBoundary[r] = FAR;
    }
}
`

// NearestShortlist returns the k nearest query points per row in single
// precision, row-major, plus the distance of the best rejected candidate.
//
// Row-major matters: the host reads one row's whole shortlist together, and a
// candidate-major layout makes each of those reads stride across the entire
// array.
//
// The shortlist is a proposal, not an answer. Its purpose is to let the caller
// redo a handful of distances in float64 instead of all of them.
func NearestShortlist(ctx context.Context, columns []Column, queries [][]float32, k int) ([]uint32, []float32, []float32, Cost, error) {
	submitMu.Lock()
	defer submitMu.Unlock()

	if len(columns) == 0 || len(queries) == 0 {
		return nil, nil, nil, Cost{}, fmt.Errorf("wgpu: nearest shortlist needs columns and queries")
	}
	if k < 1 || k > MaxShortlist {
		return nil, nil, nil, Cost{}, fmt.Errorf("wgpu: shortlist width %d is outside 1..%d", k, MaxShortlist)
	}
	if k > len(queries) {
		return nil, nil, nil, Cost{}, fmt.Errorf("wgpu: shortlist width %d exceeds %d query points", k, len(queries))
	}
	rows := len(columns[0].Values)
	dims := len(columns)
	for _, column := range columns {
		if len(column.Values) != rows {
			return nil, nil, nil, Cost{}, fmt.Errorf("wgpu: columns have differing lengths")
		}
	}

	h, err := acquire()
	if err != nil {
		return nil, nil, nil, Cost{}, err
	}
	pipeline, bgLayout, err := h.shortlistPipeline()
	if err != nil {
		return nil, nil, nil, Cost{}, err
	}

	flat := make([]float32, 0, rows*dims)
	for _, column := range columns {
		flat = append(flat, column.Values...)
	}
	flatQueries := make([]float32, 0, len(queries)*dims)
	for _, query := range queries {
		if len(query) != dims {
			return nil, nil, nil, Cost{}, fmt.Errorf("wgpu: query has %d dimensions, expected %d", len(query), dims)
		}
		flatQueries = append(flatQueries, query...)
	}

	var cost Cost
	release := make([]interface{ Release() }, 0, 8)
	defer func() {
		for _, r := range release {
			r.Release()
		}
	}()
	mk := func(label string, size uint64, usage gowgpu.BufferUsage) (*gowgpu.Buffer, error) {
		b, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{Label: label, Size: size, Usage: usage})
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBufferTooLarge, err)
		}
		release = append(release, b)
		return b, nil
	}

	dataBytes := uint64(len(flat) * 4)
	queryBytes := uint64(len(flatQueries) * 4)
	listBytes := uint64(rows * k * 4)
	rowBytes := uint64(rows * 4)

	dataBuf, err := mk("accel-shortlist-data", dataBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, nil, nil, cost, err
	}
	queryBuf, err := mk("accel-shortlist-queries", queryBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, nil, nil, cost, err
	}
	distBuf, err := mk("accel-shortlist-dist", listBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, nil, nil, cost, err
	}
	idxBuf, err := mk("accel-shortlist-idx", listBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, nil, nil, cost, err
	}
	boundaryBuf, err := mk("accel-shortlist-boundary", rowBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, nil, nil, cost, err
	}
	// Distances, then indices, then boundaries, so readback is one map.
	staging, err := mk("accel-shortlist-staging", listBytes*2+rowBytes, gowgpu.BufferUsageCopyDst|gowgpu.BufferUsageMapRead)
	if err != nil {
		return nil, nil, nil, cost, err
	}

	transferStart := time.Now()
	if err := h.device.Queue().WriteBuffer(dataBuf, 0, float32Bytes(flat)); err != nil {
		return nil, nil, nil, cost, fmt.Errorf("wgpu: upload data: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(queryBuf, 0, float32Bytes(flatQueries)); err != nil {
		return nil, nil, nil, cost, fmt.Errorf("wgpu: upload queries: %w", err)
	}
	cost.BytesUploaded = dataBytes + queryBytes

	const perDispatch = maxWorkgroups * 64
	type slice struct {
		bindGroup *gowgpu.BindGroup
		groups    int
	}
	slices := make([]slice, 0, rows/perDispatch+1)
	for base := 0; base < rows; base += perDispatch {
		count := rows - base
		if count > perDispatch {
			count = perDispatch
		}
		// Each slice needs its own params buffer: WriteBuffer is a queue
		// operation and does not interleave with recorded commands, so reusing
		// one buffer would leave every dispatch reading the last base written.
		params, err := mk("accel-shortlist-params", 32, gowgpu.BufferUsageUniform|gowgpu.BufferUsageCopyDst)
		if err != nil {
			return nil, nil, nil, cost, err
		}
		paramBytes := make([]byte, 32)
		binary.LittleEndian.PutUint32(paramBytes[0:], uint32(rows))
		binary.LittleEndian.PutUint32(paramBytes[4:], uint32(dims))
		binary.LittleEndian.PutUint32(paramBytes[8:], uint32(len(queries)))
		binary.LittleEndian.PutUint32(paramBytes[12:], uint32(base))
		binary.LittleEndian.PutUint32(paramBytes[16:], uint32(k))
		if err := h.device.Queue().WriteBuffer(params, 0, paramBytes); err != nil {
			return nil, nil, nil, cost, fmt.Errorf("wgpu: upload params: %w", err)
		}
		bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
			Label: "accel-shortlist-bg", Layout: bgLayout,
			Entries: []gowgpu.BindGroupEntry{
				{Binding: 0, Buffer: dataBuf, Size: dataBytes},
				{Binding: 1, Buffer: queryBuf, Size: queryBytes},
				{Binding: 2, Buffer: distBuf, Size: listBytes},
				{Binding: 3, Buffer: idxBuf, Size: listBytes},
				{Binding: 4, Buffer: boundaryBuf, Size: rowBytes},
				{Binding: 5, Buffer: params, Size: 32},
			},
		})
		if err != nil {
			return nil, nil, nil, cost, fmt.Errorf("wgpu: bind group: %w", err)
		}
		release = append(release, bindGroup)
		slices = append(slices, slice{bindGroup: bindGroup, groups: (count + 63) / 64})
	}
	cost.Transfer = time.Since(transferStart)

	dispatchStart := time.Now()
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, nil, nil, cost, fmt.Errorf("wgpu: command encoder: %w", err)
	}
	for _, sl := range slices {
		pass, err := encoder.BeginComputePass(nil)
		if err != nil {
			return nil, nil, nil, cost, fmt.Errorf("wgpu: compute pass: %w", err)
		}
		pass.SetPipeline(pipeline)
		pass.SetBindGroup(0, sl.bindGroup, nil)
		pass.Dispatch(uint32(sl.groups), 1, 1)
		if err := pass.End(); err != nil {
			return nil, nil, nil, cost, fmt.Errorf("wgpu: end compute pass: %w", err)
		}
	}
	encoder.CopyBufferToBuffer(distBuf, 0, staging, 0, listBytes)
	encoder.CopyBufferToBuffer(idxBuf, 0, staging, listBytes, listBytes)
	encoder.CopyBufferToBuffer(boundaryBuf, 0, staging, listBytes*2, rowBytes)
	commands, err := encoder.Finish()
	if err != nil {
		return nil, nil, nil, cost, fmt.Errorf("wgpu: finish encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, nil, nil, cost, fmt.Errorf("wgpu: submit: %w", err)
	}
	cost.Dispatch = time.Since(dispatchStart)

	total := listBytes*2 + rowBytes
	readbackStart := time.Now()
	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, total); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(mapCtx.Err(), context.DeadlineExceeded) {
			return nil, nil, nil, cost, fmt.Errorf("%w: %v", ErrReadbackTimeout, err)
		}
		return nil, nil, nil, cost, fmt.Errorf("wgpu: map staging buffer: %w", err)
	}
	mapped, err := staging.MappedRange(0, total)
	if err != nil {
		_ = staging.Unmap()
		return nil, nil, nil, cost, fmt.Errorf("wgpu: mapped range: %w", err)
	}
	raw := mapped.Bytes()
	distances := make([]float32, rows*k)
	indices := make([]uint32, rows*k)
	boundary := make([]float32, rows)
	for i := range distances {
		distances[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		indices[i] = binary.LittleEndian.Uint32(raw[int(listBytes)+i*4:])
	}
	for i := range boundary {
		boundary[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[int(listBytes)*2+i*4:]))
	}
	if err := staging.Unmap(); err != nil {
		return nil, nil, nil, cost, fmt.Errorf("wgpu: unmap staging buffer: %w", err)
	}
	cost.Readback = time.Since(readbackStart)

	return indices, distances, boundary, cost, nil
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

	shortlistOnce     sync.Once
	shortlistBGLayout *gowgpu.BindGroupLayout
	shortlistPipe     *gowgpu.ComputePipeline
	shortlistErr      error
}

// shortlistPipeline compiles the shortlist kernel once per process, the same way
// the other pipelines are cached.
func (h *handle) shortlistPipeline() (*gowgpu.ComputePipeline, *gowgpu.BindGroupLayout, error) {
	h.shortlistOnce.Do(func() {
		shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
			Label: "accel-shortlist", WGSL: nearestShortlistWGSL,
		})
		if err != nil {
			h.shortlistErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
			return
		}
		ro := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}
		rw := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}
		uni := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}
		bgLayout, err := h.device.CreateBindGroupLayout(&gowgpu.BindGroupLayoutDescriptor{
			Label: "accel-shortlist-bgl",
			Entries: []gowgpu.BindGroupLayoutEntry{
				{Binding: 0, Visibility: gowgpu.ShaderStageCompute, Buffer: ro},
				{Binding: 1, Visibility: gowgpu.ShaderStageCompute, Buffer: ro},
				{Binding: 2, Visibility: gowgpu.ShaderStageCompute, Buffer: rw},
				{Binding: 3, Visibility: gowgpu.ShaderStageCompute, Buffer: rw},
				{Binding: 4, Visibility: gowgpu.ShaderStageCompute, Buffer: rw},
				{Binding: 5, Visibility: gowgpu.ShaderStageCompute, Buffer: uni},
			},
		})
		if err != nil {
			h.shortlistErr = fmt.Errorf("wgpu: shortlist bind group layout: %w", err)
			return
		}
		plLayout, err := h.device.CreatePipelineLayout(&gowgpu.PipelineLayoutDescriptor{
			Label: "accel-shortlist-pl", BindGroupLayouts: []*gowgpu.BindGroupLayout{bgLayout},
		})
		if err != nil {
			h.shortlistErr = fmt.Errorf("wgpu: shortlist pipeline layout: %w", err)
			return
		}
		pipeline, err := h.device.CreateComputePipeline(&gowgpu.ComputePipelineDescriptor{
			Label: "accel-shortlist-pipeline", Layout: plLayout, Module: shader, EntryPoint: "main",
		})
		if err != nil {
			h.shortlistErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
			return
		}
		h.shortlistBGLayout = bgLayout
		h.shortlistPipe = pipeline
	})
	return h.shortlistPipe, h.shortlistBGLayout, h.shortlistErr
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

func float32Bytes(values []float32) []byte {
	if len(values) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&values[0])), len(values)*4)
}
