package accel

import (
	"context"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/accel/internal/wgpu"
)

// DeviceBuffer is a float32 array that lives on a device. It is what lets a
// training step keep its tensors on the GPU: kernels read and write it
// without a host copy, and ToHost is the only way its values come back. The
// device arithmetic that produces its contents is the PrecisionFloat32Rounded
// contract: each result is the exact value of its operation rounded once to
// nearest-even float32, and an exactly-zero result is +0.
//
// A DeviceBuffer must be released with Release, which is idempotent. The
// bytes it holds count against the device's memory budget until then.
type DeviceBuffer struct {
	inner  *wgpu.Buffer
	device Device
	bytes  uint64
}

var (
	residentMu    sync.Mutex
	residentBytes = map[string]uint64{}
)

// ErrDeviceMemoryBudget reports that a resident allocation would exceed the
// memory budget of its device.
var ErrDeviceMemoryBudget = errors.New("accel: resident allocation exceeds the device memory budget")

// ResidentDevice returns the device resident buffers live on, or an error
// whose FallbackReason says why there is none. It honours Config's
// acceleration switch, the session's device selection and strict GPU mode.
func ResidentDevice() (Device, error) {
	if !insyra.Config.GetAccelerationEnabled() {
		return Device{}, &ResidentError{Reason: FallbackReasonCPUOnly, Cause: errors.New("acceleration is disabled by Config")}
	}
	session := Default()
	if wgpuDisabled() {
		return Device{}, &ResidentError{Reason: FallbackReasonNoAccelerator, Cause: errors.New("the device backend is disabled")}
	}
	devices := session.Devices()
	if len(devices) == 0 {
		reason := session.Report().FallbackReason
		if reason == FallbackReasonNone {
			reason = FallbackReasonNoAccelerator
		}
		return Device{}, &ResidentError{Reason: reason, Cause: fmt.Errorf("no usable accelerator (%s)", reason)}
	}
	// Discovery can list a device through a native probe that the WebGPU
	// backend, which holds every resident buffer, cannot open. Offering that
	// device would make the first To fail, so ask the backend itself.
	if _, err := wgpu.Probe(); err != nil {
		return Device{}, &ResidentError{Reason: FallbackReasonNoAccelerator, Cause: fmt.Errorf("the resident backend cannot open a device: %w", err)}
	}
	return devices[0], nil
}

// DefaultDevice returns the device the process session selected, or the zero
// Device, which stands for the host, when there is none: acceleration is
// disabled, no accelerator was found, or strict mode found nothing to use.
// The reason is recorded in the session's report. A program written against
// DefaultDevice runs unchanged with or without a GPU and returns the same
// numbers in PrecisionFloat32Rounded.
func DefaultDevice() Device {
	device, err := ResidentDevice()
	if err != nil {
		var resident *ResidentError
		if errors.As(err, &resident) {
			RecordResidentFallback("device selection", resident.Reason, resident.Cause)
		}
		return Device{}
	}
	return device
}

// ResidentError is an error from the resident path that carries the reason
// the work did not run on the device, so a caller can fall back in the open.
type ResidentError struct {
	Reason FallbackReason
	Cause  error
}

func (e *ResidentError) Error() string {
	return fmt.Sprintf("accel resident (%s): %v", e.Reason, e.Cause)
}

// Unwrap returns the underlying error.
func (e *ResidentError) Unwrap() error { return e.Cause }

func residentFailure(err error) error {
	if err == nil {
		return nil
	}
	var existing *ResidentError
	if errors.As(err, &existing) {
		return err
	}
	return &ResidentError{Reason: deviceMatMulFallbackReason(err), Cause: err}
}

// RecordResidentFallback notes in the process session's report that work on
// resident data ran on the host instead, and why. The operation name is
// logged at debug level, and the first qualifying fallback at info level.
func RecordResidentFallback(operation string, reason FallbackReason, cause error) {
	session := Default()
	_ = recordDeviceMatMulFallback(session, reason, 0, cause)
	insyra.LogDebug("accel", "resident", "operation=%s ran on the host: %s: %v", operation, reason, cause)
}

// NewDeviceBuffer uploads values to device.
func NewDeviceBuffer(device Device, values []float32) (*DeviceBuffer, error) {
	bytes := uint64(len(values)) * 4
	if err := reserveResident(device, bytes); err != nil {
		return nil, err
	}
	inner, err := wgpu.Upload(values)
	if err != nil {
		unreserveResident(device, bytes)
		return nil, residentFailure(err)
	}
	return finalizing(&DeviceBuffer{inner: inner, device: device, bytes: bytes}), nil
}

// finalizing registers the backstop that frees a buffer nobody released. It
// is a safety net, not a lifetime policy: device memory is not reclaimed on
// the garbage collector's schedule, so the warning says to call Release.
func finalizing(b *DeviceBuffer) *DeviceBuffer {
	runtime.SetFinalizer(b, func(buffer *DeviceBuffer) {
		if buffer.inner != nil && !buffer.inner.Released() {
			insyra.LogWarning("accel", "resident", "a device buffer of %d bytes was garbage collected without Release", buffer.bytes)
			buffer.Release()
		}
	})
	return b
}

func reserveResident(device Device, bytes uint64) error {
	residentMu.Lock()
	defer residentMu.Unlock()
	if device.BudgetBytes > 0 && residentBytes[device.ID]+bytes > device.BudgetBytes {
		return &ResidentError{Reason: FallbackReasonBufferTooLarge,
			Cause: fmt.Errorf("%w: %d resident + %d requested > %d bytes on %s", ErrDeviceMemoryBudget, residentBytes[device.ID], bytes, device.BudgetBytes, device.ID)}
	}
	residentBytes[device.ID] += bytes
	return nil
}

func unreserveResident(device Device, bytes uint64) {
	residentMu.Lock()
	defer residentMu.Unlock()
	if residentBytes[device.ID] >= bytes {
		residentBytes[device.ID] -= bytes
	} else {
		residentBytes[device.ID] = 0
	}
}

// ResidentBytes reports how many bytes of resident buffers are live on device.
func ResidentBytes(device Device) uint64 {
	residentMu.Lock()
	defer residentMu.Unlock()
	return residentBytes[device.ID]
}

func (b *DeviceBuffer) wrap(inner *wgpu.Buffer) (*DeviceBuffer, error) {
	bytes := uint64(inner.Len()) * 4
	if err := reserveResident(b.device, bytes); err != nil {
		inner.Release()
		return nil, err
	}
	return finalizing(&DeviceBuffer{inner: inner, device: b.device, bytes: bytes}), nil
}

// Device reports the device the buffer lives on.
func (b *DeviceBuffer) Device() Device { return b.device }

// Len reports how many float32 values the buffer holds.
func (b *DeviceBuffer) Len() int { return b.inner.Len() }

// Released reports whether Release has been called.
func (b *DeviceBuffer) Released() bool { return b == nil || b.inner == nil || b.inner.Released() }

// ToHost copies the values back, waiting for every earlier kernel that wrote
// to the buffer. It is the only way values leave the device.
func (b *DeviceBuffer) ToHost() ([]float32, error) {
	values, err := b.inner.Download(context.Background())
	return values, residentFailure(err)
}

// Release frees the device memory. It is idempotent.
func (b *DeviceBuffer) Release() {
	if b == nil || b.inner == nil {
		return
	}
	if !b.inner.Released() {
		unreserveResident(b.device, b.bytes)
	}
	b.inner.Release()
}

func sameDevice(a, b *DeviceBuffer) error {
	if a == nil || b == nil {
		return errors.New("accel: device buffer is nil")
	}
	if a.device.ID != b.device.ID {
		return fmt.Errorf("accel: buffers live on different devices (%s and %s)", a.device.ID, b.device.ID)
	}
	return nil
}

// RoundedMatMul multiplies two resident matrices, reading a as its transpose
// when transA and b as its transpose when transB. a holds [m,k] (or [k,m]) and
// b holds [k,n] (or [n,k]); the result holds [m,n]. Every element is the exact
// dot product rounded once to float32.
func RoundedMatMul(a, b *DeviceBuffer, m, k, n int, transA, transB bool) (*DeviceBuffer, error) {
	if err := sameDevice(a, b); err != nil {
		return nil, err
	}
	out, err := wgpu.MatMulRounded(a.inner, b.inner, m, k, n, transA, transB)
	if err != nil {
		return nil, residentFailure(err)
	}
	return a.wrap(out)
}

// RoundedAdd adds two resident arrays. b may be shorter than a, in which case
// it repeats over a's trailing dimension, so a bias row broadcasts over a
// matrix. Each sum is rounded once.
func RoundedAdd(a, b *DeviceBuffer) (*DeviceBuffer, error) {
	if err := sameDevice(a, b); err != nil {
		return nil, err
	}
	out, err := wgpu.AddRounded(a.inner, b.inner)
	if err != nil {
		return nil, residentFailure(err)
	}
	return a.wrap(out)
}

// RoundedSumRows sums a resident [m,n] matrix over its rows into n values,
// each the exact sum rounded once.
func RoundedSumRows(x *DeviceBuffer, m, n int) (*DeviceBuffer, error) {
	if x == nil {
		return nil, errors.New("accel: device buffer is nil")
	}
	out, err := wgpu.SumRowsRounded(x.inner, m, n)
	if err != nil {
		return nil, residentFailure(err)
	}
	return x.wrap(out)
}

// RoundedSGD updates w in place to w - rate*g, each element the exact value
// rounded once.
func RoundedSGD(w, g *DeviceBuffer, rate float32) error {
	if err := sameDevice(w, g); err != nil {
		return err
	}
	return residentFailure(wgpu.SGDRounded(w.inner, g.inner, math.Float32bits(rate)^0x80000000))
}

// Clone copies the buffer on the device. The copy is a separate buffer with
// its own lifetime.
func (b *DeviceBuffer) Clone() (*DeviceBuffer, error) {
	out, err := wgpu.Clone(b.inner)
	if err != nil {
		return nil, residentFailure(err)
	}
	return b.wrap(out)
}

// SynchronizeResident waits until every kernel submitted so far has finished.
func SynchronizeResident() error { return residentFailure(wgpu.Synchronize()) }

// AdamScalars are the float32 constants of one Adam or AdamW step. The caller
// computes each once on the host and rounds it once to float32, so the host
// and the device read the same bits.
type AdamScalars struct {
	// Beta1, Beta2 and their complements 1-Beta1 and 1-Beta2.
	Beta1, OneMinusBeta1, Beta2, OneMinusBeta2 float32
	// Bias1 and Bias2 are the bias corrections 1-Beta1^t and 1-Beta2^t.
	Bias1, Bias2 float32
	// Epsilon is added to the square root of the corrected second moment.
	Epsilon float32
	// NegRate is -lr. NegDecay is -(lr*weightDecay), used when Decay is true
	// for AdamW's decoupled decay.
	NegRate, NegDecay float32
	Decay             bool
}

// RoundedAdam applies one Adam or AdamW step in place to w, m and v from the
// gradient g, all on the device. Every operation is correctly rounded or an
// exact two-term sum rounded once, so the host definition in package nn
// returns the same bits.
func RoundedAdam(w, g, m, v *DeviceBuffer, s AdamScalars) error {
	for _, b := range []*DeviceBuffer{g, m, v} {
		if err := sameDevice(w, b); err != nil {
			return err
		}
	}
	return residentFailure(wgpu.AdamRounded(w.inner, g.inner, m.inner, v.inner, wgpu.AdamScalars(s)))
}

// NewZeroDeviceBuffer allocates a zero-filled buffer of length values on device.
func NewZeroDeviceBuffer(device Device, length int) (*DeviceBuffer, error) {
	return NewDeviceBuffer(device, make([]float32, length))
}

// RoundedExp computes e^x for each element of a resident array, correctly
// rounded: the exact value rounded once to nearest-even float32.
func RoundedExp(x *DeviceBuffer) (*DeviceBuffer, error) {
	if x == nil {
		return nil, errors.New("accel: device buffer is nil")
	}
	out, err := wgpu.ExpRounded(x.inner)
	if err != nil {
		return nil, residentFailure(err)
	}
	return x.wrap(out)
}

// RoundedLog computes the natural logarithm of each element of a resident
// array, correctly rounded.
func RoundedLog(x *DeviceBuffer) (*DeviceBuffer, error) {
	if x == nil {
		return nil, errors.New("accel: device buffer is nil")
	}
	out, err := wgpu.LogRounded(x.inner)
	if err != nil {
		return nil, residentFailure(err)
	}
	return x.wrap(out)
}

// RoundedExpm1 computes e^x - 1 for each element of a resident array,
// correctly rounded.
func RoundedExpm1(x *DeviceBuffer) (*DeviceBuffer, error) {
	if x == nil {
		return nil, errors.New("accel: device buffer is nil")
	}
	out, err := wgpu.Expm1Rounded(x.inner)
	if err != nil {
		return nil, residentFailure(err)
	}
	return x.wrap(out)
}

// RoundedLog1p computes log(1+x) for each element of a resident array,
// correctly rounded.
func RoundedLog1p(x *DeviceBuffer) (*DeviceBuffer, error) {
	if x == nil {
		return nil, errors.New("accel: device buffer is nil")
	}
	out, err := wgpu.Log1pRounded(x.inner)
	if err != nil {
		return nil, residentFailure(err)
	}
	return x.wrap(out)
}

// SoftmaxCrossEntropyState holds what the gradient of a cross-entropy loss
// reuses from its forward pass, so each exponential is computed once. It
// occupies device memory until Release is called.
type SoftmaxCrossEntropyState struct {
	inner  *wgpu.CEState
	device Device
	bytes  uint64
}

// Release frees the state. It is idempotent.
func (s *SoftmaxCrossEntropyState) Release() {
	if s == nil || s.inner == nil {
		return
	}
	unreserveResident(s.device, s.bytes)
	s.inner.Release()
	s.inner = nil
}

// RoundedSoftmaxCrossEntropy returns the mean cross-entropy loss, one value,
// of resident logits [rows,classes] against resident labels: rows int32 class
// indices, each in [0,classes), uploaded with NewLabelBuffer. Every step is one
// correctly rounded operation or an exact sum rounded once. The state it
// returns feeds RoundedSoftmaxCrossEntropyGrad.
func RoundedSoftmaxCrossEntropy(logits, labels *DeviceBuffer, rows, classes int) (*DeviceBuffer, *SoftmaxCrossEntropyState, error) {
	if err := sameDevice(logits, labels); err != nil {
		return nil, nil, err
	}
	stateBytes := uint64(rows*classes+rows) * 4
	if err := reserveResident(logits.device, stateBytes); err != nil {
		return nil, nil, err
	}
	out, state, err := wgpu.SoftmaxCrossEntropyRounded(logits.inner, labels.inner, rows, classes)
	if err != nil {
		unreserveResident(logits.device, stateBytes)
		return nil, nil, residentFailure(err)
	}
	loss, err := logits.wrap(out)
	if err != nil {
		unreserveResident(logits.device, stateBytes)
		state.Release()
		return nil, nil, err
	}
	return loss, &SoftmaxCrossEntropyState{inner: state, device: logits.device, bytes: stateBytes}, nil
}

// RoundedSoftmaxCrossEntropyGrad returns the gradient of the loss with respect
// to the logits, scaled by the one-value upstream gradient.
func RoundedSoftmaxCrossEntropyGrad(state *SoftmaxCrossEntropyState, labels, upstream *DeviceBuffer, rows, classes int) (*DeviceBuffer, error) {
	if state == nil || state.inner == nil {
		return nil, errors.New("accel: cross-entropy state was released")
	}
	if err := sameDevice(labels, upstream); err != nil {
		return nil, err
	}
	out, err := wgpu.SoftmaxCrossEntropyGradRounded(state.inner, labels.inner, upstream.inner, rows, classes)
	if err != nil {
		return nil, residentFailure(err)
	}
	return labels.wrap(out)
}

// SoftLossState holds what the gradient of a soft-target loss reuses from its
// forward pass. It occupies device memory until Release is called.
type SoftLossState struct {
	inner  *wgpu.SoftState
	device Device
	bytes  uint64
}

// Release frees the state. It is idempotent.
func (s *SoftLossState) Release() {
	if s == nil || s.inner == nil {
		return
	}
	unreserveResident(s.device, s.bytes)
	s.inner.Release()
	s.inner = nil
}

func softLoss(a, b *DeviceBuffer, rows, classes int, withTeacher bool, run func() (*wgpu.Buffer, *wgpu.SoftState, error)) (*DeviceBuffer, *SoftLossState, error) {
	if err := sameDevice(a, b); err != nil {
		return nil, nil, err
	}
	elements := uint64(rows) * uint64(classes)
	stateBytes := (elements + uint64(rows)) * 4
	if withTeacher {
		stateBytes += elements * 4
	}
	if err := reserveResident(a.device, stateBytes); err != nil {
		return nil, nil, err
	}
	out, state, err := run()
	if err != nil {
		unreserveResident(a.device, stateBytes)
		return nil, nil, residentFailure(err)
	}
	loss, err := a.wrap(out)
	if err != nil {
		unreserveResident(a.device, stateBytes)
		state.Release()
		return nil, nil, err
	}
	return loss, &SoftLossState{inner: state, device: a.device, bytes: stateBytes}, nil
}

// RoundedSoftTargetCrossEntropy returns the mean cross-entropy loss, one value,
// of resident logits [rows,classes] against resident probability rows. Every
// step is one correctly rounded operation or an exact sum rounded once. The
// state feeds RoundedSoftTargetCrossEntropyGrad.
func RoundedSoftTargetCrossEntropy(logits, targets *DeviceBuffer, rows, classes int) (*DeviceBuffer, *SoftLossState, error) {
	return softLoss(logits, targets, rows, classes, false, func() (*wgpu.Buffer, *wgpu.SoftState, error) {
		return wgpu.SoftTargetCrossEntropyRounded(logits.inner, targets.inner, rows, classes)
	})
}

// RoundedSoftTargetCrossEntropyGrad returns the gradient with respect to the
// logits, scaled by the one-value upstream gradient.
func RoundedSoftTargetCrossEntropyGrad(state *SoftLossState, targets, upstream *DeviceBuffer, rows, classes int) (*DeviceBuffer, error) {
	if state == nil || state.inner == nil {
		return nil, errors.New("accel: soft-target state was released")
	}
	if err := sameDevice(targets, upstream); err != nil {
		return nil, err
	}
	out, err := wgpu.SoftTargetCrossEntropyGradRounded(state.inner, targets.inner, upstream.inner, rows, classes)
	if err != nil {
		return nil, residentFailure(err)
	}
	return targets.wrap(out)
}

// RoundedDistillation returns the mean cross-entropy of resident student
// logits against the softmax of resident teacher logits.
func RoundedDistillation(student, teacher *DeviceBuffer, rows, classes int) (*DeviceBuffer, *SoftLossState, error) {
	return softLoss(student, teacher, rows, classes, true, func() (*wgpu.Buffer, *wgpu.SoftState, error) {
		return wgpu.DistillationRounded(student.inner, teacher.inner, rows, classes)
	})
}

// RoundedDistillationGrad returns the gradient with respect to the student's
// logits, scaled by the one-value upstream gradient.
func RoundedDistillationGrad(state *SoftLossState, student, teacher, upstream *DeviceBuffer, rows, classes int) (*DeviceBuffer, error) {
	if state == nil || state.inner == nil {
		return nil, errors.New("accel: distillation state was released")
	}
	if err := sameDevice(student, teacher); err != nil {
		return nil, err
	}
	if err := sameDevice(student, upstream); err != nil {
		return nil, err
	}
	out, err := wgpu.DistillationGradRounded(state.inner, student.inner, teacher.inner, upstream.inner, rows, classes)
	if err != nil {
		return nil, residentFailure(err)
	}
	return student.wrap(out)
}

// NewLabelBuffer uploads class indices to device as int32. A negative index or
// one above the int32 range is an error naming it; the largest index is
// returned so the caller can check it against the class count without reading
// the buffer back.
func NewLabelBuffer(device Device, labels []int64) (*DeviceBuffer, int64, error) {
	values := make([]float32, len(labels))
	var largest int64 = -1
	for i, label := range labels {
		if label < 0 || label > math.MaxInt32 {
			return nil, 0, fmt.Errorf("accel: label %d at index %d does not fit a non-negative int32", label, i)
		}
		if label > largest {
			largest = label
		}
		values[i] = math.Float32frombits(uint32(label))
	}
	buffer, err := NewDeviceBuffer(device, values)
	return buffer, largest, err
}

// LabelsToHost reads a label buffer back as class indices.
func (b *DeviceBuffer) LabelsToHost() ([]int64, error) {
	values, err := b.ToHost()
	if err != nil {
		return nil, err
	}
	out := make([]int64, len(values))
	for i, v := range values {
		out[i] = int64(math.Float32bits(v))
	}
	return out, nil
}
