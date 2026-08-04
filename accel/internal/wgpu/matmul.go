package wgpu

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/gogpu/gputypes"
	gowgpu "github.com/gogpu/wgpu"
)

// matmulWGSL keeps one output's accumulation serial along k. The order is
// deliberate: the CPU reference and the device then produce bit-identical
// float32 results on the platforms where this path has been measured.
const matmulWGSL = `
@group(0) @binding(0) var<storage, read> a: array<f32>;
@group(0) @binding(1) var<storage, read> b: array<f32>;
@group(0) @binding(2) var<storage, read_write> c: array<f32>;

struct Params {
    m: u32,
    k: u32,
    n: u32,
    pad: u32,
}
@group(0) @binding(3) var<uniform> params: Params;

var<workgroup> tileA: array<f32, 256>;
var<workgroup> tileB: array<f32, 256>;

@compute @workgroup_size(16, 16, 1)
fn main(
	@builtin(workgroup_id) workgroup_id: vec3<u32>,
	@builtin(local_invocation_id) local_id: vec3<u32>
) {
    let row = workgroup_id.y * 16u + local_id.y;
    let column = workgroup_id.x * 16u + local_id.x;
    var acc: f32 = 0.0;
    let tileCount = (params.k + 15u) / 16u;

    for (var tile: u32 = 0u; tile < tileCount; tile = tile + 1u) {
        let aColumn = tile * 16u + local_id.x;
        let bRow = tile * 16u + local_id.y;
        let tileIndex = local_id.y * 16u + local_id.x;

        if (row < params.m && aColumn < params.k) {
            tileA[tileIndex] = a[row * params.k + aColumn];
        } else {
            tileA[tileIndex] = 0.0;
        }
        if (bRow < params.k && column < params.n) {
            tileB[tileIndex] = b[bRow * params.n + column];
        } else {
            tileB[tileIndex] = 0.0;
        }
        workgroupBarrier();

        for (var inner: u32 = 0u; inner < 16u; inner = inner + 1u) {
            acc = acc + tileA[local_id.y * 16u + inner] * tileB[inner * 16u + local_id.x];
        }
        workgroupBarrier();
    }

    if (row < params.m && column < params.n) {
        c[row * params.n + column] = acc;
    }
}
`

// MatMul computes one 2-D float32 matrix product on the discovered WebGPU
// device. The caller owns the CPU fallback and decides whether the workload is
// profitable; this function reports every device failure as an error.
func MatMul(ctx context.Context, a, b []float32, m, k, n int) ([]float32, Cost, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if m <= 0 || k <= 0 || n <= 0 {
		return nil, Cost{}, fmt.Errorf("wgpu: matmul dimensions must be positive, got [%d,%d]x[%d,%d]", m, k, k, n)
	}
	if len(a) != m*k || len(b) != k*n {
		return nil, Cost{}, fmt.Errorf("wgpu: matmul input lengths do not match [%d,%d]x[%d,%d]", m, k, k, n)
	}

	submitMu.Lock()
	defer submitMu.Unlock()

	h, err := acquire()
	if err != nil {
		return nil, Cost{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	pipeline, layout, err := h.matmulPipeline()
	if err != nil {
		return nil, Cost{}, err
	}

	aBytes := uint64(len(a) * 4)
	bBytes := uint64(len(b) * 4)
	cBytes := uint64(m * n * 4)
	release := make([]interface{ Release() }, 0, 6)
	defer func() {
		for _, resource := range release {
			resource.Release()
		}
	}()
	makeBuffer := func(label string, size uint64, usage gowgpu.BufferUsage) (*gowgpu.Buffer, error) {
		buffer, bufferErr := h.device.CreateBuffer(&gowgpu.BufferDescriptor{Label: label, Size: size, Usage: usage})
		if bufferErr != nil {
			return nil, fmt.Errorf("%w: create %s buffer: %v", ErrBufferTooLarge, label, bufferErr)
		}
		release = append(release, buffer)
		return buffer, nil
	}

	aBuffer, err := makeBuffer("accel-matmul-a", aBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, Cost{}, err
	}
	bBuffer, err := makeBuffer("accel-matmul-b", bBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, Cost{}, err
	}
	cBuffer, err := makeBuffer("accel-matmul-c", cBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, Cost{}, err
	}
	staging, err := makeBuffer("accel-matmul-staging", cBytes, gowgpu.BufferUsageCopyDst|gowgpu.BufferUsageMapRead)
	if err != nil {
		return nil, Cost{}, err
	}
	params, err := makeBuffer("accel-matmul-params", 16, gowgpu.BufferUsageUniform|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, Cost{}, err
	}

	paramsBytes := make([]byte, 16)
	binary.LittleEndian.PutUint32(paramsBytes[0:], uint32(m))
	binary.LittleEndian.PutUint32(paramsBytes[4:], uint32(k))
	binary.LittleEndian.PutUint32(paramsBytes[8:], uint32(n))
	var cost Cost
	transferStart := time.Now()
	if err := h.device.Queue().WriteBuffer(aBuffer, 0, float32Bytes(a)); err != nil {
		return nil, cost, fmt.Errorf("wgpu: upload matmul A: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(bBuffer, 0, float32Bytes(b)); err != nil {
		return nil, cost, fmt.Errorf("wgpu: upload matmul B: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(params, 0, paramsBytes); err != nil {
		return nil, cost, fmt.Errorf("wgpu: upload matmul parameters: %w", err)
	}
	cost.Transfer = time.Since(transferStart)
	cost.BytesUploaded = aBytes + bBytes

	bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
		Label: "accel-matmul-bg", Layout: layout,
		Entries: []gowgpu.BindGroupEntry{
			{Binding: 0, Buffer: aBuffer, Size: aBytes},
			{Binding: 1, Buffer: bBuffer, Size: bBytes},
			{Binding: 2, Buffer: cBuffer, Size: cBytes},
			{Binding: 3, Buffer: params, Size: 16},
		},
	})
	if err != nil {
		return nil, cost, fmt.Errorf("wgpu: create matmul bind group: %w", err)
	}
	release = append(release, bindGroup)

	dispatchStart := time.Now()
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, cost, fmt.Errorf("wgpu: create matmul command encoder: %w", err)
	}
	pass, err := encoder.BeginComputePass(nil)
	if err != nil {
		return nil, cost, fmt.Errorf("wgpu: begin matmul compute pass: %w", err)
	}
	pass.SetPipeline(pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Dispatch(uint32((n+15)/16), uint32((m+15)/16), 1)
	if err := pass.End(); err != nil {
		return nil, cost, fmt.Errorf("wgpu: end matmul compute pass: %w", err)
	}
	encoder.CopyBufferToBuffer(cBuffer, 0, staging, 0, cBytes)
	commands, err := encoder.Finish()
	if err != nil {
		return nil, cost, fmt.Errorf("wgpu: finish matmul command encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, cost, fmt.Errorf("wgpu: submit matmul: %w", err)
	}
	cost.Dispatch = time.Since(dispatchStart)

	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	readbackStart := time.Now()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, cBytes); err != nil {
		return nil, cost, fmt.Errorf("%w: map matmul readback: %v", ErrReadbackTimeout, err)
	}
	mapped, err := staging.MappedRange(0, cBytes)
	if err != nil {
		_ = staging.Unmap()
		return nil, cost, fmt.Errorf("wgpu: matmul mapped range: %w", err)
	}
	raw := mapped.Bytes()
	result := make([]float32, m*n)
	for i := range result {
		result[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}
	if err := staging.Unmap(); err != nil {
		return nil, cost, fmt.Errorf("wgpu: unmap matmul readback: %w", err)
	}
	cost.Readback = time.Since(readbackStart)
	return result, cost, nil
}

func (h *handle) matmulPipeline() (*gowgpu.ComputePipeline, *gowgpu.BindGroupLayout, error) {
	h.matmulOnce.Do(func() {
		shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
			Label: "accel-matmul", WGSL: matmulWGSL,
		})
		if err != nil {
			h.matmulErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
			return
		}
		readOnly := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}
		storage := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}
		uniform := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}
		layout, err := h.device.CreateBindGroupLayout(&gowgpu.BindGroupLayoutDescriptor{
			Label: "accel-matmul-bgl",
			Entries: []gowgpu.BindGroupLayoutEntry{
				{Binding: 0, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
				{Binding: 1, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
				{Binding: 2, Visibility: gowgpu.ShaderStageCompute, Buffer: storage},
				{Binding: 3, Visibility: gowgpu.ShaderStageCompute, Buffer: uniform},
			},
		})
		if err != nil {
			h.matmulErr = fmt.Errorf("wgpu: matmul bind group layout: %w", err)
			return
		}
		pipelineLayout, err := h.device.CreatePipelineLayout(&gowgpu.PipelineLayoutDescriptor{
			Label: "accel-matmul-pl", BindGroupLayouts: []*gowgpu.BindGroupLayout{layout},
		})
		if err != nil {
			h.matmulErr = fmt.Errorf("wgpu: matmul pipeline layout: %w", err)
			return
		}
		pipeline, err := h.device.CreateComputePipeline(&gowgpu.ComputePipelineDescriptor{
			Label: "accel-matmul-pipeline", Layout: pipelineLayout, Module: shader, EntryPoint: "main",
		})
		if err != nil {
			h.matmulErr = fmt.Errorf("%w: %v", ErrShaderCompile, err)
			return
		}
		h.matmulPipe = pipeline
		h.matmulBGLayout = layout
	})
	return h.matmulPipe, h.matmulBGLayout, h.matmulErr
}
