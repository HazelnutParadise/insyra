package wgpu

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestDeviceLostIsRecognisedFromEveryBackendsText(t *testing.T) {
	for _, text := range []string{
		"wgpu: submit failed: vulkan: vkQueueSubmit failed: -4",
		"VK_ERROR_DEVICE_LOST",
		"DXGI_ERROR_DEVICE_REMOVED (0x887a0005)",
		"wgpu: device lost during map",
	} {
		if !deviceLost(errors.New(text)) {
			t.Errorf("%q is not recognised as a lost device", text)
		}
	}
	for _, text := range []string{
		"vulkan: vkQueueSubmit failed: -1",
		"wgpu: buffer was released",
		"context deadline exceeded",
	} {
		if deviceLost(errors.New(text)) {
			t.Errorf("%q is recognised as a lost device", text)
		}
	}
	if deviceLost(nil) {
		t.Error("nil is a lost device")
	}
}

// Losing the device marks every buffer and binding made on it lost, forgets the
// shared handle so the next acquire opens a new one, and leaves a buffer that
// was made after the loss alone. It needs no GPU: the handle is empty.
func TestRetiringADeviceMarksItsBuffersLost(t *testing.T) {
	submitMu.Lock()
	defer submitMu.Unlock()

	before := &Buffer{length: 3}
	liveBuffers[before] = struct{}{}
	binding := &Binding{}
	liveBindings[binding] = struct{}{}
	h := &handle{}
	sharedMu.Lock()
	previous, previousErr, previousTried := shared, sharedErr, sharedTried
	shared, sharedErr, sharedTried = h, nil, true
	sharedMu.Unlock()
	t.Cleanup(func() {
		sharedMu.Lock()
		shared, sharedErr, sharedTried = previous, previousErr, previousTried
		sharedMu.Unlock()
	})

	err := h.failed(fmt.Errorf("wgpu: submit failed: vulkan: vkQueueSubmit failed: -4"))
	if !errors.Is(err, ErrDeviceLost) {
		t.Fatalf("failed() = %v, want ErrDeviceLost", err)
	}
	if !before.Lost() || !before.Released() {
		t.Fatalf("a buffer from before the loss: lost=%v released=%v", before.Lost(), before.Released())
	}
	if !binding.lost || !binding.released {
		t.Fatalf("a binding from before the loss: lost=%v released=%v", binding.lost, binding.released)
	}
	if len(liveBuffers) != 0 || len(liveBindings) != 0 {
		t.Fatalf("registries keep %d buffers and %d bindings after the loss", len(liveBuffers), len(liveBindings))
	}
	sharedMu.Lock()
	cleared := shared == nil && !sharedTried
	sharedMu.Unlock()
	if !cleared {
		t.Fatal("the shared handle was kept, so the next call would reuse the lost device")
	}

	// The same loss reported again, by a call that raced the first, does nothing more.
	if err := h.failed(fmt.Errorf("vkQueueSubmit failed: -4")); !errors.Is(err, ErrDeviceLost) {
		t.Fatalf("second failed() = %v", err)
	}

	before.mu.Lock()
	reason := before.unusableLocked("buffer")
	before.mu.Unlock()
	if !errors.Is(reason, ErrDeviceLost) {
		t.Fatalf("a lost buffer reports %v, want ErrDeviceLost", reason)
	}

	submitMu.Unlock()
	_, err = before.Download(context.Background())
	submitMu.Lock()
	if !errors.Is(err, ErrDeviceLost) {
		t.Fatalf("downloading a lost buffer = %v, want ErrDeviceLost", err)
	}
}

// A download holds its buffer's lock when its submit or readback meets the
// loss, so retiring the device must not take that lock again: it did, and the
// call never returned while holding submitMu, which every GPU call needs.
func TestRetiringWhileACallHoldsItsBufferDoesNotDeadlock(t *testing.T) {
	submitMu.Lock()
	defer submitMu.Unlock()

	held := &Buffer{length: 1}
	liveBuffers[held] = struct{}{}
	h := &handle{}
	sharedMu.Lock()
	previous, previousErr, previousTried := shared, sharedErr, sharedTried
	shared, sharedErr, sharedTried = h, nil, true
	sharedMu.Unlock()
	t.Cleanup(func() {
		sharedMu.Lock()
		shared, sharedErr, sharedTried = previous, previousErr, previousTried
		teardown = nil
		sharedMu.Unlock()
	})

	held.mu.Lock()
	done := make(chan error, 1)
	go func() { done <- h.failed(errors.New("vkQueueSubmit failed: -4")) }()
	select {
	case err := <-done:
		held.mu.Unlock()
		if !errors.Is(err, ErrDeviceLost) {
			t.Fatalf("failed() = %v, want ErrDeviceLost", err)
		}
	case <-time.After(5 * time.Second):
		held.mu.Unlock()
		t.Fatal("retiring the device waited for a buffer lock the failing call holds")
	}
	if !held.Lost() {
		t.Fatal("the held buffer was not marked lost")
	}
}
