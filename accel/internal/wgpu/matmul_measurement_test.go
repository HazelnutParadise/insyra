package wgpu

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sync"
	"testing"
	"time"

	dl "github.com/HazelnutParadise/insyra/dl"
	"github.com/gogpu/gputypes"
	gowgpu "github.com/gogpu/wgpu"
)

// matmulWGSL is a 16x16 tiled f32 prototype. It is deliberately kept in the
// test surface: this measurement must not become a production dl device path.
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

type matmulMeasurementShape struct {
	name   string
	batch  int
	aShape [2]int
	bShape [2]int
}

var matmulPipelineState struct {
	sync.Once
	pipeline *gowgpu.ComputePipeline
	layout   *gowgpu.BindGroupLayout
	err      error
}

func measurementMatmulPipeline(h *handle) (*gowgpu.ComputePipeline, *gowgpu.BindGroupLayout, error) {
	matmulPipelineState.Do(func() {
		shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
			Label: "accel-matmul-measurement", WGSL: matmulWGSL,
		})
		if err != nil {
			matmulPipelineState.err = fmt.Errorf("matmul shader: %w", err)
			return
		}
		readOnly := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}
		storage := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}
		uniform := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}
		layout, err := h.device.CreateBindGroupLayout(&gowgpu.BindGroupLayoutDescriptor{
			Label: "accel-matmul-measurement-bgl",
			Entries: []gowgpu.BindGroupLayoutEntry{
				{Binding: 0, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
				{Binding: 1, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
				{Binding: 2, Visibility: gowgpu.ShaderStageCompute, Buffer: storage},
				{Binding: 3, Visibility: gowgpu.ShaderStageCompute, Buffer: uniform},
			},
		})
		if err != nil {
			matmulPipelineState.err = fmt.Errorf("matmul bind group layout: %w", err)
			return
		}
		pipelineLayout, err := h.device.CreatePipelineLayout(&gowgpu.PipelineLayoutDescriptor{
			Label:            "accel-matmul-measurement-pl",
			BindGroupLayouts: []*gowgpu.BindGroupLayout{layout},
		})
		if err != nil {
			matmulPipelineState.err = fmt.Errorf("matmul pipeline layout: %w", err)
			return
		}
		pipeline, err := h.device.CreateComputePipeline(&gowgpu.ComputePipelineDescriptor{
			Label:  "accel-matmul-measurement-pipeline",
			Layout: pipelineLayout, Module: shader, EntryPoint: "main",
		})
		if err != nil {
			matmulPipelineState.err = fmt.Errorf("matmul pipeline: %w", err)
			return
		}
		matmulPipelineState.pipeline = pipeline
		matmulPipelineState.layout = layout
	})
	return matmulPipelineState.pipeline, matmulPipelineState.layout, matmulPipelineState.err
}

func measurementMatmulDevice(ctx context.Context, h *handle, shape matmulMeasurementShape, a, b []float32) ([]float32, time.Duration, error) {
	submitMu.Lock()
	defer submitMu.Unlock()

	pipeline, layout, err := measurementMatmulPipeline(h)
	if err != nil {
		return nil, 0, err
	}
	start := time.Now()
	if shape.batch <= 1 {
		result, _, err := measurementMatmulDevice2D(ctx, h, pipeline, layout, shape.aShape[0], shape.aShape[1], shape.bShape[1], a, b)
		return result, time.Since(start), err
	}

	// Batched cases intentionally loop the 2-D kernel per batch. This prototype
	// measures the round trip cost of the existing kernel, not batched dispatch.
	aSize := shape.aShape[0] * shape.aShape[1]
	bSize := shape.bShape[0] * shape.bShape[1]
	cSize := shape.aShape[0] * shape.bShape[1]
	result := make([]float32, 0, shape.batch*cSize)
	for batch := 0; batch < shape.batch; batch++ {
		batchResult, _, err := measurementMatmulDevice2D(ctx, h, pipeline, layout,
			shape.aShape[0], shape.aShape[1], shape.bShape[1],
			a[batch*aSize:(batch+1)*aSize], b[batch*bSize:(batch+1)*bSize])
		if err != nil {
			return nil, time.Since(start), err
		}
		result = append(result, batchResult...)
	}
	return result, time.Since(start), nil
}

func measurementMatmulDevice2D(ctx context.Context, h *handle, pipeline *gowgpu.ComputePipeline, layout *gowgpu.BindGroupLayout, m, k, n int, a, b []float32) ([]float32, time.Duration, error) {
	if len(a) != m*k || len(b) != k*n {
		return nil, 0, fmt.Errorf("matmul input lengths do not match [%d,%d]x[%d,%d]", m, k, k, n)
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
		buffer, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{Label: label, Size: size, Usage: usage})
		if err != nil {
			return nil, fmt.Errorf("create %s buffer: %w", label, err)
		}
		release = append(release, buffer)
		return buffer, nil
	}

	aBuffer, err := makeBuffer("accel-matmul-measurement-a", aBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, 0, err
	}
	bBuffer, err := makeBuffer("accel-matmul-measurement-b", bBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, 0, err
	}
	cBuffer, err := makeBuffer("accel-matmul-measurement-c", cBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, 0, err
	}
	staging, err := makeBuffer("accel-matmul-measurement-staging", cBytes, gowgpu.BufferUsageCopyDst|gowgpu.BufferUsageMapRead)
	if err != nil {
		return nil, 0, err
	}
	params, err := makeBuffer("accel-matmul-measurement-params", 16, gowgpu.BufferUsageUniform|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, 0, err
	}

	paramsBytes := make([]byte, 16)
	binary.LittleEndian.PutUint32(paramsBytes[0:], uint32(m))
	binary.LittleEndian.PutUint32(paramsBytes[4:], uint32(k))
	binary.LittleEndian.PutUint32(paramsBytes[8:], uint32(n))
	if err := h.device.Queue().WriteBuffer(aBuffer, 0, float32Bytes(a)); err != nil {
		return nil, 0, fmt.Errorf("upload A: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(bBuffer, 0, float32Bytes(b)); err != nil {
		return nil, 0, fmt.Errorf("upload B: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(params, 0, paramsBytes); err != nil {
		return nil, 0, fmt.Errorf("upload matmul parameters: %w", err)
	}

	bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
		Label: "accel-matmul-measurement-bg", Layout: layout,
		Entries: []gowgpu.BindGroupEntry{
			{Binding: 0, Buffer: aBuffer, Size: aBytes},
			{Binding: 1, Buffer: bBuffer, Size: bBytes},
			{Binding: 2, Buffer: cBuffer, Size: cBytes},
			{Binding: 3, Buffer: params, Size: 16},
		},
	})
	if err != nil {
		return nil, 0, fmt.Errorf("create matmul bind group: %w", err)
	}
	release = append(release, bindGroup)

	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create matmul command encoder: %w", err)
	}
	pass, err := encoder.BeginComputePass(nil)
	if err != nil {
		return nil, 0, fmt.Errorf("begin matmul compute pass: %w", err)
	}
	pass.SetPipeline(pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Dispatch(uint32((n+15)/16), uint32((m+15)/16), 1)
	if err := pass.End(); err != nil {
		return nil, 0, fmt.Errorf("end matmul compute pass: %w", err)
	}
	encoder.CopyBufferToBuffer(cBuffer, 0, staging, 0, cBytes)
	commands, err := encoder.Finish()
	if err != nil {
		return nil, 0, fmt.Errorf("finish matmul command encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, 0, fmt.Errorf("submit matmul: %w", err)
	}

	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, cBytes); err != nil {
		return nil, 0, fmt.Errorf("map matmul readback: %w", err)
	}
	mapped, err := staging.MappedRange(0, cBytes)
	if err != nil {
		_ = staging.Unmap()
		return nil, 0, fmt.Errorf("matmul mapped range: %w", err)
	}
	raw := mapped.Bytes()
	result := make([]float32, m*n)
	for i := range result {
		result[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}
	if err := staging.Unmap(); err != nil {
		return nil, 0, fmt.Errorf("unmap matmul readback: %w", err)
	}
	return result, 0, nil
}

func TestDLMatMulDeviceMeasurement(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	h, err := acquire()
	if err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	shapes := []matmulMeasurementShape{
		{name: "[4096,256]x[256,256]", aShape: [2]int{4096, 256}, bShape: [2]int{256, 256}},
		{name: "[4096,256]x[256,1024]", aShape: [2]int{4096, 256}, bShape: [2]int{256, 1024}},
		{name: "[4096,1024]x[1024,256]", aShape: [2]int{4096, 1024}, bShape: [2]int{1024, 256}},
		{name: "128x([128,64]x[64,128])", batch: 128, aShape: [2]int{128, 64}, bShape: [2]int{64, 128}},
		{name: "128x([128,128]x[128,64])", batch: 128, aShape: [2]int{128, 128}, bShape: [2]int{128, 64}},
		{name: "[4096,4096]x[4096,4096]", aShape: [2]int{4096, 4096}, bShape: [2]int{4096, 4096}},
	}

	for _, shape := range shapes {
		shape := shape
		t.Run(shape.name, func(t *testing.T) {
			a, b, err := measurementMatmulInputs(shape)
			if err != nil {
				t.Fatalf("inputs: %v", err)
			}
			cpuWant, cpuMS, err := measurementMatmulCPU(shape, a, b)
			if err != nil {
				t.Fatalf("CPU MatMul: %v", err)
			}
			deviceGot, deviceMS, err := measurementMatmulDeviceBestOfFive(t, h, shape, a, b)
			if err != nil {
				t.Fatalf("device MatMul: %v", err)
			}
			if len(deviceGot) != len(cpuWant) {
				t.Fatalf("device returned %d values, CPU returned %d", len(deviceGot), len(cpuWant))
			}
			maxAbs, maxULP := measurementDeviation(cpuWant, deviceGot)
			fmt.Printf("shape=%s device_ms=%.3f cpu_ms=%.3f ratio=%.3f maxAbs=%.9g maxULP=%d\n",
				shape.name, deviceMS, cpuMS, deviceMS/cpuMS, maxAbs, maxULP)
		})
	}
}

func measurementMatmulInputs(shape matmulMeasurementShape) ([]float32, []float32, error) {
	batch := shape.batch
	if batch == 0 {
		batch = 1
	}
	aSize := batch * shape.aShape[0] * shape.aShape[1]
	bSize := batch * shape.bShape[0] * shape.bShape[1]
	a := make([]float32, aSize)
	b := make([]float32, bSize)
	for i := range a {
		a[i] = measurementValue(i, 11)
	}
	for i := range b {
		b[i] = measurementValue(i, 37)
	}
	return a, b, nil
}

func measurementValue(index, salt int) float32 {
	return float32((index*31+salt*17)%101-50) * 0.01
}

func measurementMatmulCPU(shape matmulMeasurementShape, a, b []float32) ([]float32, float64, error) {
	aTensor, err := dl.NewTensor(measurementTensorShape(shape, true), a)
	if err != nil {
		return nil, 0, err
	}
	bTensor, err := dl.NewTensor(measurementTensorShape(shape, false), b)
	if err != nil {
		return nil, 0, err
	}
	var result *dl.Tensor
	var bestDuration time.Duration
	var bestResult []float32
	for attempt := 0; attempt < 5; attempt++ {
		start := time.Now()
		result, err = dl.MatMul(aTensor, bTensor)
		elapsed := time.Since(start)
		if err != nil {
			return nil, 0, err
		}
		if attempt == 0 || elapsed < bestDuration {
			bestDuration = elapsed
			bestResult = result.Data()
		}
	}
	return bestResult, float64(bestDuration) / float64(time.Millisecond), nil
}

func measurementTensorShape(shape matmulMeasurementShape, left bool) []int {
	if shape.batch == 1 || shape.batch == 0 {
		if left {
			return []int{shape.aShape[0], shape.aShape[1]}
		}
		return []int{shape.bShape[0], shape.bShape[1]}
	}
	if left {
		return []int{shape.batch, shape.aShape[0], shape.aShape[1]}
	}
	return []int{shape.batch, shape.bShape[0], shape.bShape[1]}
}

func measurementMatmulDeviceBestOfFive(t *testing.T, h *handle, shape matmulMeasurementShape, a, b []float32) ([]float32, float64, error) {
	t.Helper()
	var best []float32
	var bestDuration time.Duration
	for attempt := 0; attempt < 5; attempt++ {
		result, elapsed, err := measurementMatmulDevice(context.Background(), h, shape, a, b)
		if err != nil {
			return nil, 0, err
		}
		if attempt == 0 || elapsed < bestDuration {
			best = result
			bestDuration = elapsed
		}
	}
	return best, float64(bestDuration) / float64(time.Millisecond), nil
}

func measurementDeviation(want, got []float32) (float32, uint64) {
	var maxAbs float32
	var maxULP uint64
	for i := range want {
		abs := float32(math.Abs(float64(got[i] - want[i])))
		if abs > maxAbs {
			maxAbs = abs
		}
		if ulp := measurementULP(want[i], got[i]); ulp > maxULP {
			maxULP = ulp
		}
	}
	return maxAbs, maxULP
}

func measurementULP(a, b float32) uint64 {
	if a == b {
		return 0
	}
	ordered := func(value float32) uint32 {
		bits := math.Float32bits(value)
		if bits&0x80000000 != 0 {
			return ^bits
		}
		return bits | 0x80000000
	}
	first, second := ordered(a), ordered(b)
	if first > second {
		return uint64(first - second)
	}
	return uint64(second - first)
}
