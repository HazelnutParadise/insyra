package accel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Sentinel errors a backend wraps so the runtime can map a failure onto a
// stable fallback reason without knowing anything about the backend.
var (
	ErrShaderCompile   = errors.New("accel: shader compilation failed")
	ErrBufferTooLarge  = errors.New("accel: column exceeds device buffer limit")
	ErrReadbackTimeout = errors.New("accel: device readback timed out")
)

// ExecuteColumn is one projected column, already narrowed to float32 and with
// nulls replaced by the operation's identity, so a backend never has to know
// about precision policy or Insyra's null representation.
type ExecuteColumn struct {
	Name   string
	Values []float32
}

// ExecuteRequest is one operation over one dataset on one device. Columns keep
// their dataset order, because a kernel that reads across columns cares about
// position in a way a map would lose.
type ExecuteRequest struct {
	Op        Op
	Device    Device
	Columns   []ExecuteColumn
	Precision Precision
	// Queries holds one point per entry, each carrying one value per column in
	// Columns order. Only OpSquaredDistance reads it.
	Queries [][]float32
	// Shortlist is how many candidates per row OpNearestShortlist keeps. Other
	// operations ignore it.
	Shortlist int
}

// ExecuteResponse carries the computed results and what the submission cost.
// The durations describe the whole submission rather than any one column —
// transfer, dispatch and readback are properties of the submission, and a
// per-column split would be invented. They are host-observed: Metal and GLES do
// not implement GPU timestamp queries.
type ExecuteResponse struct {
	Reductions map[string]float64
	// Distances is query-major for OpSquaredDistance: entry q*rows+r is the
	// distance from row r to query q. For OpNearestQuery it holds one distance
	// per row, the smallest one.
	Distances []float32
	// NearestIndex holds the closest query point per row. Only OpNearestQuery
	// fills it.
	NearestIndex []uint32
	// ShortlistIndex and ShortlistDistance are row-major, holding Shortlist
	// entries per row: entry r*Shortlist+j is row r's j-th nearest.
	// ShortlistBoundary holds one value per row, the distance of the best
	// candidate that did not make the list. Only OpNearestShortlist fills them.
	ShortlistIndex    []uint32
	ShortlistDistance []float32
	ShortlistBoundary []float32

	Transfer      time.Duration
	Dispatch      time.Duration
	Readback      time.Duration
	BytesUploaded uint64
}

// BackendExecutor runs an operation on a real device. Registering one is how a
// backend module opts into the accel runtime.
type BackendExecutor interface {
	Name() string
	Execute(ctx context.Context, req ExecuteRequest) (ExecuteResponse, error)
}

var (
	backendExecutorsMu sync.RWMutex
	backendExecutors   = map[Backend]BackendExecutor{}
)

func RegisterBackendExecutor(backend Backend, executor BackendExecutor) error {
	if backend == "" || backend == BackendUnknown || backend == BackendCPU {
		return fmt.Errorf("accel: executor backend is required")
	}
	if executor == nil {
		return fmt.Errorf("accel: executor is required")
	}
	backendExecutorsMu.Lock()
	defer backendExecutorsMu.Unlock()
	backendExecutors[backend] = executor
	return nil
}

// ResetBackendExecutorsForTest clears every registered executor. Tests use this
// to keep one package's registration from leaking into another's.
func ResetBackendExecutorsForTest() {
	backendExecutorsMu.Lock()
	defer backendExecutorsMu.Unlock()
	backendExecutors = map[Backend]BackendExecutor{}
}

func lookupBackendExecutor(backend Backend) (BackendExecutor, bool) {
	backendExecutorsMu.RLock()
	defer backendExecutorsMu.RUnlock()
	executor, ok := backendExecutors[backend]
	return executor, ok
}

func (s *Session) finishExecution(result ExecutionResult, rows int, err error) (ExecutionResult, error) {
	s.recordExecutionMetrics(result)
	s.logExecutionLocked(string(result.Op), result, rows)
	if (!result.Accelerated || result.FallbackReason != FallbackReasonNone) && strictGPURequired(s.cfg) {
		if err != nil {
			return result, fmt.Errorf("accel: unable to execute on the acceleration path (%s): %w", result.FallbackReason, err)
		}
		return result, fmt.Errorf("accel: unable to execute on the acceleration path (%s)", result.FallbackReason)
	}
	return result, nil
}

func fallbackReasonForExecError(err error) FallbackReason {
	switch {
	case errors.Is(err, ErrShaderCompile):
		return FallbackReasonShaderCompileFailed
	case errors.Is(err, ErrBufferTooLarge):
		return FallbackReasonBufferTooLarge
	case errors.Is(err, ErrReadbackTimeout), errors.Is(err, context.DeadlineExceeded):
		return FallbackReasonReadbackTimeout
	default:
		return FallbackReasonExecutionFailed
	}
}

// deviceValues narrows a projected column into what a device can hold, or says
// why it cannot. Nulls become the additive identity so backends need no null
// concept; the returned count is the number of values that were not null.
func deviceValues(buffer Buffer, precision Precision) ([]float32, int, FallbackReason) {
	isNull := func(i int) bool { return i < len(buffer.Nulls) && buffer.Nulls[i] }

	switch values := buffer.Values.(type) {
	case []float64:
		if precision != PrecisionFloat32 {
			return nil, 0, FallbackReasonPrecisionNotAccepted
		}
		out := make([]float32, len(values))
		count := 0
		for i, value := range values {
			if isNull(i) {
				continue
			}
			out[i] = float32(value)
			count++
		}
		return out, count, FallbackReasonNone
	case []int64:
		// WGSL has no i64 either, so an integer column is only reachable under
		// the same explicit opt-in. Values above 2^24 lose precision, which is
		// exactly what the opt-in acknowledges.
		if precision != PrecisionFloat32 {
			return nil, 0, FallbackReasonPrecisionNotAccepted
		}
		out := make([]float32, len(values))
		count := 0
		for i, value := range values {
			if isNull(i) {
				continue
			}
			out[i] = float32(value)
			count++
		}
		return out, count, FallbackReasonNone
	default:
		return nil, 0, FallbackReasonDTypeNotEligible
	}
}

func estimateDatasetResidentBytes(dataset *Dataset) uint64 {
	if dataset == nil {
		return 0
	}
	total := uint64(0)
	for _, buffer := range dataset.Buffers {
		total += estimateBufferResidentBytes(buffer)
	}
	return total
}

// applyDeviceResidency records that the dataset's buffers really are resident
// on the device that executed the work. Unlike the estimate it replaces, every
// byte here was actually uploaded.
// applyDeviceResidency assumes s.mu is held.
func (s *Session) applyDeviceResidency(dataset *Dataset, deviceID string) {
	if s == nil || s.cache == nil || dataset == nil || deviceID == "" {
		return
	}
	for idx, buffer := range dataset.Buffers {
		key := cacheKey(dataset, buffer, idx)
		entry, ok := s.cache.entries[key]
		if !ok {
			continue
		}
		if !containsString(entry.DeviceIDs, deviceID) {
			entry.DeviceIDs = append(entry.DeviceIDs, deviceID)
		}
		if entry.DeviceResidentBytes == nil {
			entry.DeviceResidentBytes = map[string]uint64{}
		}
		entry.DeviceResidentBytes[deviceID] = entry.ResidentBytes
		s.cache.entries[key] = entry
	}
	s.updateCacheMetrics()
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// recordExecutionMetrics assumes s.mu is held.
func (s *Session) recordExecutionMetrics(result ExecutionResult) {
	if s == nil || len(s.reports) == 0 {
		return
	}
	report := s.reportLocked()
	if report.Metrics == nil {
		report.Metrics = map[string]float64{}
	}
	report.Metrics["execution.accelerated"] = boolMetric(result.Accelerated)
	report.Metrics["execution.fallback"] = boolMetric(result.FallbackReason != FallbackReasonNone)
	report.Metrics["execution.device_participants"] = float64(len(result.DeviceIDs))
	report.Metrics["execution.assignments"] = float64(len(result.Assignments))
	report.Metrics["execution.merge_cpu"] = boolMetric(result.MergePolicy == MergePolicyCPU)
	report.Metrics["execution.merge_backend_native"] = boolMetric(result.MergePolicy == MergePolicyBackendNative)
	report.Metrics["execution.executor_registered"] = boolMetric(result.ExecutorKind == ExecutorKindRegistered)
	// Cost metrics describe real device activity, so they are only present when
	// something actually ran on a device.
	if result.Accelerated {
		report.Metrics["execution.bytes_uploaded"] = float64(result.BytesUploaded)
		report.Metrics["execution.transfer_ms"] = float64(result.Transfer.Nanoseconds()) / 1e6
		report.Metrics["execution.dispatch_ms"] = float64(result.Dispatch.Nanoseconds()) / 1e6
		report.Metrics["execution.readback_ms"] = float64(result.Readback.Nanoseconds()) / 1e6
	} else {
		delete(report.Metrics, "execution.bytes_uploaded")
		delete(report.Metrics, "execution.transfer_ms")
		delete(report.Metrics, "execution.dispatch_ms")
		delete(report.Metrics, "execution.readback_ms")
	}
	s.reports[len(s.reports)-1] = cloneReport(report)
}

// ensureDatasetCached assumes s.mu is held.
func (s *Session) ensureDatasetCached(dataset *Dataset) {
	if s == nil || s.cache == nil || dataset == nil {
		return
	}
	if dataset.Fingerprint == "" {
		assignDatasetFingerprint(dataset)
	}
	if len(dataset.Buffers) == 0 {
		return
	}
	firstKey := cacheKey(dataset, dataset.Buffers[0], 0)
	if _, ok := s.cache.entries[firstKey]; ok {
		return
	}
	s.cacheDataset(dataset)
}

func boolMetric(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
