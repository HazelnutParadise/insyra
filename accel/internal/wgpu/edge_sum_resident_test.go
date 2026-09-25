package wgpu

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"

	"github.com/gogpu/gputypes"
	gowgpu "github.com/gogpu/wgpu"
)

type edgeSumResident struct {
	csr          edgeSumCSR
	offsets      *gowgpu.Buffer
	edges        *gowgpu.Buffer
	sources      *gowgpu.Buffer
	offsetsBytes uint64
	edgesBytes   uint64
	sourcesBytes uint64
}

func newEdgeSumResident(csr edgeSumCSR) (*edgeSumResident, error) {
	var weights, values []float32
	if csr.nodes >= 0 {
		weights = make([]float32, len(csr.edges))
		values = make([]float32, csr.nodes)
	}
	if err := validateEdgeSumInputs(csr, weights, values, csr.nodes); err != nil {
		return nil, fmt.Errorf("wgpu: edge-sum resident inputs: %w", err)
	}

	submitMu.Lock()
	defer submitMu.Unlock()

	h, err := acquire()
	if err != nil {
		return nil, fmt.Errorf("%w: edge-sum resident: %w", ErrUnavailable, err)
	}
	resident := &edgeSumResident{csr: csr}
	created := false
	defer func() {
		if !created {
			resident.Release()
		}
	}()

	resident.offsetsBytes = edgeSumBufferBytes(len(csr.offsets))
	resident.edgesBytes = edgeSumBufferBytes(len(csr.edges))
	resident.sourcesBytes = edgeSumBufferBytes(len(csr.sources))

	resident.offsets, err = h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-edge-sum-resident-offsets",
		Size:  resident.offsetsBytes,
		Usage: gowgpu.BufferUsageStorage | gowgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: create accel-edge-sum-resident-offsets buffer: %w", ErrBufferTooLarge, err)
	}
	if err := h.device.Queue().WriteBuffer(resident.offsets, 0, edgeSumUint32Bytes(csr.offsets)); err != nil {
		return nil, fmt.Errorf("wgpu: upload edge-sum resident offsets: %w", err)
	}

	resident.edges, err = h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-edge-sum-resident-edges",
		Size:  resident.edgesBytes,
		Usage: gowgpu.BufferUsageStorage | gowgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: create accel-edge-sum-resident-edges buffer: %w", ErrBufferTooLarge, err)
	}
	if err := h.device.Queue().WriteBuffer(resident.edges, 0, edgeSumUint32Bytes(csr.edges)); err != nil {
		return nil, fmt.Errorf("wgpu: upload edge-sum resident edges: %w", err)
	}

	resident.sources, err = h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-edge-sum-resident-sources",
		Size:  resident.sourcesBytes,
		Usage: gowgpu.BufferUsageStorage | gowgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: create accel-edge-sum-resident-sources buffer: %w", ErrBufferTooLarge, err)
	}
	if err := h.device.Queue().WriteBuffer(resident.sources, 0, edgeSumUint32Bytes(csr.sources)); err != nil {
		return nil, fmt.Errorf("wgpu: upload edge-sum resident sources: %w", err)
	}

	created = true
	return resident, nil
}

func (r *edgeSumResident) Release() {
	if r == nil {
		return
	}
	if r.offsets != nil {
		r.offsets.Release()
		r.offsets = nil
	}
	if r.edges != nil {
		r.edges.Release()
		r.edges = nil
	}
	if r.sources != nil {
		r.sources.Release()
		r.sources = nil
	}
}

func (r *edgeSumResident) run(ctx context.Context, wgsl string, weights, values []float32, batch int) ([]float32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if batch < 0 {
		return nil, fmt.Errorf("wgpu: edge-sum batch must not be negative: %d", batch)
	}
	if r.csr.nodes < 0 {
		return nil, fmt.Errorf("wgpu: edge-sum node count must not be negative: %d", r.csr.nodes)
	}
	if batch != 0 && r.csr.nodes > int(^uint(0)>>1)/batch {
		return nil, fmt.Errorf("wgpu: edge-sum output count overflows int")
	}
	total := batch * r.csr.nodes
	if total == 0 {
		return []float32{}, nil
	}
	if err := validateEdgeSumInputs(r.csr, weights, values, total); err != nil {
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
	release := make([]interface{ Release() }, 0, 6)
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

	weightsBytes := edgeSumBufferBytes(len(weights))
	valuesBytes := edgeSumBufferBytes(len(values))
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
	binary.LittleEndian.PutUint32(paramsBytes[0:], uint32(r.csr.nodes))
	binary.LittleEndian.PutUint32(paramsBytes[4:], uint32(total))
	binary.LittleEndian.PutUint32(paramsBytes[8:], uint32(rowStride))
	binary.LittleEndian.PutUint32(paramsBytes[12:], 0)

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
		Label: "accel-edge-sum-resident-bg", Layout: layout,
		Entries: []gowgpu.BindGroupEntry{
			{Binding: 0, Buffer: r.offsets, Size: r.offsetsBytes},
			{Binding: 1, Buffer: r.edges, Size: r.edgesBytes},
			{Binding: 2, Buffer: r.sources, Size: r.sourcesBytes},
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

func TestEdgeSumResidentMatchesFullUpload(t *testing.T) {
	if os.Getenv("INSYRA_ACCEL_GPU_TESTS") != "1" {
		t.Skip("set INSYRA_ACCEL_GPU_TESTS=1")
	}
	if _, err := Probe(); err != nil {
		t.Skipf("cannot discover a usable GPU: %v", err)
	}

	const (
		nodes = 2000
		edges = 60000
		batch = 4
	)
	rng := rand.New(rand.NewSource(7))
	sources := make([]int, edges)
	targets := make([]int, edges)
	for edge := range edges {
		sources[edge] = rng.Intn(nodes)
		targets[edge] = rng.Intn(200)
	}
	csr := newEdgeSumCSR(nodes, sources, targets)

	resident, err := newEdgeSumResident(csr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(resident.Release)

	for call := 1; call <= 2; call++ {
		weights := make([]float32, edges)
		for edge := range weights {
			weights[edge] = float32(rng.NormFloat64())
		}
		values := make([]float32, batch*nodes)
		for value := range values {
			values[value] = float32(rng.NormFloat64())
		}

		got, err := resident.run(context.Background(), edgeSumSeparatedWGSL, weights, values, batch)
		if err != nil {
			t.Fatalf("resident call %d: %v", call, err)
		}
		want, err := runEdgeSumPrototype(context.Background(), edgeSumSeparatedWGSL, csr, weights, values, batch)
		if err != nil {
			t.Fatalf("full-upload call %d: %v", call, err)
		}
		if len(got) != len(want) {
			t.Fatalf("resident call %d returned %d outputs, want %d", call, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("resident call %d output %d = %v, want %v", call, i, got[i], want[i])
			}
		}
	}
}
