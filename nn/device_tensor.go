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

// To returns a copy of a float32 tensor on device. The zero Device stands for
// the host, so code written against accel.DefaultDevice runs unchanged on a
// machine without a GPU. Moving never changes the source tensor, and values
// cross bit for bit. A tensor already on another device is moved through the
// host with ToHost, which keeps every transfer explicit.
//
// The copy owns device memory until Release is called. A tape releases the
// tensors it creates itself on Clear and Close, but not the ones a caller
// hands it.
func (t *Tensor) To(device accel.Device) (*Tensor, error) {
	if err := requireFloat32Any(t, "tensor"); err != nil {
		return nil, err
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
	if err := requireFloat32Any(t, "tensor"); err != nil {
		return nil, err
	}
	if t.device == nil {
		return copyTensor(t)
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
	return newResidentTensor(t.shape, buffer)
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
