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
	lost     bool // the device was lost; released is also set, so every use is refused
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
	if err := boundPendingLocked(); err != nil {
		return nil, err
	}
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
	b := &Buffer{buffer: buffer, length: length}
	liveBuffers[b] = struct{}{}
	return b, nil
}

// Upload allocates a buffer holding values.
func Upload(values []float32) (*Buffer, error) {
	submitMu.Lock()
	defer submitMu.Unlock()
	if err := boundPendingLocked(); err != nil {
		return nil, err
	}
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
		if err := settleWriteLocked(h); err != nil {
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
	// A loop that uploads and releases without ever dispatching a kernel
	// would otherwise keep every released allocation until the next sync.
	_ = boundPendingLocked()
}

// boundPendingLocked waits for the queue to go idle and frees the deferred
// releases once there are too many of them. submitMu must be held.
func boundPendingLocked() error {
	if len(pending) < maxPendingObjects {
		return nil
	}
	return syncLocked()
}

func (b *Buffer) releaseLocked() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.released {
		return
	}
	b.released = true
	delete(liveBuffers, b)
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
	// Only a download that waited for the queue proves earlier work is done;
	// an empty buffer returns without waiting, so it must not free anything.
	if err == nil && b.length > 0 {
		flushPending()
	}
	return values, err
}

func (b *Buffer) downloadLocked(ctx context.Context) ([]float32, error) {
	values := make([]float32, b.length)
	err := b.downloadRawLocked(ctx, func(raw []byte) {
		for i := range values {
			values[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		}
	})
	if err != nil {
		return nil, err
	}
	return values, nil
}

// downloadRawLocked copies the buffer to a staging buffer and calls decode
// with its bytes while they are mapped. decode must not keep raw.
func (b *Buffer) downloadRawLocked(ctx context.Context, decode func(raw []byte)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.released {
		return b.unusableLocked("buffer")
	}
	if b.length == 0 {
		return nil
	}
	h, err := acquire()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	size := bufferBytes(b.length)
	staging, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-resident-staging", Size: size,
		Usage: gowgpu.BufferUsageCopyDst | gowgpu.BufferUsageMapRead,
	})
	if err != nil {
		return fmt.Errorf("%w: create staging buffer: %w", ErrBufferTooLarge, err)
	}
	defer releaseQuietly(staging)
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return fmt.Errorf("wgpu: create download encoder: %w", err)
	}
	encoder.CopyBufferToBuffer(b.buffer, 0, staging, 0, size)
	commands, err := encoder.Finish()
	if err != nil {
		return fmt.Errorf("wgpu: finish download encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return h.failed(fmt.Errorf("wgpu: submit download: %w", err))
	}
	mapCtx, cancel := context.WithTimeout(ctx, readbackTimeout)
	defer cancel()
	if err := staging.Map(mapCtx, gowgpu.MapModeRead, 0, size); err != nil {
		return h.failed(fmt.Errorf("%w: map resident readback: %w", ErrReadbackTimeout, err))
	}
	mapped, err := staging.MappedRange(0, size)
	if err != nil {
		_ = staging.Unmap()
		return fmt.Errorf("wgpu: resident mapped range: %w", err)
	}
	decode(mapped.Bytes())
	if err := staging.Unmap(); err != nil {
		return fmt.Errorf("wgpu: unmap resident readback: %w", err)
	}
	return nil
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
	delete(liveBuffers, probe)
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
	// fixed adds the fixed-point library, with its constants, to the source.
	fixed bool
	// noLibrary leaves the exact-sum and IEEE library out: a caller's kernel
	// that does not ask for it gets only its own source.
	noLibrary bool
	// dynamicUniform makes the uniform binding take a dynamic offset, so a
	// caller's launches read their parameters from one shared ring.
	dynamicUniform bool

	once   sync.Once
	pipe   *gowgpu.ComputePipeline
	layout *gowgpu.BindGroupLayout
	// The shader module and pipeline layout are kept only so retireLocked can
	// release them: a device cannot be replaced while they are alive.
	module   *gowgpu.ShaderModule
	plLayout *gowgpu.PipelineLayout
	err      error
}

func (k *kernel) library() string {
	if k.fixed {
		return fixedLibrary()
	}
	return ""
}

func (k *kernel) source() string {
	if k.noLibrary {
		return k.body
	}
	return exactSumWGSL + ieeeWGSL + k.library() + k.body
}

func (k *kernel) build(h *handle) {
	k.once.Do(func() {
		shader, err := h.device.CreateShaderModule(&gowgpu.ShaderModuleDescriptor{
			Label: "accel-" + k.name, WGSL: k.source(),
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
			Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform, HasDynamicOffset: k.dynamicUniform},
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
		k.pipe, k.layout, k.module, k.plLayout = pipeline, layout, shader, pipelineLayout
		builtKernels[k] = struct{}{}
	})
}

// run submits the kernel and returns without waiting for it to finish.
func (k *kernel) run(reads, writes []*Buffer, params [16]uint32, groupsX, groupsY uint32) error {
	return submit([]launch{{k: k, reads: reads, writes: writes, params: params, x: groupsX, y: groupsY, z: 1}})
}

// Synchronize waits until every earlier submission has finished.
func Synchronize() error {
	submitMu.Lock()
	defer submitMu.Unlock()
	return syncLocked()
}

// settleWriteLocked makes a queue write to a resident buffer complete before
// anything else uses the buffer. gogpu stages Queue.WriteBuffer into a copy
// recorded ahead of the next submission, and its flush emits a barrier after
// that copy only for a buffer used as vertex, index, uniform or indirect data
// (pending_writes.go, bufferReadUsage). A storage buffer gets none, so on
// Vulkan the copy and the kernel or readback that follows it are unordered:
// measured on a GT 1030, Zero() followed by a readback returned the old words
// in 3 of 20 runs. Metal tracks the hazard itself. submitMu must be held.
func settleWriteLocked(h *handle) error {
	if h.info.Backend == gputypes.BackendMetal {
		return nil
	}
	return syncLocked()
}
