package wgpu

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	gowgpu "github.com/gogpu/wgpu"
)

// NewKernel returns a caller's compute shader with the resident binding
// layout: nRead read-only storage buffers, nWrite read-write ones, then the
// 64-byte uniform. The source is compiled as it is, with none of Insyra's
// WGSL before it. It compiles on first use. How many storage buffers one
// kernel may bind is the device's limit, checked when it is submitted.
func NewKernel(name, source string, nRead, nWrite int) (*Kernel, error) {
	switch {
	case name == "":
		return nil, errors.New("wgpu: a kernel needs a name")
	case source == "":
		return nil, fmt.Errorf("wgpu: kernel %s has no source", name)
	case nRead < 0 || nWrite < 0:
		return nil, fmt.Errorf("wgpu: kernel %s binds a negative number of buffers", name)
	}
	return &Kernel{k: &kernel{name: "custom-" + name, body: source, nRead: nRead, nWrite: nWrite, noLibrary: true, dynamicUniform: true}}, nil
}

// Kernel is a caller's compute shader.
type Kernel struct{ k *kernel }

// Name reports the name the kernel was created with.
func (k *Kernel) Name() string { return k.k.name[len("custom-"):] }

// Binding is a kernel tied to its buffers: the bind group is built once and
// every Submit reuses it, so a loop that launches the same kernel on the same
// buffers allocates nothing per launch. Release it when done.
type Binding struct {
	k             *Kernel
	reads, writes []*Buffer
	group         *gowgpu.BindGroup
	released      bool
	lost          bool // the device was lost; see retireLocked
}

// uniformSlots is how many 64-byte parameter blocks the shared ring holds.
// The blocks of one Submit are consecutive; when a Submit would run past the
// end the queue is drained and the ring starts again at its beginning, so a
// block is rewritten only when no submitted work can still be reading it.
const uniformSlots = 4096

var (
	uniformRing   *gowgpu.Buffer
	uniformStride uint64
	uniformNext   int
)

// ensureRingLocked creates the shared parameter ring. submitMu must be held.
func ensureRingLocked(h *handle) error {
	if uniformRing != nil {
		return nil
	}
	stride := uint64(256)
	if align := uint64(h.limits.MinUniformBufferOffsetAlignment); align > stride {
		stride = align
	}
	ring, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
		Label: "accel-custom-params", Size: stride * uniformSlots,
		Usage: gowgpu.BufferUsageUniform | gowgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return fmt.Errorf("wgpu: create the parameter ring: %w", err)
	}
	uniformRing, uniformStride = ring, stride
	return nil
}

// Bind ties the kernel to its buffers. It compiles the kernel if needed and
// refuses the wrong number of buffers, a nil or released buffer, one buffer
// both read and written, and more storage buffers than the device allows.
func (k *Kernel) Bind(reads, writes []*Buffer) (*Binding, error) {
	name := k.k.name
	if len(reads) != k.k.nRead || len(writes) != k.k.nWrite {
		return nil, fmt.Errorf("wgpu: %s takes %d read-only and %d read-write buffers, got %d and %d", name, k.k.nRead, k.k.nWrite, len(reads), len(writes))
	}
	if err := distinctBuffers(reads, writes); err != nil {
		return nil, fmt.Errorf("wgpu: %s: %w", name, err)
	}
	submitMu.Lock()
	defer submitMu.Unlock()
	// A buffer that died with a lost device is reported as such, before
	// acquire tries to open a device that may not come back.
	if err := lostBuffers(name, append(append([]*Buffer(nil), reads...), writes...)); err != nil {
		return nil, err
	}
	h, err := acquire()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if limit := int(h.limits.MaxStorageBuffersPerShaderStage); limit > 0 && k.k.nRead+k.k.nWrite > limit {
		return nil, fmt.Errorf("wgpu: %s binds %d storage buffers, the device allows %d", name, k.k.nRead+k.k.nWrite, limit)
	}
	k.k.build(h)
	if k.k.err != nil {
		return nil, k.k.err
	}
	if err := ensureRingLocked(h); err != nil {
		return nil, err
	}
	if err := boundPendingLocked(); err != nil {
		return nil, err
	}
	all := append(append([]*Buffer(nil), reads...), writes...)
	entries := make([]gowgpu.BindGroupEntry, 0, len(all)+1)
	for j, buffer := range all {
		if buffer == nil {
			return nil, fmt.Errorf("wgpu: %s: buffer %d is nil", name, j)
		}
		buffer.mu.Lock()
		released, lost, handle, size := buffer.released, buffer.lost, buffer.buffer, bufferBytes(buffer.length)
		buffer.mu.Unlock()
		if lost {
			return nil, fmt.Errorf("%w: %s: buffer %d", ErrDeviceLost, name, j)
		}
		if released || handle == nil {
			return nil, fmt.Errorf("wgpu: %s: buffer %d was released", name, j)
		}
		entries = append(entries, gowgpu.BindGroupEntry{Binding: uint32(j), Buffer: handle, Size: size})
	}
	entries = append(entries, gowgpu.BindGroupEntry{Binding: uint32(len(all)), Buffer: uniformRing, Size: 64})
	group, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
		Label: "accel-" + name + "-bg", Layout: k.k.layout, Entries: entries,
	})
	if err != nil {
		return nil, fmt.Errorf("wgpu: %s: create bind group: %w", name, err)
	}
	binding := &Binding{k: k, reads: append([]*Buffer(nil), reads...), writes: append([]*Buffer(nil), writes...), group: group}
	liveBindings[binding] = struct{}{}
	return binding, nil
}

// Release frees the bind group once the queue is idle. It is idempotent. The
// buffers stay with their owners.
func (b *Binding) Release() {
	submitMu.Lock()
	defer submitMu.Unlock()
	if b.released {
		return
	}
	b.released = true
	delete(liveBindings, b)
	pending = append(pending, b.group)
	b.group = nil
	// A loop that binds and releases without ever submitting would otherwise
	// keep every released bind group until the next synchronisation.
	_ = boundPendingLocked()
}

// Dispatch is one launch of a binding: its uniform values and its workgroup
// counts.
type Dispatch struct {
	Binding *Binding
	Params  [16]uint32
	X, Y, Z uint32
}

// Submit encodes the dispatches, in order, into one submission and returns
// without waiting for it. Nothing is submitted when any dispatch cannot run.
// The parameters of all dispatches are written to the shared ring in one
// upload before the submission, each at its own offset, and read through a
// dynamic offset, so no uniform buffer or bind group is created per launch.
func Submit(dispatches []Dispatch) error {
	if len(dispatches) == 0 {
		return nil
	}
	if len(dispatches) > uniformSlots {
		return fmt.Errorf("wgpu: %d dispatches in one submission, at most %d", len(dispatches), uniformSlots)
	}
	submitMu.Lock()
	defer submitMu.Unlock()
	for i, d := range dispatches {
		if d.Binding != nil && d.Binding.lost {
			return fmt.Errorf("%w: dispatch %d: the binding", ErrDeviceLost, i)
		}
	}
	h, err := acquire()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	most := h.limits.MaxComputeWorkgroupsPerDimension
	for i, d := range dispatches {
		if d.Binding != nil && d.Binding.lost {
			return fmt.Errorf("%w: dispatch %d: the binding", ErrDeviceLost, i)
		}
		if d.Binding == nil || d.Binding.released {
			return fmt.Errorf("wgpu: dispatch %d has no binding or its binding was released", i)
		}
		name := d.Binding.k.k.name
		if d.X == 0 || d.Y == 0 || d.Z == 0 {
			return fmt.Errorf("wgpu: dispatch %d: %s has a zero workgroup count", i, name)
		}
		if most > 0 && (d.X > most || d.Y > most || d.Z > most) {
			return fmt.Errorf("wgpu: dispatch %d: %s asks for %d x %d x %d workgroups, the device allows %d in each dimension", i, name, d.X, d.Y, d.Z, most)
		}
		for j, buffer := range append(append([]*Buffer(nil), d.Binding.reads...), d.Binding.writes...) {
			buffer.mu.Lock()
			released, lost := buffer.released, buffer.lost
			buffer.mu.Unlock()
			if lost {
				return fmt.Errorf("%w: dispatch %d: %s: buffer %d", ErrDeviceLost, i, name, j)
			}
			if released {
				return fmt.Errorf("wgpu: dispatch %d: %s: buffer %d was released", i, name, j)
			}
		}
	}
	if uniformNext+len(dispatches) > uniformSlots {
		if err := syncLocked(); err != nil {
			return err
		}
		uniformNext = 0
	}
	first := uniformNext
	raw := make([]byte, uint64(len(dispatches))*uniformStride)
	for i, d := range dispatches {
		for j, value := range d.Params {
			binary.LittleEndian.PutUint32(raw[uint64(i)*uniformStride+uint64(j)*4:], value)
		}
	}
	if err := h.device.Queue().WriteBuffer(uniformRing, uint64(first)*uniformStride, raw); err != nil {
		return fmt.Errorf("wgpu: upload parameters: %w", err)
	}
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return fmt.Errorf("wgpu: create command encoder: %w", err)
	}
	for i, d := range dispatches {
		k := d.Binding.k.k
		pass, err := encoder.BeginComputePass(nil)
		if err != nil {
			return fmt.Errorf("wgpu: dispatch %d: %s: begin compute pass: %w", i, k.name, err)
		}
		pass.SetPipeline(k.pipe)
		pass.SetBindGroup(0, d.Binding.group, []uint32{uint32(uint64(first+i) * uniformStride)})
		pass.Dispatch(d.X, d.Y, d.Z)
		if err := pass.End(); err != nil {
			return fmt.Errorf("wgpu: dispatch %d: %s: end compute pass: %w", i, k.name, err)
		}
	}
	commands, err := encoder.Finish()
	if err != nil {
		return fmt.Errorf("wgpu: finish command encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return h.failed(fmt.Errorf("wgpu: submit: %w", err))
	}
	uniformNext = first + len(dispatches)
	return nil
}

// launch is one dispatch on the internal path every resident kernel takes.
type launch struct {
	k             *kernel
	reads, writes []*Buffer
	params        [16]uint32
	x, y, z       uint32
}

// submit binds every launch, encodes one compute pass per launch into one
// command encoder and submits once. WebGPU orders the passes of a submission,
// so each launch sees what the ones before it wrote; Insyra's own kernels and
// a caller's go through this one queue under submitMu, so separate calls are
// ordered the same way.
func submit(launches []launch) error {
	if len(launches) == 0 {
		return nil
	}
	submitMu.Lock()
	defer submitMu.Unlock()
	for _, l := range launches {
		if err := lostBuffers(l.k.name, append(append([]*Buffer(nil), l.reads...), l.writes...)); err != nil {
			return err
		}
	}
	h, err := acquire()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	limit := int(h.limits.MaxStorageBuffersPerShaderStage)
	type bound struct {
		entries []gowgpu.BindGroupEntry
	}
	plan := make([]bound, len(launches))
	for i, l := range launches {
		k := l.k
		if len(l.reads) != k.nRead || len(l.writes) != k.nWrite {
			return fmt.Errorf("wgpu: dispatch %d: %s takes %d read and %d write buffers, got %d and %d", i, k.name, k.nRead, k.nWrite, len(l.reads), len(l.writes))
		}
		if limit > 0 && k.nRead+k.nWrite > limit {
			return fmt.Errorf("wgpu: dispatch %d: %s binds %d storage buffers, the device allows %d", i, k.name, k.nRead+k.nWrite, limit)
		}
		if most := h.limits.MaxComputeWorkgroupsPerDimension; most > 0 && (l.x > most || l.y > most || l.z > most) {
			return fmt.Errorf("wgpu: dispatch %d: %s asks for %d x %d x %d workgroups, the device allows %d in each dimension", i, k.name, l.x, l.y, l.z, most)
		}
		if err := distinctBuffers(l.reads, l.writes); err != nil {
			return fmt.Errorf("wgpu: dispatch %d: %s: %w", i, k.name, err)
		}
		k.build(h)
		if k.err != nil {
			return k.err
		}
		all := append(append([]*Buffer(nil), l.reads...), l.writes...)
		entries := make([]gowgpu.BindGroupEntry, 0, len(all)+1)
		for j, buffer := range all {
			if buffer == nil {
				return fmt.Errorf("wgpu: dispatch %d: %s: buffer %d is nil", i, k.name, j)
			}
			buffer.mu.Lock()
			released, lost, handle, size := buffer.released, buffer.lost, buffer.buffer, bufferBytes(buffer.length)
			buffer.mu.Unlock()
			if lost {
				return fmt.Errorf("%w: dispatch %d: %s: buffer %d", ErrDeviceLost, i, k.name, j)
			}
			if released || handle == nil {
				return fmt.Errorf("wgpu: dispatch %d: %s: buffer %d was released", i, k.name, j)
			}
			entries = append(entries, gowgpu.BindGroupEntry{Binding: uint32(j), Buffer: handle, Size: size})
		}
		plan[i] = bound{entries: entries}
	}
	// Every launch keeps a uniform and a bind group alive until the queue is
	// idle. A call that would pass the bound waits for earlier work first.
	if len(pending) > 0 && len(pending)+2*len(launches) > maxPendingObjects {
		if err := syncLocked(); err != nil {
			return err
		}
	}
	encoder, err := h.device.CreateCommandEncoder(nil)
	if err != nil {
		return fmt.Errorf("wgpu: create command encoder: %w", err)
	}
	for i, l := range launches {
		k := l.k
		uniform, err := h.device.CreateBuffer(&gowgpu.BufferDescriptor{
			Label: "accel-" + k.name + "-params", Size: 64,
			Usage: gowgpu.BufferUsageUniform | gowgpu.BufferUsageCopyDst,
		})
		if err != nil {
			return fmt.Errorf("wgpu: dispatch %d: %s: create parameters: %w", i, k.name, err)
		}
		pending = append(pending, uniform)
		raw := make([]byte, 64)
		for j, value := range l.params {
			binary.LittleEndian.PutUint32(raw[j*4:], value)
		}
		if err := h.device.Queue().WriteBuffer(uniform, 0, raw); err != nil {
			return fmt.Errorf("wgpu: dispatch %d: %s: upload parameters: %w", i, k.name, err)
		}
		entries := append(plan[i].entries, gowgpu.BindGroupEntry{Binding: uint32(k.nRead + k.nWrite), Buffer: uniform, Size: 64})
		bindGroup, err := h.device.CreateBindGroup(&gowgpu.BindGroupDescriptor{
			Label: "accel-" + k.name + "-bg", Layout: k.layout, Entries: entries,
		})
		if err != nil {
			return fmt.Errorf("wgpu: dispatch %d: %s: create bind group: %w", i, k.name, err)
		}
		pending = append(pending, bindGroup)
		pass, err := encoder.BeginComputePass(nil)
		if err != nil {
			return fmt.Errorf("wgpu: dispatch %d: %s: begin compute pass: %w", i, k.name, err)
		}
		pass.SetPipeline(k.pipe)
		pass.SetBindGroup(0, bindGroup, nil)
		pass.Dispatch(l.x, l.y, l.z)
		if err := pass.End(); err != nil {
			return fmt.Errorf("wgpu: dispatch %d: %s: end compute pass: %w", i, k.name, err)
		}
	}
	commands, err := encoder.Finish()
	if err != nil {
		return fmt.Errorf("wgpu: finish command encoder: %w", err)
	}
	if _, err := h.device.Queue().Submit(commands); err != nil {
		return h.failed(fmt.Errorf("wgpu: submit: %w", err))
	}
	return nil
}

// distinctBuffers refuses a dispatch that binds one buffer twice with a
// write among them: WebGPU rejects that usage, and the error would otherwise
// surface only when the whole submission is finished, without a dispatch
// number.
func distinctBuffers(reads, writes []*Buffer) error {
	for i, w := range writes {
		for _, r := range reads {
			if w == r {
				return fmt.Errorf("buffer %d of the writes is also bound for reading", i)
			}
		}
		for j := 0; j < i; j++ {
			if writes[j] == w {
				return fmt.Errorf("buffer %d of the writes is also write buffer %d", i, j)
			}
		}
	}
	return nil
}

// UploadBits allocates a buffer holding bits as they are.
func UploadBits(bits []uint32) (*Buffer, error) {
	submitMu.Lock()
	defer submitMu.Unlock()
	if err := boundPendingLocked(); err != nil {
		return nil, err
	}
	buffer, err := newBufferLocked(len(bits))
	if err != nil {
		return nil, err
	}
	if len(bits) > 0 {
		h, _ := acquire()
		if err := h.device.Queue().WriteBuffer(buffer.buffer, 0, bitsBytes(bits)); err != nil {
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

func bitsBytes(bits []uint32) []byte {
	raw := make([]byte, len(bits)*4)
	for i, v := range bits {
		binary.LittleEndian.PutUint32(raw[i*4:], v)
	}
	return raw
}

// DownloadBits copies the buffer back as bit patterns, waiting for every
// earlier submission that wrote to it. The words are read as bytes, never as
// floats, so a NaN payload comes back untouched on every platform.
func (b *Buffer) DownloadBits(ctx context.Context) ([]uint32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	submitMu.Lock()
	defer submitMu.Unlock()
	bits := make([]uint32, b.length)
	err := b.downloadRawLocked(ctx, func(raw []byte) {
		for i := range bits {
			bits[i] = binary.LittleEndian.Uint32(raw[i*4:])
		}
	})
	if err == nil && b.length > 0 {
		flushPending()
	}
	if err != nil {
		return nil, err
	}
	return bits, nil
}

// WriteBits overwrites the whole buffer with bits. It waits for every earlier
// submission first, because a write that raced a kernel still reading the
// buffer would give that kernel a mix of old and new words.
func (b *Buffer) WriteBits(bits []uint32) error { return b.write(len(bits), bitsBytes(bits)) }

// Write overwrites the whole buffer with values, with the same wait as
// WriteBits.
func (b *Buffer) Write(values []float32) error { return b.write(len(values), float32Bytes(values)) }

// Zero sets every word of the buffer to zero, with the same wait as WriteBits.
func (b *Buffer) Zero() error { return b.write(b.length, make([]byte, bufferBytes(b.length))) }

func (b *Buffer) write(length int, raw []byte) error {
	submitMu.Lock()
	defer submitMu.Unlock()
	b.mu.Lock()
	defer b.mu.Unlock()
	switch {
	case b.released || b.buffer == nil:
		return b.unusableLocked("buffer")
	case length != b.length:
		return fmt.Errorf("wgpu: buffer holds %d values, got %d", b.length, length)
	case length == 0:
		return nil
	}
	if err := syncLocked(); err != nil {
		return err
	}
	h, err := acquire()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if err := h.device.Queue().WriteBuffer(b.buffer, 0, raw); err != nil {
		return fmt.Errorf("wgpu: write resident buffer: %w", err)
	}
	return settleWriteLocked(h)
}

// Limits is what a device allows a caller's kernel.
type Limits struct {
	MaxStorageBuffers          int
	MaxBufferBytes             uint64
	MaxWorkgroupsPerDimension  uint32
	MaxInvocationsPerWorkgroup uint32
}

// DeviceLimits reports the limits of the device resident buffers live on.
func DeviceLimits() (Limits, error) {
	submitMu.Lock()
	defer submitMu.Unlock()
	h, err := acquire()
	if err != nil {
		return Limits{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return Limits{
		MaxStorageBuffers:          int(h.limits.MaxStorageBuffersPerShaderStage),
		MaxBufferBytes:             h.limits.MaxStorageBufferBindingSize,
		MaxWorkgroupsPerDimension:  h.limits.MaxComputeWorkgroupsPerDimension,
		MaxInvocationsPerWorkgroup: h.limits.MaxComputeInvocationsPerWorkgroup,
	}, nil
}
