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

// segment is one dispatch: a chunk of one column small enough for the device's
// limits. Columns larger than a chunk become several segments.
type segment struct {
	column string
	values []float32
	groups int
	offset uint64 // byte offset of this segment's partials within the staging buffer
}

// SumColumns reduces every column on the device and returns a total per column.
// Nulls are expected to have been replaced by the additive identity by the
// caller. Cost describes the whole call.
//
// Every segment is encoded into one command buffer and every partial lands in
// one staging buffer, so the call waits on a single map regardless of how many
// columns it was given. Readback is the dominant device cost, so a wait per
// column would make a wide table pay for its width.
func SumColumns(ctx context.Context, columns []Column) (map[string]float64, Cost, error) {
	submitMu.Lock()
	defer submitMu.Unlock()

	h, err := acquire()
	if err != nil {
		return nil, Cost{}, err
	}
	pipeline, bgLayout, err := h.computePipeline()
	if err != nil {
		return nil, Cost{}, err
	}
	chunkSize := h.maxElementsPerChunk()
	if chunkSize <= 0 {
		return nil, Cost{}, fmt.Errorf("%w: device reports no bindable storage", ErrBufferTooLarge)
	}

	sums := make(map[string]float64, len(columns))
	segments := make([]segment, 0, len(columns))
	partialBytes := uint64(0)
	for _, column := range columns {
		sums[column.Name] = 0
		for start := 0; start < len(column.Values); start += chunkSize {
			end := start + chunkSize
			if end > len(column.Values) {
				end = len(column.Values)
			}
			values := column.Values[start:end]
			groups := (len(values) + elemsPerGroup - 1) / elemsPerGroup
			segments = append(segments, segment{
				column: column.Name,
				values: values,
				groups: groups,
				offset: partialBytes,
			})
			partialBytes += uint64(groups * 4)
		}
	}
	if len(segments) == 0 {
		return sums, Cost{}, nil
	}

	var cost Cost
	release := make([]interface{ Release() }, 0, len(segments)*4+1)
	defer func() {
		for _, r := range release {
			r.Release()
		}
	}()

	staging, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-staging", Size: partialBytes,
		Usage: gowgpu.BufferUsageCopyDst | gowgpu.BufferUsageMapRead,
	})
	if err != nil {
		return nil, cost, fmt.Errorf("wgpu: staging buffer: %w", err)
	}
	release = append(release, staging)

	bindGroups := make([]*gowgpu.BindGroup, len(segments))
	partials := make([]*gowgpu.Buffer, len(segments))

	transferStart := time.Now()
	for i, seg := range segments {
		inputBytes := uint64(len(seg.values) * 4)
		input, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
			Label: "accel-input", Size: inputBytes,
			Usage: gowgpu.BufferUsageStorage | gowgpu.BufferUsageCopyDst,
		})
		if err != nil {
			return nil, cost, fmt.Errorf("%w: %v", ErrBufferTooLarge, err)
		}
		release = append(release, input)

		partial, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
			Label: "accel-partials", Size: uint64(seg.groups * 4),
			Usage: gowgpu.BufferUsageStorage | gowgpu.BufferUsageCopySrc,
		})
		if err != nil {
			return nil, cost, fmt.Errorf("wgpu: partial buffer: %w", err)
		}
		release = append(release, partial)
		partials[i] = partial

		params, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
			Label: "accel-params", Size: 4,
			Usage: gowgpu.BufferUsageUniform | gowgpu.BufferUsageCopyDst,
		})
		if err != nil {
			return nil, cost, fmt.Errorf("wgpu: params buffer: %w", err)
		}
		release = append(release, params)

		countBuf := make([]byte, 4)
		binary.LittleEndian.PutUint32(countBuf, uint32(len(seg.values)))
		if err := h.device.Queue().WriteBuffer(input, 0, float32Bytes(seg.values)); err != nil {
			return nil, cost, fmt.Errorf("wgpu: upload column: %w", err)
		}
		if err := h.device.Queue().WriteBuffer(params, 0, countBuf); err != nil {
			return nil, cost, fmt.Errorf("wgpu: upload params: %w", err)
		}
		cost.BytesUploaded += inputBytes

		bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
			Label: "accel-sum-bg", Layout: bgLayout,
			Entries: []gowgpu.BindGroupEntry{
				{Binding: 0, Buffer: input, Size: inputBytes},
				{Binding: 1, Buffer: partial, Size: uint64(seg.groups * 4)},
				{Binding: 2, Buffer: params, Size: 4},
			},
		})
		if err != nil {
			return nil, cost, fmt.Errorf("wgpu: bind group: %w", err)
		}
		release = append(release, bindGroup)
		bindGroups[i] = bindGroup
	}
	cost.Transfer = time.Since(transferStart)

	dispatchStart := time.Now()
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, cost, fmt.Errorf("wgpu: command encoder: %w", err)
	}
	for i, seg := range segments {
		pass, err := encoder.BeginComputePass(nil)
		if err != nil {
			return nil, cost, fmt.Errorf("wgpu: compute pass: %w", err)
		}
		pass.SetPipeline(pipeline)
		pass.SetBindGroup(0, bindGroups[i], nil)
		pass.Dispatch(uint32(seg.groups), 1, 1)
		if err := pass.End(); err != nil {
			return nil, cost, fmt.Errorf("wgpu: end compute pass: %w", err)
		}
	}
	for i, seg := range segments {
		encoder.CopyBufferToBuffer(partials[i], 0, staging, seg.offset, uint64(seg.groups*4))
	}
	commands, err := encoder.Finish()
	if err != nil {
		return nil, cost, fmt.Errorf("wgpu: finish encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, cost, fmt.Errorf("wgpu: submit: %w", err)
	}
	cost.Dispatch = time.Since(dispatchStart)

	readbackStart := time.Now()
	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, partialBytes); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(mapCtx.Err(), context.DeadlineExceeded) {
			return nil, cost, fmt.Errorf("%w: %v", ErrReadbackTimeout, err)
		}
		return nil, cost, fmt.Errorf("wgpu: map staging buffer: %w", err)
	}
	mapped, err := staging.MappedRange(0, partialBytes)
	if err != nil {
		_ = staging.Unmap()
		return nil, cost, fmt.Errorf("wgpu: mapped range: %w", err)
	}
	raw := mapped.Bytes()
	// Fold the per-workgroup partials in float64: only the reduction inside a
	// workgroup runs at single precision.
	for _, seg := range segments {
		for i := 0; i < seg.groups; i++ {
			at := seg.offset + uint64(i*4)
			sums[seg.column] += float64(math.Float32frombits(binary.LittleEndian.Uint32(raw[at:])))
		}
	}
	if err := staging.Unmap(); err != nil {
		return nil, cost, fmt.Errorf("wgpu: unmap staging buffer: %w", err)
	}
	cost.Readback = time.Since(readbackStart)

	return sums, cost, nil
}

// distanceWGSL computes one squared distance per (query, row) pair. One thread
// owns one output and loops the dimensions sequentially, so the operation order
// matches the CPU reference exactly — there is no cross-thread reduction and
// therefore no associativity question. The accumulation is left as
// `acc + diff*diff` because Metal and Go/arm64 both contract it into a fused
// multiply-add in the same way; forcing a rounded product here would break bit
// parity rather than protect it.
const distanceWGSL = `
@group(0) @binding(0) var<storage, read> data: array<f32>;
@group(0) @binding(1) var<storage, read> queries: array<f32>;
@group(0) @binding(2) var<storage, read_write> out: array<f32>;

struct Params { rows: u32, dims: u32, queryCount: u32, base: u32, }
@group(0) @binding(3) var<uniform> params: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let total = params.rows * params.queryCount;
    // base lets one dispatch cover a slice of the output, so an output larger
    // than the 65535 workgroup limit is split rather than refused.
    let idx = params.base + gid.x;
    if (idx >= total) { return; }
    let q = idx / params.rows;
    let r = idx % params.rows;

    var acc: f32 = 0.0;
    for (var c: u32 = 0u; c < params.dims; c = c + 1u) {
        let diff = data[c * params.rows + r] - queries[q * params.dims + c];
        acc = acc + diff * diff;
    }
    out[idx] = acc;
}
`

// SquaredDistances computes the squared Euclidean distance from every row to
// every query point, query-major. columns is column-major and every column must
// have the same length; queries carry one value per column.
func SquaredDistances(ctx context.Context, columns []Column, queries [][]float32) ([]float32, Cost, error) {
	submitMu.Lock()
	defer submitMu.Unlock()

	if len(columns) == 0 || len(queries) == 0 {
		return nil, Cost{}, fmt.Errorf("wgpu: squared distance needs columns and queries")
	}
	rows := len(columns[0].Values)
	dims := len(columns)
	for _, column := range columns {
		if len(column.Values) != rows {
			return nil, Cost{}, fmt.Errorf("wgpu: columns have differing lengths")
		}
	}

	h, err := acquire()
	if err != nil {
		return nil, Cost{}, err
	}
	pipeline, bgLayout, err := h.distancePipeline()
	if err != nil {
		return nil, Cost{}, err
	}

	// Column-major, matching the shader's indexing.
	flat := make([]float32, 0, rows*dims)
	for _, column := range columns {
		flat = append(flat, column.Values...)
	}
	flatQueries := make([]float32, 0, len(queries)*dims)
	for _, query := range queries {
		if len(query) != dims {
			return nil, Cost{}, fmt.Errorf("wgpu: query has %d dimensions, expected %d", len(query), dims)
		}
		flatQueries = append(flatQueries, query...)
	}
	outCount := rows * len(queries)

	var cost Cost
	release := make([]interface{ Release() }, 0, 5)
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
	outBytes := uint64(outCount * 4)

	dataBuf, err := mk("accel-distance-data", dataBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, cost, err
	}
	queryBuf, err := mk("accel-distance-queries", queryBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, cost, err
	}
	outBuf, err := mk("accel-distance-out", outBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, cost, err
	}
	staging, err := mk("accel-distance-staging", outBytes, gowgpu.BufferUsageCopyDst|gowgpu.BufferUsageMapRead)
	if err != nil {
		return nil, cost, err
	}
	transferStart := time.Now()
	if err := h.device.Queue().WriteBuffer(dataBuf, 0, float32Bytes(flat)); err != nil {
		return nil, cost, fmt.Errorf("wgpu: upload data: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(queryBuf, 0, float32Bytes(flatQueries)); err != nil {
		return nil, cost, fmt.Errorf("wgpu: upload queries: %w", err)
	}
	cost.BytesUploaded = dataBytes + queryBytes

	// The output can need more workgroups than one dispatch may carry, so it is
	// split into slices that each fit. Each slice gets its own params buffer and
	// bind group: WriteBuffer is a queue operation and does not interleave with
	// recorded commands, so reusing one params buffer would give every dispatch
	// whichever base was written last.
	const perDispatch = maxWorkgroups * 64
	type slice struct {
		bindGroup *gowgpu.BindGroup
		groups    int
	}
	slices := make([]slice, 0, outCount/perDispatch+1)
	for base := 0; base < outCount; base += perDispatch {
		count := outCount - base
		if count > perDispatch {
			count = perDispatch
		}
		params, err := mk("accel-distance-params", 16, gowgpu.BufferUsageUniform|gowgpu.BufferUsageCopyDst)
		if err != nil {
			return nil, cost, err
		}
		paramBytes := make([]byte, 16)
		binary.LittleEndian.PutUint32(paramBytes[0:], uint32(rows))
		binary.LittleEndian.PutUint32(paramBytes[4:], uint32(dims))
		binary.LittleEndian.PutUint32(paramBytes[8:], uint32(len(queries)))
		binary.LittleEndian.PutUint32(paramBytes[12:], uint32(base))
		if err := h.device.Queue().WriteBuffer(params, 0, paramBytes); err != nil {
			return nil, cost, fmt.Errorf("wgpu: upload params: %w", err)
		}
		bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
			Label: "accel-distance-bg", Layout: bgLayout,
			Entries: []gowgpu.BindGroupEntry{
				{Binding: 0, Buffer: dataBuf, Size: dataBytes},
				{Binding: 1, Buffer: queryBuf, Size: queryBytes},
				{Binding: 2, Buffer: outBuf, Size: outBytes},
				{Binding: 3, Buffer: params, Size: 16},
			},
		})
		if err != nil {
			return nil, cost, fmt.Errorf("wgpu: bind group: %w", err)
		}
		release = append(release, bindGroup)
		slices = append(slices, slice{bindGroup: bindGroup, groups: (count + 63) / 64})
	}
	cost.Transfer = time.Since(transferStart)

	dispatchStart := time.Now()
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, cost, fmt.Errorf("wgpu: command encoder: %w", err)
	}
	for _, sl := range slices {
		pass, err := encoder.BeginComputePass(nil)
		if err != nil {
			return nil, cost, fmt.Errorf("wgpu: compute pass: %w", err)
		}
		pass.SetPipeline(pipeline)
		pass.SetBindGroup(0, sl.bindGroup, nil)
		pass.Dispatch(uint32(sl.groups), 1, 1)
		if err := pass.End(); err != nil {
			return nil, cost, fmt.Errorf("wgpu: end compute pass: %w", err)
		}
	}
	encoder.CopyBufferToBuffer(outBuf, 0, staging, 0, outBytes)
	commands, err := encoder.Finish()
	if err != nil {
		return nil, cost, fmt.Errorf("wgpu: finish encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, cost, fmt.Errorf("wgpu: submit: %w", err)
	}
	cost.Dispatch = time.Since(dispatchStart)

	readbackStart := time.Now()
	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, outBytes); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(mapCtx.Err(), context.DeadlineExceeded) {
			return nil, cost, fmt.Errorf("%w: %v", ErrReadbackTimeout, err)
		}
		return nil, cost, fmt.Errorf("wgpu: map staging buffer: %w", err)
	}
	mapped, err := staging.MappedRange(0, outBytes)
	if err != nil {
		_ = staging.Unmap()
		return nil, cost, fmt.Errorf("wgpu: mapped range: %w", err)
	}
	raw := mapped.Bytes()
	out := make([]float32, outCount)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}
	if err := staging.Unmap(); err != nil {
		return nil, cost, fmt.Errorf("wgpu: unmap staging buffer: %w", err)
	}
	cost.Readback = time.Since(readbackStart)

	return out, cost, nil
}

// nearestWGSL takes the minimum on the device, so the host reads one pair per
// row instead of the whole rows-by-queries matrix. One thread owns one row and
// walks every query, which keeps the operation order identical to the CPU
// reference. The running minimum is seeded from query zero rather than an
// infinity sentinel, and the comparison is strict so ties keep the lowest index.
const nearestWGSL = `
@group(0) @binding(0) var<storage, read> data: array<f32>;
@group(0) @binding(1) var<storage, read> queries: array<f32>;
@group(0) @binding(2) var<storage, read_write> outDist: array<f32>;
@group(0) @binding(3) var<storage, read_write> outIdx: array<u32>;

struct Params { rows: u32, dims: u32, queryCount: u32, base: u32, }
@group(0) @binding(4) var<uniform> params: Params;

@compute @workgroup_size(64)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let r = params.base + gid.x;
    if (r >= params.rows) { return; }

    var best: f32 = 0.0;
    for (var c: u32 = 0u; c < params.dims; c = c + 1u) {
        let diff = data[c * params.rows + r] - queries[c];
        best = best + diff * diff;
    }
    var bestIdx: u32 = 0u;

    for (var q: u32 = 1u; q < params.queryCount; q = q + 1u) {
        var acc: f32 = 0.0;
        for (var c: u32 = 0u; c < params.dims; c = c + 1u) {
            let diff = data[c * params.rows + r] - queries[q * params.dims + c];
            acc = acc + diff * diff;
        }
        if (acc < best) {
            best = acc;
            bestIdx = q;
        }
    }
    outDist[r] = best;
    outIdx[r] = bestIdx;
}
`

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

// NearestQuery reports the closest query point per row and its squared distance.
func NearestQuery(ctx context.Context, columns []Column, queries [][]float32) ([]uint32, []float32, Cost, error) {
	submitMu.Lock()
	defer submitMu.Unlock()

	if len(columns) == 0 || len(queries) == 0 {
		return nil, nil, Cost{}, fmt.Errorf("wgpu: nearest query needs columns and queries")
	}
	rows := len(columns[0].Values)
	dims := len(columns)
	for _, column := range columns {
		if len(column.Values) != rows {
			return nil, nil, Cost{}, fmt.Errorf("wgpu: columns have differing lengths")
		}
	}

	h, err := acquire()
	if err != nil {
		return nil, nil, Cost{}, err
	}
	pipeline, bgLayout, err := h.nearestPipeline()
	if err != nil {
		return nil, nil, Cost{}, err
	}

	flat := make([]float32, 0, rows*dims)
	for _, column := range columns {
		flat = append(flat, column.Values...)
	}
	flatQueries := make([]float32, 0, len(queries)*dims)
	for _, query := range queries {
		if len(query) != dims {
			return nil, nil, Cost{}, fmt.Errorf("wgpu: query has %d dimensions, expected %d", len(query), dims)
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
	outBytes := uint64(rows * 4)

	dataBuf, err := mk("accel-nearest-data", dataBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, nil, cost, err
	}
	queryBuf, err := mk("accel-nearest-queries", queryBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, nil, cost, err
	}
	distBuf, err := mk("accel-nearest-dist", outBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, nil, cost, err
	}
	idxBuf, err := mk("accel-nearest-idx", outBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, nil, cost, err
	}
	// One staging buffer holding distances then indices, so there is one map.
	staging, err := mk("accel-nearest-staging", outBytes*2, gowgpu.BufferUsageCopyDst|gowgpu.BufferUsageMapRead)
	if err != nil {
		return nil, nil, cost, err
	}

	transferStart := time.Now()
	if err := h.device.Queue().WriteBuffer(dataBuf, 0, float32Bytes(flat)); err != nil {
		return nil, nil, cost, fmt.Errorf("wgpu: upload data: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(queryBuf, 0, float32Bytes(flatQueries)); err != nil {
		return nil, nil, cost, fmt.Errorf("wgpu: upload queries: %w", err)
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
		params, err := mk("accel-nearest-params", 16, gowgpu.BufferUsageUniform|gowgpu.BufferUsageCopyDst)
		if err != nil {
			return nil, nil, cost, err
		}
		paramBytes := make([]byte, 16)
		binary.LittleEndian.PutUint32(paramBytes[0:], uint32(rows))
		binary.LittleEndian.PutUint32(paramBytes[4:], uint32(dims))
		binary.LittleEndian.PutUint32(paramBytes[8:], uint32(len(queries)))
		binary.LittleEndian.PutUint32(paramBytes[12:], uint32(base))
		if err := h.device.Queue().WriteBuffer(params, 0, paramBytes); err != nil {
			return nil, nil, cost, fmt.Errorf("wgpu: upload params: %w", err)
		}
		bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
			Label: "accel-nearest-bg", Layout: bgLayout,
			Entries: []gowgpu.BindGroupEntry{
				{Binding: 0, Buffer: dataBuf, Size: dataBytes},
				{Binding: 1, Buffer: queryBuf, Size: queryBytes},
				{Binding: 2, Buffer: distBuf, Size: outBytes},
				{Binding: 3, Buffer: idxBuf, Size: outBytes},
				{Binding: 4, Buffer: params, Size: 16},
			},
		})
		if err != nil {
			return nil, nil, cost, fmt.Errorf("wgpu: bind group: %w", err)
		}
		release = append(release, bindGroup)
		slices = append(slices, slice{bindGroup: bindGroup, groups: (count + 63) / 64})
	}
	cost.Transfer = time.Since(transferStart)

	dispatchStart := time.Now()
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, nil, cost, fmt.Errorf("wgpu: command encoder: %w", err)
	}
	for _, sl := range slices {
		pass, err := encoder.BeginComputePass(nil)
		if err != nil {
			return nil, nil, cost, fmt.Errorf("wgpu: compute pass: %w", err)
		}
		pass.SetPipeline(pipeline)
		pass.SetBindGroup(0, sl.bindGroup, nil)
		pass.Dispatch(uint32(sl.groups), 1, 1)
		if err := pass.End(); err != nil {
			return nil, nil, cost, fmt.Errorf("wgpu: end compute pass: %w", err)
		}
	}
	encoder.CopyBufferToBuffer(distBuf, 0, staging, 0, outBytes)
	encoder.CopyBufferToBuffer(idxBuf, 0, staging, outBytes, outBytes)
	commands, err := encoder.Finish()
	if err != nil {
		return nil, nil, cost, fmt.Errorf("wgpu: finish encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, nil, cost, fmt.Errorf("wgpu: submit: %w", err)
	}
	cost.Dispatch = time.Since(dispatchStart)

	readbackStart := time.Now()
	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, outBytes*2); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(mapCtx.Err(), context.DeadlineExceeded) {
			return nil, nil, cost, fmt.Errorf("%w: %v", ErrReadbackTimeout, err)
		}
		return nil, nil, cost, fmt.Errorf("wgpu: map staging buffer: %w", err)
	}
	mapped, err := staging.MappedRange(0, outBytes*2)
	if err != nil {
		_ = staging.Unmap()
		return nil, nil, cost, fmt.Errorf("wgpu: mapped range: %w", err)
	}
	raw := mapped.Bytes()
	distances := make([]float32, rows)
	indices := make([]uint32, rows)
	for i := 0; i < rows; i++ {
		distances[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		indices[i] = binary.LittleEndian.Uint32(raw[int(outBytes)+i*4:])
	}
	if err := staging.Unmap(); err != nil {
		return nil, nil, cost, fmt.Errorf("wgpu: unmap staging buffer: %w", err)
	}
	cost.Readback = time.Since(readbackStart)

	return indices, distances, cost, nil
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

	distanceOnce     sync.Once
	distanceBGLayout *gowgpu.BindGroupLayout
	distancePipe     *gowgpu.ComputePipeline
	distanceErr      error

	nearestOnce     sync.Once
	nearestBGLayout *gowgpu.BindGroupLayout
	nearestPipe     *gowgpu.ComputePipeline
	nearestErr      error

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

func (h *handle) nearestPipeline() (*gowgpu.ComputePipeline, *gowgpu.BindGroupLayout, error) {
	h.nearestOnce.Do(func() {
		shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
			Label: "accel-nearest", WGSL: nearestWGSL,
		})
		if err != nil {
			h.nearestErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
			return
		}
		ro := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}
		rw := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}
		uni := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}
		bgLayout, err := h.device.CreateBindGroupLayout(&gowgpu.BindGroupLayoutDescriptor{
			Label: "accel-nearest-bgl",
			Entries: []gowgpu.BindGroupLayoutEntry{
				{Binding: 0, Visibility: gowgpu.ShaderStageCompute, Buffer: ro},
				{Binding: 1, Visibility: gowgpu.ShaderStageCompute, Buffer: ro},
				{Binding: 2, Visibility: gowgpu.ShaderStageCompute, Buffer: rw},
				{Binding: 3, Visibility: gowgpu.ShaderStageCompute, Buffer: rw},
				{Binding: 4, Visibility: gowgpu.ShaderStageCompute, Buffer: uni},
			},
		})
		if err != nil {
			h.nearestErr = fmt.Errorf("wgpu: nearest bind group layout: %w", err)
			return
		}
		plLayout, err := h.device.CreatePipelineLayout(&gowgpu.PipelineLayoutDescriptor{
			Label: "accel-nearest-pl", BindGroupLayouts: []*gowgpu.BindGroupLayout{bgLayout},
		})
		if err != nil {
			h.nearestErr = fmt.Errorf("wgpu: nearest pipeline layout: %w", err)
			return
		}
		pipeline, err := h.device.CreateComputePipeline(&gowgpu.ComputePipelineDescriptor{
			Label: "accel-nearest-pipeline", Layout: plLayout, Module: shader, EntryPoint: "main",
		})
		if err != nil {
			h.nearestErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
			return
		}
		h.nearestBGLayout = bgLayout
		h.nearestPipe = pipeline
	})
	return h.nearestPipe, h.nearestBGLayout, h.nearestErr
}

// distancePipeline compiles the distance kernel once per process, the same way
// the sum pipeline is cached.
func (h *handle) distancePipeline() (*gowgpu.ComputePipeline, *gowgpu.BindGroupLayout, error) {
	h.distanceOnce.Do(func() {
		shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
			Label: "accel-distance", WGSL: distanceWGSL,
		})
		if err != nil {
			h.distanceErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
			return
		}
		ro := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}
		rw := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}
		uni := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}
		bgLayout, err := h.device.CreateBindGroupLayout(&gowgpu.BindGroupLayoutDescriptor{
			Label: "accel-distance-bgl",
			Entries: []gowgpu.BindGroupLayoutEntry{
				{Binding: 0, Visibility: gowgpu.ShaderStageCompute, Buffer: ro},
				{Binding: 1, Visibility: gowgpu.ShaderStageCompute, Buffer: ro},
				{Binding: 2, Visibility: gowgpu.ShaderStageCompute, Buffer: rw},
				{Binding: 3, Visibility: gowgpu.ShaderStageCompute, Buffer: uni},
			},
		})
		if err != nil {
			h.distanceErr = fmt.Errorf("wgpu: distance bind group layout: %w", err)
			return
		}
		plLayout, err := h.device.CreatePipelineLayout(&gowgpu.PipelineLayoutDescriptor{
			Label: "accel-distance-pl", BindGroupLayouts: []*gowgpu.BindGroupLayout{bgLayout},
		})
		if err != nil {
			h.distanceErr = fmt.Errorf("wgpu: distance pipeline layout: %w", err)
			return
		}
		pipeline, err := h.device.CreateComputePipeline(&gowgpu.ComputePipelineDescriptor{
			Label: "accel-distance-pipeline", Layout: plLayout, Module: shader, EntryPoint: "main",
		})
		if err != nil {
			h.distanceErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
			return
		}
		h.distanceBGLayout = bgLayout
		h.distancePipe = pipeline
	})
	return h.distancePipe, h.distanceBGLayout, h.distanceErr
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
