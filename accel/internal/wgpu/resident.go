package wgpu

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"github.com/gogpu/gputypes"
	gowgpu "github.com/gogpu/wgpu"
)

// Buffer is a float32 array that lives on the device. Kernels read and write
// it without a host copy; Download is the only way its values reach the host.
// A Buffer holds float32 bit patterns and never materialises them as f32 in a
// shader, so nothing a shader compiler does to floating point can change them.
type Buffer struct {
	mu       sync.Mutex
	buffer   *gowgpu.Buffer
	length   int
	released bool
}

// maxPendingObjects bounds the uniform buffers and bind groups kept alive
// until the queue is known to be idle; reaching it forces a synchronisation.
const maxPendingObjects = 512

// pending holds the small per-dispatch objects whose release must wait until
// the work that used them has finished. It is guarded by submitMu.
var pending []interface{ Release() }

func bufferBytes(length int) uint64 {
	if length < 1 {
		return 4
	}
	return uint64(length) * 4
}

// NewBuffer allocates a zero-filled buffer of length float32 values.
func NewBuffer(length int) (*Buffer, error) {
	if length < 0 {
		return nil, fmt.Errorf("wgpu: buffer length %d is negative", length)
	}
	submitMu.Lock()
	defer submitMu.Unlock()
	return newBufferLocked(length)
}

func newBufferLocked(length int) (*Buffer, error) {
	h, err := acquire()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	size := bufferBytes(length)
	if size > h.limits.MaxStorageBufferBindingSize {
		return nil, fmt.Errorf("%w: %d bytes exceeds the device limit of %d", ErrBufferTooLarge, size, h.limits.MaxStorageBufferBindingSize)
	}
	buffer, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-resident", Size: size,
		Usage: gowgpu.BufferUsageStorage | gowgpu.BufferUsageCopyDst | gowgpu.BufferUsageCopySrc,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: create resident buffer: %w", ErrBufferTooLarge, err)
	}
	return &Buffer{buffer: buffer, length: length}, nil
}

// Upload allocates a buffer holding values.
func Upload(values []float32) (*Buffer, error) {
	submitMu.Lock()
	defer submitMu.Unlock()
	buffer, err := newBufferLocked(len(values))
	if err != nil {
		return nil, err
	}
	if len(values) > 0 {
		h, _ := acquire()
		if err := h.device.Queue().WriteBuffer(buffer.buffer, 0, float32Bytes(values)); err != nil {
			buffer.releaseLocked()
			return nil, fmt.Errorf("wgpu: upload resident buffer: %w", err)
		}
	}
	return buffer, nil
}

// Len reports how many float32 values the buffer holds.
func (b *Buffer) Len() int { return b.length }

// Bytes reports the device memory the buffer occupies.
func (b *Buffer) Bytes() uint64 { return bufferBytes(b.length) }

// Released reports whether Release has been called.
func (b *Buffer) Released() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.released
}

// Release frees the device memory. It is idempotent.
func (b *Buffer) Release() {
	submitMu.Lock()
	defer submitMu.Unlock()
	b.releaseLocked()
}

func (b *Buffer) releaseLocked() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.released {
		return
	}
	b.released = true
	if b.buffer != nil {
		// The queue may still be reading the buffer; keep it until idle.
		pending = append(pending, b.buffer)
		b.buffer = nil
	}
}

// Download copies the buffer back to the host, waiting for every earlier
// submission that wrote to it.
func (b *Buffer) Download(ctx context.Context) ([]float32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	submitMu.Lock()
	defer submitMu.Unlock()
	values, err := b.downloadLocked(ctx)
	if err == nil {
		flushPending()
	}
	return values, err
}

func (b *Buffer) downloadLocked(ctx context.Context) ([]float32, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.released {
		return nil, fmt.Errorf("wgpu: buffer was released")
	}
	if b.length == 0 {
		return []float32{}, nil
	}
	h, err := acquire()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	size := bufferBytes(b.length)
	staging, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-resident-staging", Size: size,
		Usage: gowgpu.BufferUsageCopyDst | gowgpu.BufferUsageMapRead,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: create staging buffer: %w", ErrBufferTooLarge, err)
	}
	defer staging.Release()
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, fmt.Errorf("wgpu: create download encoder: %w", err)
	}
	encoder.CopyBufferToBuffer(b.buffer, 0, staging, 0, size)
	commands, err := encoder.Finish()
	if err != nil {
		return nil, fmt.Errorf("wgpu: finish download encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return nil, fmt.Errorf("wgpu: submit download: %w", err)
	}
	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, size); err != nil {
		return nil, fmt.Errorf("%w: map resident readback: %w", ErrReadbackTimeout, err)
	}
	mapped, err := staging.MappedRange(0, size)
	if err != nil {
		_ = staging.Unmap()
		return nil, fmt.Errorf("wgpu: resident mapped range: %w", err)
	}
	raw := mapped.Bytes()
	values := make([]float32, b.length)
	for i := range values {
		values[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}
	if err := staging.Unmap(); err != nil {
		return nil, fmt.Errorf("wgpu: unmap resident readback: %w", err)
	}
	return values, nil
}

// flushPending releases what earlier dispatches kept alive. It is called only
// when the queue is known to be idle. submitMu must be held.
func flushPending() {
	for _, resource := range pending {
		resource.Release()
	}
	pending = pending[:0]
}

// syncLocked waits for the queue to go idle by reading back four bytes.
func syncLocked() error {
	probe, err := newBufferLocked(1)
	if err != nil {
		return err
	}
	_, err = probe.downloadLocked(context.Background())
	probe.mu.Lock()
	if probe.buffer != nil {
		probe.buffer.Release()
		probe.buffer = nil
	}
	probe.released = true
	probe.mu.Unlock()
	if err == nil {
		flushPending()
	}
	return err
}

// kernel is one compute shader over resident buffers: nRead read-only storage
// buffers, then nWrite read-write ones, then one 16-byte uniform. Its source
// is the exact-sum and IEEE library followed by body.
type kernel struct {
	name   string
	body   string
	nRead  int
	nWrite int

	once   sync.Once
	pipe   *gowgpu.ComputePipeline
	layout *gowgpu.BindGroupLayout
	err    error
}

func (k *kernel) build(h *handle) {
	k.once.Do(func() {
		shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
			Label: "accel-" + k.name, WGSL: exactSumWGSL + ieeeWGSL + k.body,
		})
		if err != nil {
			k.err = fmt.Errorf("%w: %s: %w", ErrShaderCompile, k.name, err)
			return
		}
		entries := make([]gputypes.BindGroupLayoutEntry, 0, k.nRead+k.nWrite+1)
		for i := 0; i < k.nRead; i++ {
			entries = append(entries, gputypes.BindGroupLayoutEntry{
				Binding: uint32(i), Visibility: gowgpu.ShaderStageCompute,
				Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage},
			})
		}
		for i := 0; i < k.nWrite; i++ {
			entries = append(entries, gputypes.BindGroupLayoutEntry{
				Binding: uint32(k.nRead + i), Visibility: gowgpu.ShaderStageCompute,
				Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage},
			})
		}
		entries = append(entries, gputypes.BindGroupLayoutEntry{
			Binding: uint32(k.nRead + k.nWrite), Visibility: gowgpu.ShaderStageCompute,
			Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform},
		})
		layout, err := h.device.CreateBindGroupLayout(&gowgpu.BindGroupLayoutDescriptor{
			Label: "accel-" + k.name + "-bgl", Entries: entries,
		})
		if err != nil {
			k.err = fmt.Errorf("wgpu: %s bind group layout: %w", k.name, err)
			return
		}
		pipelineLayout, err := h.device.CreatePipelineLayout(&gowgpu.PipelineLayoutDescriptor{
			Label: "accel-" + k.name + "-pl", BindGroupLayouts: []*gowgpu.BindGroupLayout{layout},
		})
		if err != nil {
			k.err = fmt.Errorf("wgpu: %s pipeline layout: %w", k.name, err)
			return
		}
		pipeline, err := h.device.CreateComputePipeline(&gowgpu.ComputePipelineDescriptor{
			Label: "accel-" + k.name + "-pipeline", Layout: pipelineLayout, Module: shader, EntryPoint: "main",
		})
		if err != nil {
			k.err = fmt.Errorf("%w: %s: %w", ErrShaderCompile, k.name, err)
			return
		}
		k.pipe, k.layout = pipeline, layout
	})
}

// run submits the kernel and returns without waiting for it to finish.
func (k *kernel) run(reads, writes []*Buffer, params [16]uint32, groupsX, groupsY uint32) error {
	if len(reads) != k.nRead || len(writes) != k.nWrite {
		return fmt.Errorf("wgpu: %s takes %d read and %d write buffers, got %d and %d", k.name, k.nRead, k.nWrite, len(reads), len(writes))
	}
	submitMu.Lock()
	defer submitMu.Unlock()
	h, err := acquire()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	k.build(h)
	if k.err != nil {
		return k.err
	}
	if len(pending) >= maxPendingObjects {
		if err := syncLocked(); err != nil {
			return err
		}
	}
	all := append(append([]*Buffer(nil), reads...), writes...)
	entries := make([]gowgpu.BindGroupEntry, 0, len(all)+1)
	for i, buffer := range all {
		buffer.mu.Lock()
		released, handle, size := buffer.released, buffer.buffer, bufferBytes(buffer.length)
		buffer.mu.Unlock()
		if released || handle == nil {
			return fmt.Errorf("wgpu: %s: buffer %d was released", k.name, i)
		}
		entries = append(entries, gowgpu.BindGroupEntry{Binding: uint32(i), Buffer: handle, Size: size})
	}
	uniform, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-" + k.name + "-params", Size: 64,
		Usage: gowgpu.BufferUsageUniform | gowgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return fmt.Errorf("wgpu: %s: create parameters: %w", k.name, err)
	}
	pending = append(pending, uniform)
	raw := make([]byte, 64)
	for i, value := range params {
		binary.LittleEndian.PutUint32(raw[i*4:], value)
	}
	if err := h.device.Queue().WriteBuffer(uniform, 0, raw); err != nil {
		return fmt.Errorf("wgpu: %s: upload parameters: %w", k.name, err)
	}
	entries = append(entries, gowgpu.BindGroupEntry{Binding: uint32(len(all)), Buffer: uniform, Size: 64})
	bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
		Label: "accel-" + k.name + "-bg", Layout: k.layout, Entries: entries,
	})
	if err != nil {
		return fmt.Errorf("wgpu: %s: create bind group: %w", k.name, err)
	}
	pending = append(pending, bindGroup)
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return fmt.Errorf("wgpu: %s: create command encoder: %w", k.name, err)
	}
	pass, err := encoder.BeginComputePass(nil)
	if err != nil {
		return fmt.Errorf("wgpu: %s: begin compute pass: %w", k.name, err)
	}
	pass.SetPipeline(k.pipe)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Dispatch(groupsX, groupsY, 1)
	if err := pass.End(); err != nil {
		return fmt.Errorf("wgpu: %s: end compute pass: %w", k.name, err)
	}
	commands, err := encoder.Finish()
	if err != nil {
		return fmt.Errorf("wgpu: %s: finish command encoder: %w", k.name, err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return fmt.Errorf("wgpu: %s: submit: %w", k.name, err)
	}
	return nil
}

// Synchronize waits until every earlier submission has finished.
func Synchronize() error {
	submitMu.Lock()
	defer submitMu.Unlock()
	return syncLocked()
}
