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
	return &Kernel{k: &kernel{name: "custom-" + name, body: source, nRead: nRead, nWrite: nWrite, noLibrary: true}}, nil
}

// Kernel is a caller's compute shader.
type Kernel struct{ k *kernel }

// Name reports the name the kernel was created with.
func (k *Kernel) Name() string { return k.k.name[len("custom-"):] }

// Dispatch is one launch of a kernel: its buffers in binding order, its
// uniform values and its workgroup counts.
type Dispatch struct {
	Kernel  *Kernel
	Reads   []*Buffer
	Writes  []*Buffer
	Params  [16]uint32
	X, Y, Z uint32
}

// Submit encodes the dispatches, in order, into one submission and returns
// without waiting for it. Nothing is submitted when any dispatch cannot be
// bound.
func Submit(dispatches []Dispatch) error {
	launches := make([]launch, len(dispatches))
	for i, d := range dispatches {
		if d.Kernel == nil {
			return fmt.Errorf("wgpu: dispatch %d has no kernel", i)
		}
		if d.X == 0 || d.Y == 0 || d.Z == 0 {
			return fmt.Errorf("wgpu: dispatch %d: %s has a zero workgroup count", i, d.Kernel.Name())
		}
		launches[i] = launch{k: d.Kernel.k, reads: d.Reads, writes: d.Writes, params: d.Params, x: d.X, y: d.Y, z: d.Z}
	}
	return submit(launches)
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
			released, handle, size := buffer.released, buffer.buffer, bufferBytes(buffer.length)
			buffer.mu.Unlock()
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
		return fmt.Errorf("wgpu: submit: %w", err)
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
		return errors.New("wgpu: buffer was released")
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
	return nil
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
