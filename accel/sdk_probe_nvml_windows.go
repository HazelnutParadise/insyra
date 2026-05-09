//go:build windows

package accel

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// NVML buffer-size constants (NVML 11+ headers); over-allocating is safe.
const (
	nvmlSystemDriverVersionBufferSize = 80
	nvmlDeviceNameBufferSize          = 96
	nvmlReturnSuccess                 = 0
)

// nvmlDLLNames lists candidate DLL names searched in standard system paths.
// nvml.dll is the canonical entry point installed by every modern NVIDIA
// driver; the legacy `libnvidia-ml.dll` form is here as a defensive fallback.
var nvmlDLLNames = []string{"nvml.dll", "libnvidia-ml.dll"}

// nvmlDisableEnv mirrors INSYRA_ACCEL_DISABLE_NATIVE_PROBES — set it when a
// host has NVML installed but tests need to exercise the native/stub paths.
const nvmlDisableEnv = "INSYRA_ACCEL_DISABLE_NVML_SDK"

func init() {
	probe := newNVMLSDKProbe(openWindowsNVML)
	RegisterSDKProbe(probe)
}

func isEnvFlagOn(key string) bool {
	value := strings.TrimSpace(strings.ToLower(syscallGetenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// syscallGetenv is split out so tests on hosts without NVML can still reason
// about init() registration without pulling in os.Getenv at the top of the
// file (kept consistent with native_probe.go's lookup style).
func syscallGetenv(key string) string {
	v, _ := syscall.Getenv(key)
	return v
}

func openWindowsNVML() (nvmlLoader, error) {
	if isEnvFlagOn(nvmlDisableEnv) {
		return nil, ErrSDKProbeUnavailable
	}
	for _, name := range nvmlDLLNames {
		handle, err := syscall.LoadLibrary(name)
		if err != nil {
			continue
		}
		loader, err := newWindowsNVMLLoader(handle)
		if err != nil {
			_ = syscall.FreeLibrary(handle)
			continue
		}
		return loader, nil
	}
	return nil, ErrSDKProbeUnavailable
}

type windowsNVMLLoader struct {
	mu                       sync.Mutex
	handle                   syscall.Handle
	freed                    bool
	procInit                 uintptr
	procShutdown             uintptr
	procDriverVersion        uintptr
	procDeviceCount          uintptr
	procDeviceHandleByIndex  uintptr
	procDeviceName           uintptr
	procDeviceMemoryInfo     uintptr
	procDeviceComputeCapInts uintptr
}

func newWindowsNVMLLoader(handle syscall.Handle) (*windowsNVMLLoader, error) {
	loader := &windowsNVMLLoader{handle: handle}
	procs := []struct {
		name string
		dst  *uintptr
	}{
		{"nvmlInit_v2", &loader.procInit},
		{"nvmlShutdown", &loader.procShutdown},
		{"nvmlSystemGetDriverVersion", &loader.procDriverVersion},
		{"nvmlDeviceGetCount_v2", &loader.procDeviceCount},
		{"nvmlDeviceGetHandleByIndex_v2", &loader.procDeviceHandleByIndex},
		{"nvmlDeviceGetName", &loader.procDeviceName},
		{"nvmlDeviceGetMemoryInfo", &loader.procDeviceMemoryInfo},
		{"nvmlDeviceGetCudaComputeCapability", &loader.procDeviceComputeCapInts},
	}
	for _, p := range procs {
		addr, err := syscall.GetProcAddress(handle, p.name)
		if err != nil {
			return nil, fmt.Errorf("accel: nvml missing symbol %s: %w", p.name, err)
		}
		*p.dst = addr
	}
	return loader, nil
}

func (l *windowsNVMLLoader) Init() error {
	if rc, _, _ := syscall.SyscallN(l.procInit); rc != nvmlReturnSuccess {
		return fmt.Errorf("accel: nvmlInit_v2 returned %d", rc)
	}
	return nil
}

func (l *windowsNVMLLoader) Shutdown() error {
	rc, _, _ := syscall.SyscallN(l.procShutdown)
	l.free()
	if rc != nvmlReturnSuccess {
		return fmt.Errorf("accel: nvmlShutdown returned %d", rc)
	}
	return nil
}

func (l *windowsNVMLLoader) free() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.freed {
		return
	}
	_ = syscall.FreeLibrary(l.handle)
	l.freed = true
}

func (l *windowsNVMLLoader) DriverVersion() (string, error) {
	buf := make([]byte, nvmlSystemDriverVersionBufferSize)
	rc, _, _ := syscall.SyscallN(
		l.procDriverVersion,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if rc != nvmlReturnSuccess {
		return "", fmt.Errorf("accel: nvmlSystemGetDriverVersion returned %d", rc)
	}
	return cstringToGo(buf), nil
}

func (l *windowsNVMLLoader) DeviceCount() (int, error) {
	var count uint32
	rc, _, _ := syscall.SyscallN(
		l.procDeviceCount,
		uintptr(unsafe.Pointer(&count)),
	)
	if rc != nvmlReturnSuccess {
		return 0, fmt.Errorf("accel: nvmlDeviceGetCount_v2 returned %d", rc)
	}
	return int(count), nil
}

func (l *windowsNVMLLoader) Device(index int) (nvmlDeviceInfo, error) {
	var handle uintptr
	rc, _, _ := syscall.SyscallN(
		l.procDeviceHandleByIndex,
		uintptr(uint32(index)),
		uintptr(unsafe.Pointer(&handle)),
	)
	if rc != nvmlReturnSuccess {
		return nvmlDeviceInfo{}, fmt.Errorf("accel: nvmlDeviceGetHandleByIndex_v2(%d) returned %d", index, rc)
	}

	info := nvmlDeviceInfo{}

	nameBuf := make([]byte, nvmlDeviceNameBufferSize)
	rc, _, _ = syscall.SyscallN(
		l.procDeviceName,
		handle,
		uintptr(unsafe.Pointer(&nameBuf[0])),
		uintptr(len(nameBuf)),
	)
	if rc == nvmlReturnSuccess {
		info.Name = cstringToGo(nameBuf)
	}

	// nvmlMemory_t is { unsigned long long total; free; used; } = 24 bytes.
	var mem [3]uint64
	rc, _, _ = syscall.SyscallN(
		l.procDeviceMemoryInfo,
		handle,
		uintptr(unsafe.Pointer(&mem[0])),
	)
	if rc == nvmlReturnSuccess {
		info.TotalMemoryBytes = mem[0]
	}

	var major, minor int32
	rc, _, _ = syscall.SyscallN(
		l.procDeviceComputeCapInts,
		handle,
		uintptr(unsafe.Pointer(&major)),
		uintptr(unsafe.Pointer(&minor)),
	)
	if rc == nvmlReturnSuccess {
		info.ComputeMajor = int(major)
		info.ComputeMinor = int(minor)
	}

	if info.Name == "" && info.TotalMemoryBytes == 0 {
		return info, errors.New("accel: nvml returned empty device info")
	}
	return info, nil
}

func cstringToGo(buf []byte) string {
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}
