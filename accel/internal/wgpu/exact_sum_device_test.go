package wgpu

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"sync"
	"testing"

	"github.com/gogpu/gputypes"
	gowgpu "github.com/gogpu/wgpu"
)

func TestExactSumDeviceKnownValues(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	p := func(n int) uint32 { return math.Float32bits(float32(math.Ldexp(1, n))) }
	b := math.Float32bits
	rows := [][][2]uint32{
		{},
		{{b(float32(0.25)), b(float32(0.3))}},
		{
			{b(1e30), b(1)},
			{b(-1e30), b(1)},
			{b(1e-30), b(1)},
		},
		{
			{b(1), b(1)},
			{p(-24), b(1)},
		},
		{
			{b(1), b(1)},
			{p(-23), b(1)},
			{p(-24), b(1)},
		},
		{{0x00000001, b(1.5)}},
		{{p(-75), p(-75)}},
		{{p(-75), b(float32(1.5 * math.Ldexp(1, -75)))}},
		{{b(float32(-math.Ldexp(1, -75))), p(-75)}},
		{
			{b(math.MaxFloat32), b(1)},
			{b(math.MaxFloat32), b(1)},
		},
		{
			{b(math.MaxFloat32), b(1)},
			{p(103), b(1)},
		},
		{
			{b(math.MaxFloat32), b(1)},
			{p(102), b(1)},
		},
		{{b(math.MaxFloat32), b(math.MaxFloat32)}},
		{
			{b(math.MaxFloat32), b(math.MaxFloat32)},
			{b(-math.MaxFloat32), b(math.MaxFloat32)},
		},
		{
			{p(60), p(60)},
			{b(float32(-math.Ldexp(1, -149))), p(-149)},
		},
		{
			{b(-3), b(1)},
			{b(1), b(1)},
		},
		{
			{b(1), b(1)},
			{b(-1), b(1)},
		},
		{{b(float32(math.Copysign(0, -1))), b(5)}},
		{{0x7fc00000, b(1)}},
		{{0x7f800001, b(1)}},
		{{b(float32(math.Inf(1))), b(0)}},
		{
			{b(float32(math.Inf(1))), b(1)},
			{b(float32(math.Inf(-1))), b(1)},
		},
		{{b(float32(math.Inf(1))), b(-2)}},
		{
			{b(float32(math.Inf(1))), b(1)},
			{b(5), b(1)},
		},
	}
	want := []uint32{
		0x00000000,
		b(float32(0.25) * float32(0.3)),
		b(1e-30),
		0x3f800000,
		0x3f800002,
		0x00000002,
		0x00000000,
		0x00000001,
		0x80000000,
		0x7f800000,
		0x7f800000,
		0x7f7fffff,
		0x7f800000,
		0x00000000,
		p(120),
		b(-2),
		0x00000000,
		0x00000000,
		0x7fc00000,
		0x7fc00000,
		0x7fc00000,
		0x7fc00000,
		0xff800000,
		0x7f800000,
	}

	single, split, err := runExactSumHarness(context.Background(), rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(single) != len(want) || len(split) != len(want) {
		t.Fatalf("output lengths = %d/%d, want %d/%d", len(single), len(split), len(want), len(want))
	}
	for i := range want {
		oracle := exactSumOracleBits(rows[i])
		var x, y uint32
		if len(rows[i]) != 0 {
			x = rows[i][0][0]
			y = rows[i][0][1]
		}
		if oracle != want[i] {
			t.Errorf("row %d x=%#08x y=%#08x oracle=%#08x single=%#08x split=%#08x", i, x, y, oracle, single[i], split[i])
		}
		if single[i] != want[i] {
			t.Errorf("row %d x=%#08x y=%#08x oracle=%#08x single=%#08x split=%#08x", i, x, y, oracle, single[i], split[i])
		}
		if split[i] != want[i] {
			t.Errorf("row %d x=%#08x y=%#08x oracle=%#08x single=%#08x split=%#08x", i, x, y, oracle, single[i], split[i])
		}
	}
}

func exactSumOracleBits(pairs [][2]uint32) uint32 {
	sum := new(big.Float).SetPrec(2048).SetMode(big.ToNearestEven)
	nan := false
	posInf := false
	negInf := false
	for _, pair := range pairs {
		x := math.Float32frombits(pair[0])
		y := math.Float32frombits(pair[1])
		if math.IsNaN(float64(x)) || math.IsNaN(float64(y)) {
			nan = true
			continue
		}
		if math.IsInf(float64(x), 0) || math.IsInf(float64(y), 0) {
			if x == 0 || y == 0 {
				nan = true
			} else if math.Signbit(float64(x)) != math.Signbit(float64(y)) {
				negInf = true
			} else {
				posInf = true
			}
			continue
		}
		if x != 0 && y != 0 {
			product := new(big.Float).SetFloat64(float64(x) * float64(y))
			sum.Add(sum, product)
		}
	}

	if nan || (posInf && negInf) {
		return 0x7fc00000
	}
	if posInf {
		return 0x7f800000
	}
	if negInf {
		return 0xff800000
	}
	if sum.Sign() == 0 {
		return 0
	}
	f, _ := sum.Float32()
	return math.Float32bits(f)
}

const exactSumHarnessWGSL = `
@group(0) @binding(0) var<storage, read> offsets: array<u32>;
@group(0) @binding(1) var<storage, read> xs: array<u32>;
@group(0) @binding(2) var<storage, read> ys: array<u32>;
@group(0) @binding(3) var<storage, read_write> outputs: array<u32>;
struct HarnessParams { rows: u32, rowStride: u32, pad0: u32, pad1: u32, }
@group(0) @binding(4) var<uniform> params: HarnessParams;

@compute @workgroup_size(64, 1, 1)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let row = gid.x + gid.y * params.rowStride;
    if (row >= params.rows) { return; }
    let start = offsets[row];
    let end = offsets[row + 1u];
    var single: ExactRegister;
    exact_clear(&single);
    var even: ExactRegister;
    exact_clear(&even);
    var odd: ExactRegister;
    exact_clear(&odd);
    for (var k: u32 = start; k < end; k = k + 1u) {
        exact_add_product(&single, xs[k], ys[k]);
    }
    for (var j: u32 = 0u; j < end - start; j = j + 1u) {
        let k = end - 1u - j;
        if ((j & 1u) == 0u) {
            exact_add_product(&even, xs[k], ys[k]);
        } else {
            exact_add_product(&odd, xs[k], ys[k]);
        }
    }
    exact_merge(&even, odd);
    outputs[2u * row] = exact_round(single);
    outputs[2u * row + 1u] = exact_round(even);
}
`

type exactSumCachedPipeline struct {
	pipeline *gowgpu.ComputePipeline
	layout   *gowgpu.BindGroupLayout
}

var exactSumPipelineCacheMu sync.Mutex
var exactSumPipelineCache *exactSumCachedPipeline

func exactSumPipelineFor(h *handle) (*gowgpu.ComputePipeline, *gowgpu.BindGroupLayout, error) {
	exactSumPipelineCacheMu.Lock()
	defer exactSumPipelineCacheMu.Unlock()
	if exactSumPipelineCache != nil {
		return exactSumPipelineCache.pipeline, exactSumPipelineCache.layout, nil
	}

	shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
		Label: "accel-exact-sum", WGSL: exactSumWGSL + exactSumHarnessWGSL,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: exact-sum shader: %w", ErrShaderCompile, err)
	}
	readOnly := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}
	storage := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}
	uniform := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}
	layout, err := h.device.CreateBindGroupLayout(&gowgpu.BindGroupLayoutDescriptor{
		Label: "accel-exact-sum-bgl",
		Entries: []gputypes.BindGroupLayoutEntry{
			{Binding: 0, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
			{Binding: 1, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
			{Binding: 2, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
			{Binding: 3, Visibility: gowgpu.ShaderStageCompute, Buffer: storage},
			{Binding: 4, Visibility: gowgpu.ShaderStageCompute, Buffer: uniform},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("wgpu: exact-sum bind group layout: %w", err)
	}
	pipelineLayout, err := h.device.CreatePipelineLayout(&gowgpu.PipelineLayoutDescriptor{
		Label: "accel-exact-sum-pl", BindGroupLayouts: []*gowgpu.BindGroupLayout{layout},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("wgpu: exact-sum pipeline layout: %w", err)
	}
	pipeline, err := h.device.CreateComputePipeline(&gowgpu.ComputePipelineDescriptor{
		Label: "accel-exact-sum-pipeline", Layout: pipelineLayout, Module: shader, EntryPoint: "main",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: exact-sum pipeline: %w", ErrShaderCompile, err)
	}
	exactSumPipelineCache = &exactSumCachedPipeline{pipeline: pipeline, layout: layout}
	return pipeline, layout, nil
}

func exactSumBufferBytes(length int) uint64 {
	if length == 0 {
		return 4
	}
	return uint64(length) * 4
}

func exactSumUint32Bytes(values []uint32) []byte {
	if len(values) == 0 {
		return make([]byte, 4)
	}
	data := make([]byte, len(values)*4)
	for i, value := range values {
		binary.LittleEndian.PutUint32(data[i*4:], value)
	}
	return data
}

func runExactSumHarness(ctx context.Context, rows [][][2]uint32) (single, split []uint32, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(rows) == 0 {
		return []uint32{}, []uint32{}, nil
	}

	offsets := make([]uint32, len(rows)+1)
	xs := make([]uint32, 0)
	ys := make([]uint32, 0)
	for i, row := range rows {
		for _, pair := range row {
			xs = append(xs, pair[0])
			ys = append(ys, pair[1])
		}
		offsets[i+1] = uint32(len(xs))
	}

	submitMu.Lock()
	defer submitMu.Unlock()

	h, err := acquire()
	if err != nil {
		return nil, nil, fmt.Errorf("%w: exact-sum: %w", ErrUnavailable, err)
	}
	pipeline, layout, err := exactSumPipelineFor(h)
	if err != nil {
		return nil, nil, fmt.Errorf("wgpu: exact-sum pipeline: %w", err)
	}

	outputBytes := exactSumBufferBytes(2 * len(rows))
	release := make([]interface{ Release() }, 0, 8)
	defer func() {
		for i := len(release) - 1; i >= 0; i-- {
			release[i].Release()
		}
	}()
	makeBuffer := func(label string, size uint64, usage gputypes.BufferUsage) (*gowgpu.Buffer, error) {
		buffer, bufferErr := h.device.CreateBuffer(&gowgpu.BufferDescriptor{Label: label, Size: size, Usage: usage})
		if bufferErr != nil {
			return nil, fmt.Errorf("%w: create %s buffer: %w", ErrBufferTooLarge, label, bufferErr)
		}
		release = append(release, buffer)
		return buffer, nil
	}

	offsetsBytes := exactSumBufferBytes(len(offsets))
	xsBytes := exactSumBufferBytes(len(xs))
	ysBytes := exactSumBufferBytes(len(ys))
	offsetsBuffer, err := makeBuffer("accel-exact-sum-offsets", offsetsBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, nil, err
	}
	xsBuffer, err := makeBuffer("accel-exact-sum-xs", xsBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, nil, err
	}
	ysBuffer, err := makeBuffer("accel-exact-sum-ys", ysBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, nil, err
	}
	outputsBuffer, err := makeBuffer("accel-exact-sum-outputs", outputBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, nil, err
	}
	staging, err := makeBuffer("accel-exact-sum-staging", outputBytes, gowgpu.BufferUsageCopyDst|gowgpu.BufferUsageMapRead)
	if err != nil {
		return nil, nil, err
	}
	paramsBuffer, err := makeBuffer("accel-exact-sum-params", 16, gowgpu.BufferUsageUniform|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, nil, err
	}

	groups := (len(rows)-1)/64 + 1
	x := groups
	if x > maxWorkgroups {
		x = maxWorkgroups
	}
	y := groups / x
	if groups%x != 0 {
		y++
	}
	if y > maxWorkgroups {
		return nil, nil, fmt.Errorf("wgpu: exact-sum dispatch exceeds workgroup limit: %dx%d", x, y)
	}
	rowStride := x * 64
	paramsBytes := make([]byte, 16)
	binary.LittleEndian.PutUint32(paramsBytes[0:], uint32(len(rows)))
	binary.LittleEndian.PutUint32(paramsBytes[4:], uint32(rowStride))
	binary.LittleEndian.PutUint32(paramsBytes[8:], 0)
	binary.LittleEndian.PutUint32(paramsBytes[12:], 0)

	if err := h.device.Queue().WriteBuffer(offsetsBuffer, 0, exactSumUint32Bytes(offsets)); err != nil {
		return nil, nil, fmt.Errorf("wgpu: upload exact-sum offsets: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(xsBuffer, 0, exactSumUint32Bytes(xs)); err != nil {
		return nil, nil, fmt.Errorf("wgpu: upload exact-sum xs: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(ysBuffer, 0, exactSumUint32Bytes(ys)); err != nil {
		return nil, nil, fmt.Errorf("wgpu: upload exact-sum ys: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(paramsBuffer, 0, paramsBytes); err != nil {
		return nil, nil, fmt.Errorf("wgpu: upload exact-sum parameters: %w", err)
	}

	bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
		Label: "accel-exact-sum-bg", Layout: layout,
		Entries: []gowgpu.BindGroupEntry{
			{Binding: 0, Buffer: offsetsBuffer, Size: offsetsBytes},
			{Binding: 1, Buffer: xsBuffer, Size: xsBytes},
			{Binding: 2, Buffer: ysBuffer, Size: ysBytes},
			{Binding: 3, Buffer: outputsBuffer, Size: outputBytes},
			{Binding: 4, Buffer: paramsBuffer, Size: 16},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("wgpu: create exact-sum bind group: %w", err)
	}
	release = append(release, bindGroup)

	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("wgpu: create exact-sum command encoder: %w", err)
	}
	pass, err := encoder.BeginComputePass(nil)
	if err != nil {
		return nil, nil, fmt.Errorf("wgpu: begin exact-sum compute pass: %w", err)
	}
	pass.SetPipeline(pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Dispatch(uint32(x), uint32(y), 1)
	if err := pass.End(); err != nil {
		return nil, nil, fmt.Errorf("wgpu: end exact-sum compute pass: %w", err)
	}
	encoder.CopyBufferToBuffer(outputsBuffer, 0, staging, 0, outputBytes)
	commands, err := encoder.Finish()
	if err != nil {
		return nil, nil, fmt.Errorf("wgpu: finish exact-sum command encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, nil, fmt.Errorf("wgpu: submit exact-sum: %w", err)
	}

	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, outputBytes); err != nil {
		return nil, nil, fmt.Errorf("%w: map exact-sum readback: %w", ErrReadbackTimeout, err)
	}
	mapped, err := staging.MappedRange(0, outputBytes)
	if err != nil {
		_ = staging.Unmap()
		return nil, nil, fmt.Errorf("wgpu: exact-sum mapped range: %w", err)
	}
	raw := mapped.Bytes()
	if uint64(len(raw)) < outputBytes {
		_ = staging.Unmap()
		return nil, nil, fmt.Errorf("wgpu: exact-sum mapped range returned %d bytes, want %d", len(raw), outputBytes)
	}
	values := make([]uint32, 2*len(rows))
	for i := range values {
		values[i] = binary.LittleEndian.Uint32(raw[i*4:])
	}
	if err := staging.Unmap(); err != nil {
		return nil, nil, fmt.Errorf("wgpu: unmap exact-sum readback: %w", err)
	}

	single = make([]uint32, len(rows))
	split = make([]uint32, len(rows))
	for i := range rows {
		single[i] = values[2*i]
		split[i] = values[2*i+1]
	}
	return single, split, nil
}
