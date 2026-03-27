package accel

import "sort"

var builtinBackendAllocators = map[Backend]BackendAllocator{
	BackendCUDA:   builtinBackendAllocator{backend: BackendCUDA, name: "cuda-builtin"},
	BackendMetal:  builtinBackendAllocator{backend: BackendMetal, name: "metal-builtin"},
	BackendWebGPU: builtinBackendAllocator{backend: BackendWebGPU, name: "webgpu-builtin"},
}

type builtinBackendAllocator struct {
	backend Backend
	name    string
}

func (a builtinBackendAllocator) Name() string {
	return a.name
}

func (a builtinBackendAllocator) Materialize(dataset *Dataset, plan ShardPlan) AllocationRecord {
	record := AllocationRecord{
		DeviceIDs:         append([]string(nil), plan.DeviceIDs...),
		DeviceResidentMap: map[string]map[string]uint64{},
	}
	if dataset == nil || len(plan.Assignments) == 0 {
		sort.Strings(record.DeviceIDs)
		return record
	}

	for _, buffer := range dataset.Buffers {
		bufferBytes := estimateBufferResidentBytes(buffer)
		deviceBytes := distributeResidentBytes(bufferBytes, plan.Assignments)
		record.DeviceResidentMap[buffer.Name] = deviceBytes
		record.BytesMoved += builtinAllocatorBytesMoved(a.backend, bufferBytes, deviceBytes)
	}

	sort.Strings(record.DeviceIDs)
	return record
}

func ensureBuiltinBackendAllocators() {
	backendAllocatorsMu.Lock()
	defer backendAllocatorsMu.Unlock()
	for backend, allocator := range builtinBackendAllocators {
		if _, ok := backendAllocators[backend]; ok {
			continue
		}
		backendAllocators[backend] = allocator
	}
}

func distributeResidentBytes(total uint64, assignments []ShardAssignment) map[string]uint64 {
	deviceBytes := make(map[string]uint64, len(assignments))
	if total == 0 || len(assignments) == 0 {
		return deviceBytes
	}

	remaining := total
	for idx, assignment := range assignments {
		var portion uint64
		if idx == len(assignments)-1 {
			portion = remaining
		} else {
			portion = uint64(float64(total) * assignment.SharePercent)
			if portion > remaining {
				portion = remaining
			}
			remaining -= portion
		}
		deviceBytes[assignment.DeviceID] = portion
	}
	return deviceBytes
}

func builtinAllocatorBytesMoved(backend Backend, bufferBytes uint64, deviceBytes map[string]uint64) uint64 {
	switch backend {
	case BackendMetal:
		// Unified memory backends still incur some synchronization overhead, but not a full
		// host-to-device copy for every resident byte.
		return bufferBytes / 8
	case BackendWebGPU:
		// Portable shared-memory paths typically stage uploads through host-visible buffers.
		if len(deviceBytes) <= 1 {
			return bufferBytes / 2
		}
		return sumDeviceResidentBytes(deviceBytes) / 2
	case BackendCUDA:
		fallthrough
	default:
		return sumDeviceResidentBytes(deviceBytes)
	}
}

func sumDeviceResidentBytes(deviceBytes map[string]uint64) uint64 {
	total := uint64(0)
	for _, bytes := range deviceBytes {
		total += bytes
	}
	return total
}
