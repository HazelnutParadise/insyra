package nn

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/accel"
)

// Device reports where the tensor lives. The zero Device is the host.
func (t *Tensor) Device() accel.Device {
	if t == nil || t.device == nil {
		return accel.Device{}
	}
	return t.device.Device()
}

// onDevice reports whether the tensor's values live on a device.
func (t *Tensor) onDevice() bool { return t != nil && t.device != nil }

// To returns a copy of a tensor on device. Float32 tensors and int64 label
// tensors can move. Int64 values are held as int32 on the device, so a value
// that is negative or does not fit is an error naming it; such a tensor can be
// used only as the labels of SoftmaxCrossEntropy. The zero Device stands for
// the host, so code written against accel.DefaultDevice runs unchanged on a
// machine without a GPU. Moving never changes the source tensor, and values
// cross bit for bit. A tensor already on another device is moved through the
// host with ToHost, which keeps every transfer explicit.
//
// The copy owns device memory until Release is called. A tape releases the
// tensors it creates itself on Clear and Close, but not the ones a caller
// hands it.
func (t *Tensor) To(device accel.Device) (*Tensor, error) {
	if t == nil {
		return nil, fmt.Errorf("tensor is nil")
	}
	if t.dtype != DTypeFloat32 && t.dtype != DTypeInt64 {
		return nil, fmt.Errorf("tensor has unsupported dtype %s; only float32 and int64 labels can move to a device", dtypeName(t.dtype))
	}
	if device.ID == "" {
		return t.ToHost()
	}
	if t.device != nil {
		if t.device.Device().ID != device.ID {
			return nil, fmt.Errorf("tensor lives on device %s; move it to the host with ToHost before moving it to %s", t.device.Device().ID, device.ID)
		}
		return t.cloneResident()
	}
	if t.dtype == DTypeInt64 {
		buffer, largest, err := accel.NewLabelBuffer(device, t.int64Data)
		if err != nil {
			return nil, err
		}
		moved, err := newResidentTensor(t.shape, buffer)
		if err != nil {
			return nil, err
		}
		moved.dtype = DTypeInt64
		moved.maxLabel = largest
		return moved, nil
	}
	buffer, err := accel.NewDeviceBuffer(device, t.data)
	if err != nil {
		return nil, err
	}
	return newResidentTensor(t.shape, buffer)
}

// ToHost returns a copy of the tensor on the host. It waits for every kernel
// that wrote the tensor, and it is the only way a resident tensor's values
// are read.
func (t *Tensor) ToHost() (*Tensor, error) {
	if t == nil {
		return nil, fmt.Errorf("tensor is nil")
	}
	if t.device == nil {
		return copyTensor(t)
	}
	if t.dtype == DTypeInt64 {
		values, err := t.device.LabelsToHost()
		if err != nil {
			return nil, err
		}
		return newInt64Tensor(t.shape, values)
	}
	values, err := t.device.ToHost()
	if err != nil {
		return nil, err
	}
	return newFloat32Tensor(t.shape, values)
}

// Release frees the device memory of a resident tensor. It does nothing to a
// host tensor and is safe to call more than once. A released tensor cannot be
// read or used in an operation.
func (t *Tensor) Release() {
	if t == nil || t.device == nil {
		return
	}
	t.device.Release()
}

func (t *Tensor) cloneResident() (*Tensor, error) {
	buffer, err := t.device.Clone()
	if err != nil {
		return nil, err
	}
	clone, err := newResidentTensor(t.shape, buffer)
	if err != nil {
		return nil, err
	}
	clone.dtype, clone.maxLabel = t.dtype, t.maxLabel
	return clone, nil
}

func newResidentTensor(shape []int, buffer *accel.DeviceBuffer) (*Tensor, error) {
	shapeCopy, strides, count, err := makeLayout(shape)
	if err != nil {
		buffer.Release()
		return nil, err
	}
	if buffer.Len() != count {
		buffer.Release()
		return nil, fmt.Errorf("device buffer holds %d values, want %d for shape %v", buffer.Len(), count, shapeCopy)
	}
	return &Tensor{dtype: DTypeFloat32, shape: shapeCopy, strides: strides, device: buffer}, nil
}

// requireFloat32Any accepts a float32 tensor wherever it lives.
func requireFloat32Any(t *Tensor, operand string) error {
	if t == nil {
		return fmt.Errorf("%s tensor is nil", operand)
	}
	if t.dtype != DTypeFloat32 {
		return fmt.Errorf("%s has unsupported dtype %s", operand, dtypeName(t.dtype))
	}
	return nil
}

func residentRefusal(t *Tensor, operand string) error {
	return fmt.Errorf("%s is on device %s; call ToHost to read it on the host", operand, t.device.Device().ID)
}

// rejectResident returns an error naming ToHost when any of tensors is on a
// device. The host kernels read host slices, which a resident tensor does not
// have, so they refuse it up front instead of reading nothing.
func rejectResident(operation string, tensors ...*Tensor) error {
	for _, tensor := range tensors {
		if tensor != nil && tensor.device != nil {
			return residentRefusal(tensor, operation+" input")
		}
	}
	return nil
}

// DeviceBuffer lends the device buffer of a resident float32 tensor, so a
// kernel of your own can read or write it in place (accel.Submit). The tensor
// keeps owning the buffer: releasing the tensor releases it, and the buffer
// must not be released separately. A host tensor, and an int64 tensor, which
// a device holds as int32 class indices, are refused.
func (t *Tensor) DeviceBuffer() (*accel.DeviceBuffer, error) {
	switch {
	case t == nil:
		return nil, fmt.Errorf("tensor is nil")
	case t.device == nil:
		return nil, fmt.Errorf("tensor is on the host; move it with To first")
	case t.dtype != DTypeFloat32:
		return nil, fmt.Errorf("tensor has dtype %s; only a float32 tensor lends its device buffer", dtypeName(t.dtype))
	}
	return t.device, nil
}

// NewDeviceTensor wraps buffer, a float32 buffer on a device, as a tensor of
// shape. The tensor owns the buffer from then on, so only give it a buffer you
// made yourself and no longer release: a buffer lent by another tensor's
// DeviceBuffer would be released twice. On an error the buffer stays yours.
func NewDeviceTensor(shape []int, buffer *accel.DeviceBuffer) (*Tensor, error) {
	switch {
	case buffer == nil:
		return nil, fmt.Errorf("device buffer is nil")
	case buffer.Device().ID == "":
		return nil, fmt.Errorf("device buffer is on the host; a device tensor needs a buffer on a device")
	case buffer.Released():
		return nil, fmt.Errorf("device buffer was released")
	}
	if _, _, count, err := makeLayout(shape); err != nil {
		return nil, err
	} else if buffer.Len() != count {
		return nil, fmt.Errorf("device buffer holds %d values, want %d for shape %v", buffer.Len(), count, shape)
	}
	return newResidentTensor(shape, buffer)
}

// ownResidentGradients wraps the vjp of a Custom operation on a device so that
// every gradient it returns is a float32 tensor on the same device as the
// output, which the tape releases at Clear.
func (t *Tape) ownResidentGradients(name string, output *Tensor, vjp func(upstream *Tensor) ([]*Tensor, error)) func(upstream *Tensor) ([]*Tensor, error) {
	device := output.Device().ID
	return func(upstream *Tensor) ([]*Tensor, error) {
		gradients, err := vjp(upstream)
		if err != nil {
			return nil, err
		}
		// Own every gradient first, so a refusal below leaves none behind.
		for _, gradient := range gradients {
			t.own(gradient)
		}
		for index, gradient := range gradients {
			if gradient != nil && (gradient.device == nil || gradient.Device().ID != device) {
				return nil, fmt.Errorf("tape custom %s: gradient %d is not on the device %s of the output; the vjp of an operation on a device returns tensors on that device", name, index, device)
			}
		}
		return gradients, nil
	}
}
