package accel

import (
	"os"
	"runtime"
	"strings"
)

type builtinDiscoverer struct {
	name        string
	backend     Backend
	vendor      string
	deviceType  DeviceType
	memoryClass MemoryClass
	budgetBytes uint64
	caps        map[string]bool
	envKey      string
}

func builtinDiscoverers() []Discoverer {
	return []Discoverer{
		builtinDiscoverer{
			name:        "cuda",
			backend:     BackendCUDA,
			vendor:      "nvidia",
			deviceType:  DeviceTypeDiscrete,
			memoryClass: MemoryClassDevice,
			budgetBytes: 8 * 1024 * 1024 * 1024,
			envKey:      "INSYRA_ACCEL_STUB_CUDA",
			caps: map[string]bool{
				"multi_gpu":        true,
				"device_local_mem": true,
			},
		},
		builtinDiscoverer{
			name:        "metal",
			backend:     BackendMetal,
			vendor:      "apple",
			deviceType:  DeviceTypeIntegrated,
			memoryClass: MemoryClassShared,
			budgetBytes: 4 * 1024 * 1024 * 1024,
			envKey:      "INSYRA_ACCEL_STUB_METAL",
			caps: map[string]bool{
				"shared_memory": true,
				"unified_mem":   true,
			},
		},
		builtinDiscoverer{
			name:        "webgpu",
			backend:     BackendWebGPU,
			vendor:      "portable",
			deviceType:  DeviceTypeIntegrated,
			memoryClass: MemoryClassShared,
			budgetBytes: 2 * 1024 * 1024 * 1024,
			envKey:      "INSYRA_ACCEL_STUB_WEBGPU",
			caps: map[string]bool{
				"portable":      true,
				"shared_memory": true,
			},
		},
	}
}

func (d builtinDiscoverer) Name() string {
	return d.name
}

func (d builtinDiscoverer) Discover(cfg Config) ([]Device, error) {
	if !envEnabled(d.envKey) {
		return nil, nil
	}

	return []Device{
		{
			ID:                string(d.backend) + ":stub:0",
			Name:              stubDeviceName(d.backend),
			Vendor:            d.vendor,
			Backend:           d.backend,
			Type:              d.deviceType,
			MemoryClass:       d.memoryClass,
			SharedMemory:      d.memoryClass == MemoryClassShared,
			BudgetBytes:       d.budgetBytes,
			CapabilitySummary: cloneCaps(d.caps),
			Score:             stubDeviceScore(d.backend),
		},
	}, nil
}

func envEnabled(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func stubDeviceName(backend Backend) string {
	switch backend {
	case BackendCUDA:
		return "Stub CUDA Device"
	case BackendMetal:
		if runtime.GOOS == "darwin" {
			return "Stub Apple GPU"
		}
		return "Stub Metal Device"
	case BackendWebGPU:
		return "Stub WebGPU Device"
	default:
		return "Stub Accelerator Device"
	}
}

func stubDeviceScore(backend Backend) float64 {
	switch backend {
	case BackendCUDA:
		return 95
	case BackendMetal:
		return 85
	case BackendWebGPU:
		return 75
	default:
		return 50
	}
}

func cloneCaps(caps map[string]bool) map[string]bool {
	if caps == nil {
		return nil
	}
	cloned := make(map[string]bool, len(caps))
	for key, value := range caps {
		cloned[key] = value
	}
	return cloned
}
