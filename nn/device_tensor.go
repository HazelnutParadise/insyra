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
