package accel

import (
	"fmt"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

// logExecutionLocked records the execution lifecycle after a result has been
// decided. Callers hold s.mu, so the once-only flags are safe when a session is
// used concurrently.
func (s *Session) logExecutionLocked(operation string, result ExecutionResult, rows int) {
	if s == nil {
		return
	}
	if operation == "" {
		operation = string(OpUnknown)
	}
	if rows <= 0 {
		for _, assignment := range result.Assignments {
			rows += assignment.Rows
		}
	}
	if rows < 0 {
		rows = 0
	}

	insyra.LogDebug("accel", "execute", "execution operation=%s rows=%d chunks=%d placement=%s fallback_reason=%s",
		operation, rows, result.Chunks, formatPlacements(s, result.Assignments), result.FallbackReason)

	if result.Accelerated && !s.accelerationInfoLogged {
		devices, backends := s.executionDevices(result)
		insyra.LogInfo("accel", "execute",
			"session acceleration engaged device=%s backend=%s mode=%s shard_strategy=%s",
			devices, backends, s.cfg.Mode, effectiveShardStrategy(s.cfg.ShardStrategy))
		s.accelerationInfoLogged = true
	}
	if result.FallbackReason != FallbackReasonNone && qualifyingFallback(result.FallbackReason) && !s.fallbackInfoLogged {
		insyra.LogInfo("accel", "execute", "session acceleration fallback reason=%s", result.FallbackReason)
		s.fallbackInfoLogged = true
	}
}

func qualifyingFallback(reason FallbackReason) bool {
	switch reason {
	case FallbackReasonNoAccelerator,
		FallbackReasonCPUOnly,
		FallbackReasonDiscoveryError,
		FallbackReasonStrictGPUUnavailable,
		FallbackReasonDeviceSelectionEmpty,
		FallbackReasonWorkloadUnsupported,
		FallbackReasonWorkloadNotProfitable,
		FallbackReasonPrecisionNotAccepted,
		FallbackReasonDTypeNotEligible:
		return false
	default:
		return true
	}
}

func (s *Session) executionDevices(result ExecutionResult) (string, string) {
	byID := make(map[string]Device, len(s.devices))
	for _, device := range s.devices {
		byID[device.ID] = device
	}
	ids := append([]string(nil), result.DeviceIDs...)
	backends := make([]string, 0, len(ids))
	devices := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	for _, assignment := range result.Assignments {
		if assignment.DeviceID == "" {
			continue
		}
		if _, ok := seen[assignment.DeviceID]; ok {
			continue
		}
		seen[assignment.DeviceID] = struct{}{}
		ids = append(ids, assignment.DeviceID)
	}
	for _, id := range ids {
		device := byID[id]
		name := device.Name
		if name == "" {
			name = id
		}
		if name != "" {
			devices = append(devices, name)
		}
		backend := device.Backend
		if backend == "" {
			for _, assignment := range result.Assignments {
				if assignment.DeviceID == id {
					backend = assignment.Backend
					break
				}
			}
		}
		if backend != "" {
			backends = append(backends, string(backend))
		}
	}
	if len(devices) == 0 {
		devices = append(devices, "unknown")
	}
	if len(backends) == 0 {
		backends = append(backends, string(BackendUnknown))
	}
	return strings.Join(devices, ","), strings.Join(backends, ",")
}

func formatPlacements(s *Session, assignments []ShardAssignment) string {
	if len(assignments) == 0 {
		return "none"
	}
	byID := make(map[string]Device, len(s.devices))
	for _, device := range s.devices {
		byID[device.ID] = device
	}
	placements := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		name := assignment.DeviceID
		if device := byID[assignment.DeviceID]; device.Name != "" {
			name = device.Name
		}
		chunks := assignment.Chunks
		if chunks == 0 && assignment.Rows > 0 {
			chunks = len(exactNearestChunkRanges(assignment.Rows, exactNearestChunkRows))
		}
		placements = append(placements, fmt.Sprintf("device=%s backend=%s rows=%d chunks=%d fallback=%s",
			name, assignment.Backend, assignment.Rows, chunks, assignment.FallbackReason))
	}
	return strings.Join(placements, ";")
}
