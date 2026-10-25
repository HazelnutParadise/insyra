package wgpu

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	gowgpu "github.com/gogpu/wgpu"
)

// ErrDeviceLost reports that the GPU device was lost: the driver reset it, the
// operating system stopped a dispatch that ran too long (Windows does this to
// a dispatch of about two seconds or more), or the adapter was removed. Every
// buffer, binding and pipeline made on that device is gone. The next call
// that needs a device opens a new one; a Buffer or Binding from before the
// loss stays unusable and reports this error, so its owner re-creates it.
//
// This is the recovery WebGPU specifies for GPUDevice.lost: a lost device
// never works again and the application requests a new adapter and device
// (https://www.w3.org/TR/webgpu/#device-lost).
var ErrDeviceLost = errors.New("wgpu: the GPU device was lost")

// deviceLost reports whether err says the device is gone. gogpu/wgpu v0.34.5
// returns a Vulkan VK_ERROR_DEVICE_LOST from Queue.Submit as a plain formatted
// string ("vkQueueSubmit failed: -4") that does not wrap its own ErrDeviceLost,
// so the text is checked as well as the sentinel.
func deviceLost(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gowgpu.ErrDeviceLost) {
		return true
	}
	text := err.Error()
	for _, marker := range []string{
		"vkQueueSubmit failed: -4",
		"VK_ERROR_DEVICE_LOST",
		"DXGI_ERROR_DEVICE_REMOVED",
		"DXGI_ERROR_DEVICE_RESET",
		"device lost",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// failed turns a device-loss error into ErrDeviceLost after retiring the
// device. Any other error comes back unchanged. submitMu must be held.
func (h *handle) failed(err error) error {
	if !deviceLost(err) {
		return err
	}
	retireLocked(h)
	return fmt.Errorf("%w: %w", ErrDeviceLost, err)
}

// Everything made on the shared device that outlives one call. All three are
// guarded by submitMu. retireLocked walks them to release what the lost device
// still holds, because gogpu cannot create a new device while the old one has
// live children (vkCreateDevice failed: -3 on a GT 1030 until every buffer,
// bind group and pipeline of the lost device was released).
var (
	liveBuffers  = map[*Buffer]struct{}{}
	liveBindings = map[*Binding]struct{}{}
	builtKernels = map[*kernel]struct{}{}
)

func releaseQuietly(resource interface{ Release() }) {
	defer func() { _ = recover() }()
	resource.Release()
}

// retireLocked drops the lost device: it releases every object made on it,
// marks the buffers and bindings that referred to it lost, and forgets the
// shared handle so the next acquire opens a new device. It is idempotent per
// handle. submitMu must be held.
func retireLocked(h *handle) {
	if h == nil || h.retired {
		return
	}
	h.retired = true

	var objects []interface{ Release() }
	for b := range liveBuffers {
		b.mu.Lock()
		if b.buffer != nil {
			objects = append(objects, b.buffer)
			b.buffer = nil
		}
		b.released, b.lost = true, true
		b.mu.Unlock()
	}
	liveBuffers = map[*Buffer]struct{}{}
	for b := range liveBindings {
		if b.group != nil {
			objects = append(objects, b.group)
			b.group = nil
		}
		b.released, b.lost = true, true
	}
	liveBindings = map[*Binding]struct{}{}
	for k := range builtKernels {
		if k.pipe != nil {
			objects = append(objects, k.pipe)
		}
		if k.layout != nil {
			objects = append(objects, k.layout)
		}
		k.reset()
	}
	builtKernels = map[*kernel]struct{}{}
	for _, resource := range pending {
		objects = append(objects, resource)
	}
	pending = nil
	if uniformRing != nil {
		objects = append(objects, uniformRing)
		uniformRing, uniformNext = nil, 0
	}
	if h.shortlistPipe != nil {
		objects = append(objects, h.shortlistPipe)
	}
	if h.shortlistBGLayout != nil {
		objects = append(objects, h.shortlistBGLayout)
	}
	if h.matmulPipe != nil {
		objects = append(objects, h.matmulPipe)
	}
	if h.matmulBGLayout != nil {
		objects = append(objects, h.matmulBGLayout)
	}
	for _, resource := range objects {
		releaseQuietly(resource)
	}
	if h.device != nil {
		releaseQuietly(h.device)
	}
	if h.adapter != nil {
		releaseQuietly(h.adapter)
	}
	if h.instance != nil {
		releaseQuietly(h.instance)
	}

	sharedMu.Lock()
	if shared == h {
		shared, sharedErr, sharedTried = nil, nil, false
	}
	sharedMu.Unlock()
}

// Lost reports whether the device was lost after the buffer was made. The
// buffer's contents are gone with it.
func (b *Buffer) Lost() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.lost
}

// unusableLocked is the error for a buffer that cannot be used: lost with the
// device, or released. b.mu must be held.
func (b *Buffer) unusableLocked(what string) error {
	if b.lost {
		return fmt.Errorf("%w: %s", ErrDeviceLost, what)
	}
	return fmt.Errorf("wgpu: %s was released", what)
}

func (k *kernel) reset() {
	k.once = sync.Once{}
	k.pipe, k.layout, k.err = nil, nil, nil
}
