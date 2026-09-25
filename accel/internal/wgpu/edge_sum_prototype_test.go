package wgpu

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gogpu/gputypes"
	gowgpu "github.com/gogpu/wgpu"
)

func TestEdgeSumPrototypeSmoke(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	variants := []struct {
		name string
		wgsl string
	}{
		{name: "plain", wgsl: edgeSumPlainWGSL},
		{name: "separated", wgsl: edgeSumSeparatedWGSL},
	}

	csr := newEdgeSumCSR(3, []int{0, 1, 2}, []int{1, 2, 0})
	weights := []float32{0.5, -1, 0.25}
	values := []float32{0.1, 0.2, 0.3}
	want := []float32{
		float32(float32(0.25) * float32(0.3)),
		float32(float32(0.5) * float32(0.1)),
		float32(float32(-1) * float32(0.2)),
	}
	for _, variant := range variants {
		t.Run(variant.name+"/single", func(t *testing.T) {
			got, err := runEdgeSumPrototype(context.Background(), variant.wgsl, csr, weights, values, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(want) {
				t.Fatalf("got %d outputs, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("output %d = %v (%08x), want %v (%08x)", i, got[i], float32Bits(got[i]), want[i], float32Bits(want[i]))
				}
			}
		})
	}

	empty := newEdgeSumCSR(3, nil, nil)
	emptyValues := make([]float32, 6)
	emptyWant := make([]float32, 6)
	for _, variant := range variants {
		t.Run(variant.name+"/empty", func(t *testing.T) {
			got, err := runEdgeSumPrototype(context.Background(), variant.wgsl, empty, nil, emptyValues, 2)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(emptyWant) {
				t.Fatalf("got %d outputs, want %d", len(got), len(emptyWant))
			}
			for i := range emptyWant {
				if got[i] != emptyWant[i] {
					t.Fatalf("output %d = %v (%08x), want %v (%08x)", i, got[i], float32Bits(got[i]), emptyWant[i], float32Bits(emptyWant[i]))
				}
			}
		})
	}
}

func float32Bits(value float32) uint32 {
	return math.Float32bits(value)
}

const edgeSumWGSLTemplate = `@group(0) @binding(0) var<storage, read> offsets: array<u32>;
@group(0) @binding(1) var<storage, read> edges: array<u32>;
@group(0) @binding(2) var<storage, read> sources: array<u32>;
@group(0) @binding(3) var<storage, read> weights: array<f32>;
@group(0) @binding(4) var<storage, read> values: array<f32>;
@group(0) @binding(5) var<storage, read_write> outputs: array<f32>;
struct Params { nodes: u32, total: u32, rowStride: u32, zero: u32, }
@group(0) @binding(6) var<uniform> params: Params;

@compute @workgroup_size(256, 1, 1)
fn main(@builtin(global_invocation_id) gid: vec3<u32>) {
    let i = gid.x + gid.y * params.rowStride;
    if (i >= params.total) { return; }
    let b = i / params.nodes;
    let t = i % params.nodes;
    var acc: f32 = 0.0;
    for (var k: u32 = offsets[t]; k < offsets[t + 1u]; k = k + 1u) {
        let e = edges[k];
        let p = weights[e] * values[b * params.nodes + sources[e]];
        ACCUMULATE
    }
    outputs[i] = acc;
}
`

var edgeSumPlainWGSL = strings.Replace(edgeSumWGSLTemplate, "ACCUMULATE", "acc = acc + p;", 1)
var edgeSumSeparatedWGSL = strings.Replace(edgeSumWGSLTemplate, "ACCUMULATE", "acc = acc + bitcast<f32>(bitcast<u32>(p) ^ params.zero);", 1)

type edgeSumCSR struct {
	offsets, edges, sources []uint32
	nodes                   int
}

func newEdgeSumCSR(nodes int, sources, targets []int) edgeSumCSR {
	if nodes < 0 {
		panic("newEdgeSumCSR: negative node count")
	}
	if len(sources) != len(targets) {
		panic("newEdgeSumCSR: source and target counts differ")
	}
	for _, source := range sources {
		if source < 0 || source >= nodes {
			panic("newEdgeSumCSR: source out of range")
		}
	}

	offsets := make([]int, nodes+1)
	for _, target := range targets {
		if target < 0 || target >= nodes {
			panic("newEdgeSumCSR: target out of range")
		}
		offsets[target+1]++
	}
	for node := 1; node < len(offsets); node++ {
		offsets[node] += offsets[node-1]
	}

	edges := make([]uint32, len(targets))
	convertedSources := make([]uint32, len(sources))
	for edgeIndex, source := range sources {
		convertedSources[edgeIndex] = uint32(source)
	}
	next := append([]int(nil), offsets[:nodes]...)
	for edgeIndex, target := range targets {
		position := next[target]
		edges[position] = uint32(edgeIndex)
		next[target]++
	}

	convertedOffsets := make([]uint32, len(offsets))
	for i, offset := range offsets {
		convertedOffsets[i] = uint32(offset)
	}
	return edgeSumCSR{offsets: convertedOffsets, edges: edges, sources: convertedSources, nodes: nodes}
}

type edgeSumCachedPipeline struct {
	pipeline *gowgpu.ComputePipeline
	layout   *gowgpu.BindGroupLayout
}

var edgeSumPipelineCacheMu sync.Mutex
var edgeSumPipelineCache = make(map[string]edgeSumCachedPipeline)

func edgeSumPipelineFor(h *handle, wgsl string) (*gowgpu.ComputePipeline, *gowgpu.BindGroupLayout, error) {
	edgeSumPipelineCacheMu.Lock()
	defer edgeSumPipelineCacheMu.Unlock()
	if cached, ok := edgeSumPipelineCache[wgsl]; ok {
		return cached.pipeline, cached.layout, nil
	}

	shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
		Label: "accel-edge-sum", WGSL: wgsl,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: edge-sum shader: %w", ErrShaderCompile, err)
	}
	readOnly := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}
	storage := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}
	uniform := &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}
	layout, err := h.device.CreateBindGroupLayout(&gowgpu.BindGroupLayoutDescriptor{
		Label: "accel-edge-sum-bgl",
		Entries: []gputypes.BindGroupLayoutEntry{
			{Binding: 0, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
			{Binding: 1, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
			{Binding: 2, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
			{Binding: 3, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
			{Binding: 4, Visibility: gowgpu.ShaderStageCompute, Buffer: readOnly},
			{Binding: 5, Visibility: gowgpu.ShaderStageCompute, Buffer: storage},
			{Binding: 6, Visibility: gowgpu.ShaderStageCompute, Buffer: uniform},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("wgpu: edge-sum bind group layout: %w", err)
	}
	pipelineLayout, err := h.device.CreatePipelineLayout(&gowgpu.PipelineLayoutDescriptor{
		Label: "accel-edge-sum-pl", BindGroupLayouts: []*gowgpu.BindGroupLayout{layout},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("wgpu: edge-sum pipeline layout: %w", err)
	}
	pipeline, err := h.device.CreateComputePipeline(&gowgpu.ComputePipelineDescriptor{
		Label: "accel-edge-sum-pipeline", Layout: pipelineLayout, Module: shader, EntryPoint: "main",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("%w: edge-sum pipeline: %w", ErrShaderCompile, err)
	}
	edgeSumPipelineCache[wgsl] = edgeSumCachedPipeline{pipeline: pipeline, layout: layout}
	return pipeline, layout, nil
}

func edgeSumBufferBytes(length int) uint64 {
	if length == 0 {
		return 4
	}
	return uint64(length) * 4
}

func edgeSumUint32Bytes(values []uint32) []byte {
	if len(values) == 0 {
		return make([]byte, 4)
	}
	data := make([]byte, len(values)*4)
	for i, value := range values {
		binary.LittleEndian.PutUint32(data[i*4:], value)
	}
	return data
}

func edgeSumFloat32Bytes(values []float32) []byte {
	if len(values) == 0 {
		return make([]byte, 4)
	}
	return float32Bytes(values)
}

func validateEdgeSumInputs(csr edgeSumCSR, weights, values []float32, total int) error {
	if csr.nodes < 0 {
		return fmt.Errorf("negative node count %d", csr.nodes)
	}
	if len(csr.offsets) != csr.nodes+1 {
		return fmt.Errorf("offsets length %d, want %d", len(csr.offsets), csr.nodes+1)
	}
	if len(csr.edges) != len(csr.sources) {
		return fmt.Errorf("edge/source lengths differ: %d/%d", len(csr.edges), len(csr.sources))
	}
	if len(weights) != len(csr.edges) {
		return fmt.Errorf("weights length %d, want %d", len(weights), len(csr.edges))
	}
	if len(values) != total {
		return fmt.Errorf("values length %d, want %d", len(values), total)
	}
	if uint64(csr.nodes) > uint64(^uint32(0)) || uint64(total) > uint64(^uint32(0)) {
		return fmt.Errorf("node and output counts must fit in uint32")
	}

	previous := uint32(0)
	for i, offset := range csr.offsets {
		if i == 0 && offset != 0 {
			return fmt.Errorf("offsets[0] = %d, want 0", offset)
		}
		if offset < previous {
			return fmt.Errorf("offsets[%d] = %d, less than previous value %d", i, offset, previous)
		}
		if uint64(offset) > uint64(len(csr.edges)) {
			return fmt.Errorf("offsets[%d] = %d, exceeds edge count %d", i, offset, len(csr.edges))
		}
		previous = offset
	}
	if previous != uint32(len(csr.edges)) {
		return fmt.Errorf("final offset = %d, want %d", previous, len(csr.edges))
	}
	for i, source := range csr.sources {
		if uint64(source) >= uint64(csr.nodes) {
			return fmt.Errorf("sources[%d] = %d, outside node count %d", i, source, csr.nodes)
		}
	}
	for i, edge := range csr.edges {
		if uint64(edge) >= uint64(len(weights)) {
			return fmt.Errorf("edges[%d] = %d, outside weight count %d", i, edge, len(weights))
		}
	}
	return nil
}

func runEdgeSumPrototype(ctx context.Context, wgsl string, csr edgeSumCSR, weights, values []float32, batch int) ([]float32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if batch < 0 {
		return nil, fmt.Errorf("wgpu: edge-sum batch must not be negative: %d", batch)
	}
	if csr.nodes < 0 {
		return nil, fmt.Errorf("wgpu: edge-sum node count must not be negative: %d", csr.nodes)
	}
	if batch != 0 && csr.nodes > int(^uint(0)>>1)/batch {
		return nil, fmt.Errorf("wgpu: edge-sum output count overflows int")
	}
	total := batch * csr.nodes
	if total == 0 {
		return []float32{}, nil
	}
	if err := validateEdgeSumInputs(csr, weights, values, total); err != nil {
		return nil, fmt.Errorf("wgpu: edge-sum inputs: %w", err)
	}

	submitMu.Lock()
	defer submitMu.Unlock()

	h, err := acquire()
	if err != nil {
		return nil, fmt.Errorf("%w: edge-sum: %w", ErrUnavailable, err)
	}
	pipeline, layout, err := edgeSumPipelineFor(h, wgsl)
	if err != nil {
		return nil, fmt.Errorf("wgpu: edge-sum pipeline: %w", err)
	}

	outputBytes := uint64(total) * 4
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

	offsetsBytes := edgeSumBufferBytes(len(csr.offsets))
	edgesBytes := edgeSumBufferBytes(len(csr.edges))
	sourcesBytes := edgeSumBufferBytes(len(csr.sources))
	weightsBytes := edgeSumBufferBytes(len(weights))
	valuesBytes := edgeSumBufferBytes(len(values))
	offsetsBuffer, err := makeBuffer("accel-edge-sum-offsets", offsetsBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, err
	}
	edgesBuffer, err := makeBuffer("accel-edge-sum-edges", edgesBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, err
	}
	sourcesBuffer, err := makeBuffer("accel-edge-sum-sources", sourcesBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, err
	}
	weightsBuffer, err := makeBuffer("accel-edge-sum-weights", weightsBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, err
	}
	valuesBuffer, err := makeBuffer("accel-edge-sum-values", valuesBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, err
	}
	outputsBuffer, err := makeBuffer("accel-edge-sum-outputs", outputBytes, gowgpu.BufferUsageStorage|gowgpu.BufferUsageCopySrc)
	if err != nil {
		return nil, err
	}
	staging, err := makeBuffer("accel-edge-sum-staging", outputBytes, gowgpu.BufferUsageCopyDst|gowgpu.BufferUsageMapRead)
	if err != nil {
		return nil, err
	}
	paramsBuffer, err := makeBuffer("accel-edge-sum-params", 16, gowgpu.BufferUsageUniform|gowgpu.BufferUsageCopyDst)
	if err != nil {
		return nil, err
	}

	groups := (total-1)/256 + 1
	x := groups
	if x > maxWorkgroups {
		x = maxWorkgroups
	}
	y := groups / x
	if groups%x != 0 {
		y++
	}
	if y > maxWorkgroups {
		return nil, fmt.Errorf("wgpu: edge-sum dispatch exceeds workgroup limit: %dx%d", x, y)
	}
	rowStride := x * 256
	paramsBytes := make([]byte, 16)
	binary.LittleEndian.PutUint32(paramsBytes[0:], uint32(csr.nodes))
	binary.LittleEndian.PutUint32(paramsBytes[4:], uint32(total))
	binary.LittleEndian.PutUint32(paramsBytes[8:], uint32(rowStride))
	binary.LittleEndian.PutUint32(paramsBytes[12:], 0)

	if err := h.device.Queue().WriteBuffer(offsetsBuffer, 0, edgeSumUint32Bytes(csr.offsets)); err != nil {
		return nil, fmt.Errorf("wgpu: upload edge-sum offsets: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(edgesBuffer, 0, edgeSumUint32Bytes(csr.edges)); err != nil {
		return nil, fmt.Errorf("wgpu: upload edge-sum edges: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(sourcesBuffer, 0, edgeSumUint32Bytes(csr.sources)); err != nil {
		return nil, fmt.Errorf("wgpu: upload edge-sum sources: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(weightsBuffer, 0, edgeSumFloat32Bytes(weights)); err != nil {
		return nil, fmt.Errorf("wgpu: upload edge-sum weights: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(valuesBuffer, 0, edgeSumFloat32Bytes(values)); err != nil {
		return nil, fmt.Errorf("wgpu: upload edge-sum values: %w", err)
	}
	if err := h.device.Queue().WriteBuffer(paramsBuffer, 0, paramsBytes); err != nil {
		return nil, fmt.Errorf("wgpu: upload edge-sum parameters: %w", err)
	}

	bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
		Label: "accel-edge-sum-bg", Layout: layout,
		Entries: []gowgpu.BindGroupEntry{
			{Binding: 0, Buffer: offsetsBuffer, Size: offsetsBytes},
			{Binding: 1, Buffer: edgesBuffer, Size: edgesBytes},
			{Binding: 2, Buffer: sourcesBuffer, Size: sourcesBytes},
			{Binding: 3, Buffer: weightsBuffer, Size: weightsBytes},
			{Binding: 4, Buffer: valuesBuffer, Size: valuesBytes},
			{Binding: 5, Buffer: outputsBuffer, Size: outputBytes},
			{Binding: 6, Buffer: paramsBuffer, Size: 16},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("wgpu: create edge-sum bind group: %w", err)
	}
	release = append(release, bindGroup)

	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, fmt.Errorf("wgpu: create edge-sum command encoder: %w", err)
	}
	pass, err := encoder.BeginComputePass(nil)
	if err != nil {
		return nil, fmt.Errorf("wgpu: begin edge-sum compute pass: %w", err)
	}
	pass.SetPipeline(pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Dispatch(uint32(x), uint32(y), 1)
	if err := pass.End(); err != nil {
		return nil, fmt.Errorf("wgpu: end edge-sum compute pass: %w", err)
	}
	encoder.CopyBufferToBuffer(outputsBuffer, 0, staging, 0, outputBytes)
	commands, err := encoder.Finish()
	if err != nil {
		return nil, fmt.Errorf("wgpu: finish edge-sum command encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, fmt.Errorf("wgpu: submit edge-sum: %w", err)
	}

	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, outputBytes); err != nil {
		return nil, fmt.Errorf("%w: map edge-sum readback: %w", ErrReadbackTimeout, err)
	}
	mapped, err := staging.MappedRange(0, outputBytes)
	if err != nil {
		_ = staging.Unmap()
		return nil, fmt.Errorf("wgpu: edge-sum mapped range: %w", err)
	}
	raw := mapped.Bytes()
	if uint64(len(raw)) < outputBytes {
		_ = staging.Unmap()
		return nil, fmt.Errorf("wgpu: edge-sum mapped range returned %d bytes, want %d", len(raw), outputBytes)
	}
	result := make([]float32, total)
	for i := range result {
		result[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}
	if err := staging.Unmap(); err != nil {
		return nil, fmt.Errorf("wgpu: unmap edge-sum readback: %w", err)
	}
	return result, nil
}
