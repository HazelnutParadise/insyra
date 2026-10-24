package accel

import (
	"context"
	"errors"
	"fmt"

	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
)

// KernelConfig describes a caller's WGSL compute shader. Its bindings in
// group 0 are ReadOnly read-only storage buffers (bindings 0 to ReadOnly-1),
// then ReadWrite read-write storage buffers, then one uniform of sixteen u32
// values (64 bytes) at binding ReadOnly+ReadWrite. The entry point is main.
// This is the layout Insyra's own resident kernels use. The source is
// compiled as written: none of Insyra's WGSL comes before it.
//
// A kernel is a program of its own. Insyra does not check what it computes,
// so a tape that runs one is in PrecisionFloat32, not PrecisionFloat32Rounded
// (see nn.Tape.SetPrecision). A kernel can write anywhere in a read-write
// buffer it is given, and whether the device clamps a write past the end is
// up to the platform: Metal does, Vulkan does not in the backend Insyra uses,
// so a kernel that indexes past a buffer can overwrite another resident
// buffer there. Windows stops a single dispatch that runs for about two
// seconds, so keep each dispatch well under one. Optional shader features
// such as f16 or subgroups are not enabled on the device.
type KernelConfig struct {
	Name      string
	WGSL      string
	ReadOnly  int
	ReadWrite int
}

// Kernel is a caller's compute shader, compiled by its first Bind.
// Create it once and reuse it: each Kernel keeps its compiled pipeline for
// the life of the process.
type Kernel struct {
	inner  *wgpu.Kernel
	config KernelConfig
}

// NewKernel checks config and returns its kernel. It refuses an empty name or
// source and a negative buffer count. How many storage buffers a kernel may
// bind depends on the device (DeviceLimits), and a shader that does not
// compile is reported by the first Submit that uses it, as ErrShaderCompile.
func NewKernel(config KernelConfig) (*Kernel, error) {
	inner, err := wgpu.NewKernel(config.Name, config.WGSL, config.ReadOnly, config.ReadWrite)
	if err != nil {
		return nil, fmt.Errorf("accel: %w", err)
	}
	return &Kernel{inner: inner, config: config}, nil
}

// Binding ties a kernel to the buffers it reads and writes. Bind builds it
// once, and every Submit reuses it, so a loop that launches the same kernel on
// the same buffers allocates nothing per launch. Release it when done; the
// buffers stay with their owners.
type Binding struct {
	inner  *wgpu.Binding
	kernel *Kernel
	device Device
	// buffers keeps the DeviceBuffer wrappers reachable: each has a finalizer
	// that releases its memory, so a binding must not outlive the wrapper of a
	// buffer it still uses.
	buffers []*DeviceBuffer
}

// Bind ties the kernel to its buffers: ReadOnly buffers it only reads, then
// ReadWrite buffers it may write, in binding order. It compiles the kernel,
// so a shader that does not compile fails here with ErrShaderCompile. It
// refuses the wrong number of buffers, a nil or released buffer, a buffer on
// the host or on another device, one buffer both read and written (or written
// twice), no buffer at all, and more storage buffers than the device allows
// (ResidentLimits). A buffer on the host is refused rather than computed
// there: a caller keeps its own host implementation for when there is no
// device.
//
// A Binding refers to its buffers until it is released. Submitting it after
// one of them was released is an error.
func (k *Kernel) Bind(reads, writes []*DeviceBuffer) (*Binding, error) {
	name := k.config.Name
	if len(reads) != k.config.ReadOnly || len(writes) != k.config.ReadWrite {
		return nil, fmt.Errorf("accel: kernel %s takes %d read-only and %d read-write buffers, got %d and %d", name, k.config.ReadOnly, k.config.ReadWrite, len(reads), len(writes))
	}
	if len(reads)+len(writes) == 0 {
		return nil, fmt.Errorf("accel: kernel %s binds no buffer, so there is no device to run it on", name)
	}
	var device *Device
	innerReads, err := innerBuffers(name, "read-only", reads, &device)
	if err != nil {
		return nil, err
	}
	innerWrites, err := innerBuffers(name, "read-write", writes, &device)
	if err != nil {
		return nil, err
	}
	inner, err := k.inner.Bind(innerReads, innerWrites)
	if err != nil {
		return nil, residentFailure(err)
	}
	return &Binding{inner: inner, kernel: k, device: *device, buffers: append(append([]*DeviceBuffer(nil), reads...), writes...)}, nil
}

// Release frees the binding's bind group once the queue is idle. It is
// idempotent. The buffers it referred to are untouched.
func (b *Binding) Release() {
	if b == nil || b.inner == nil {
		return
	}
	b.inner.Release()
	b.buffers = nil
}

// Dispatch is one launch of a binding: the sixteen uniform values and the
// workgroup counts in x, y and z. Every workgroup count must be at least one.
type Dispatch struct {
	Binding    *Binding
	Params     [16]uint32
	Workgroups [3]uint32
}

// Submit sends the dispatches, in order, as one submission and returns
// without waiting for them. Each dispatch sees every write of the dispatches
// before it. Submit and Insyra's own resident operations share one queue and
// one submit lock, so work submitted by separate calls, either way round,
// also runs in the order it was submitted, with no synchronisation needed
// between them. At most 4096 dispatches go in one Submit.
//
// Submit sends nothing when any dispatch has no binding or a released one,
// a binding whose buffer was released, bindings on different devices, or a
// workgroup count of zero or past what the device allows. Parameters go to
// one shared ring, each dispatch at its own offset, and the queue is drained
// only when the ring is used up, which is every few thousand dispatches.
func Submit(dispatches ...Dispatch) error {
	if len(dispatches) == 0 {
		return nil
	}
	var device *Device
	inner := make([]wgpu.Dispatch, len(dispatches))
	for i, d := range dispatches {
		if d.Binding == nil || d.Binding.inner == nil {
			return fmt.Errorf("accel: dispatch %d has no binding", i)
		}
		switch {
		case device == nil:
			dev := d.Binding.device
			device = &dev
		case device.ID != d.Binding.device.ID:
			return fmt.Errorf("accel: dispatch %d: kernel %s: bindings live on different devices (%s and %s)", i, d.Binding.kernel.config.Name, device.ID, d.Binding.device.ID)
		}
		inner[i] = wgpu.Dispatch{Binding: d.Binding.inner, Params: d.Params, X: d.Workgroups[0], Y: d.Workgroups[1], Z: d.Workgroups[2]}
	}
	if err := wgpu.Submit(inner); err != nil {
		return residentFailure(fmt.Errorf("accel: %w", err))
	}
	return nil
}

func innerBuffers(kernel, role string, buffers []*DeviceBuffer, device **Device) ([]*wgpu.Buffer, error) {
	out := make([]*wgpu.Buffer, len(buffers))
	for i, b := range buffers {
		switch {
		case b == nil || b.inner == nil:
			return nil, fmt.Errorf("accel: kernel %s: %s buffer %d is nil", kernel, role, i)
		case b.inner.Released():
			return nil, fmt.Errorf("accel: kernel %s: %s buffer %d was released", kernel, role, i)
		case b.device.ID == "":
			return nil, fmt.Errorf("accel: kernel %s: %s buffer %d is on the host; a kernel runs on device buffers only", kernel, role, i)
		case *device == nil:
			d := b.device
			*device = &d
		case (*device).ID != b.device.ID:
			return nil, fmt.Errorf("accel: kernel %s: buffers live on different devices (%s and %s)", kernel, (*device).ID, b.device.ID)
		}
		out[i] = b.inner
	}
	return out, nil
}

// NewBitsBuffer places bits on device as they are, for indices, topology or
// any 32-bit data a kernel reads as u32 or i32. The buffer is the same type
// NewDeviceBuffer returns; BitsToHost reads it back unchanged.
func NewBitsBuffer(device Device, bits []uint32) (*DeviceBuffer, error) {
	if device.ID == "" {
		return nil, errors.New("accel: NewBitsBuffer needs a device; the host holds the slice itself")
	}
	bytes := uint64(len(bits)) * 4
	if err := reserveResident(device, bytes); err != nil {
		return nil, err
	}
	inner, err := wgpu.UploadBits(bits)
	if err != nil {
		unreserveResident(device, bytes)
		return nil, residentFailure(err)
	}
	return finalizing(&DeviceBuffer{inner: inner, device: device, bytes: bytes}), nil
}

// BitsToHost copies the buffer back as bit patterns, waiting for every
// earlier kernel that wrote to it.
func (b *DeviceBuffer) BitsToHost() ([]uint32, error) {
	bits, err := b.inner.DownloadBits(context.Background())
	return bits, residentFailure(err)
}

// Write overwrites the buffer with values, which must be as many as it holds.
// It first waits for every earlier kernel to finish, so a kernel never sees a
// half-written buffer; a loop that rewrites inputs every step therefore waits
// once per call. WriteBits is the same for bit patterns.
func (b *DeviceBuffer) Write(values []float32) error {
	return residentFailure(b.inner.Write(values))
}

// WriteBits overwrites the buffer with bit patterns, with the wait Write has.
func (b *DeviceBuffer) WriteBits(bits []uint32) error {
	return residentFailure(b.inner.WriteBits(bits))
}

// Zero sets every word of the buffer to zero, with the wait Write has.
func (b *DeviceBuffer) Zero() error {
	return residentFailure(b.inner.Zero())
}

// DeviceLimits is what a device allows a kernel.
type DeviceLimits struct {
	// MaxStorageBuffers is how many storage buffers one kernel may bind.
	MaxStorageBuffers int
	// MaxBufferBytes is the largest buffer a kernel may bind.
	MaxBufferBytes uint64
	// MaxWorkgroupsPerDimension is the largest workgroup count in each of x, y and z.
	MaxWorkgroupsPerDimension uint32
	// MaxInvocationsPerWorkgroup is the most invocations one workgroup may have,
	// the product of the sizes in its @workgroup_size.
	MaxInvocationsPerWorkgroup uint32
}

// ResidentLimits reports what the resident device allows a kernel. The
// numbers belong to the one device resident buffers live on.
func ResidentLimits() (DeviceLimits, error) {
	limits, err := wgpu.DeviceLimits()
	if err != nil {
		return DeviceLimits{}, residentFailure(err)
	}
	return DeviceLimits(limits), nil
}
