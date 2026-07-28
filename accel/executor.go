package accel

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/HazelnutParadise/insyra"
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

func (s *Session) ExecuteProjectedDataset(dataset *Dataset, workload WorkloadEstimate) (ExecutionResult, error) {
	if s == nil {
		return ExecutionResult{}, fmt.Errorf("accel: nil session")
	}
	if dataset == nil {
		return ExecutionResult{}, fmt.Errorf("accel: nil dataset")
	}
	if workload.Class == "" {
		workload.Class = WorkloadClassColumnar
	}
	if workload.Op == "" {
		workload.Op = OpSum
	}
	if workload.Precision == "" {
		workload.Precision = PrecisionExact
	}
	if workload.Rows <= 0 {
		workload.Rows = dataset.Rows
	}
	if workload.Bytes == 0 {
		workload.Bytes = estimateDatasetResidentBytes(dataset)
	}

	// Held across backend execution: releasing it around the device call would
	// let another goroutine observe half-updated cache and report state, which
	// is the race this lock exists to remove.
	s.mu.Lock()
	defer s.mu.Unlock()

	plan := s.planShardableWorkloadLocked(workload)
	result := ExecutionResult{
		Accelerated:    plan.Accelerated,
		FallbackReason: plan.FallbackReason,
		MergePolicy:    plan.MergePolicy,
		ExecutorKind:   ExecutorKindUnknown,
		Assignments:    append([]ShardAssignment(nil), plan.Assignments...),
		DeviceIDs:      append([]string(nil), plan.DeviceIDs...),
		Op:             workload.Op,
	}
	if !plan.Accelerated {
		return s.finishExecution(result, nil)
	}

	if workload.Op != OpSum {
		return s.abortExecution(result, FallbackReasonWorkloadUnsupported,
			fmt.Errorf("accel: unsupported operation %q", workload.Op))
	}

	executor, ok := lookupBackendExecutor(plan.Backend)
	if !ok {
		return s.abortExecution(result, FallbackReasonNoBackendExecutor,
			fmt.Errorf("accel: no execution backend registered for %q", plan.Backend))
	}
	result.Executor = executor.Name()
	result.ExecutorKind = ExecutorKindRegistered

	// This change ships single-device execution. Narrow the plan to the device
	// that actually runs the work rather than reporting devices that did not.
	device, ok := s.executionDevice(plan)
	if !ok {
		return s.abortExecution(result, FallbackReasonNoAccelerator,
			errors.New("accel: plan named no usable device"))
	}
	result.DeviceIDs = []string{device.ID}
	result.Assignments = assignmentsForDevice(plan, device.ID)

	columns := make([]ExecuteColumn, 0, len(dataset.Buffers))
	counts := make(map[string]int, len(dataset.Buffers))
	empties := make([]string, 0, len(dataset.Buffers))
	for _, buffer := range dataset.Buffers {
		values, count, reason := deviceValues(buffer, workload.Precision)
		if reason != FallbackReasonNone {
			return s.abortExecution(result, reason,
				fmt.Errorf("accel: column %q is not eligible for device execution (%s)", buffer.Name, reason))
		}
		counts[buffer.Name] = count
		// An empty column needs no device work; dispatching zero workgroups is
		// a validation error on some backends.
		if len(values) == 0 {
			empties = append(empties, buffer.Name)
			continue
		}
		columns = append(columns, ExecuteColumn{Name: buffer.Name, Values: values})
	}

	result.Precision = workload.Precision
	result.Counts = counts
	result.Reductions = make(map[string]float64, len(dataset.Buffers))
	for _, name := range empties {
		result.Reductions[name] = 0
	}

	// One submission for the whole dataset: readback dominates device cost, so
	// a request per column would pay that wait once per column.
	if len(columns) > 0 {
		response, err := executor.Execute(context.Background(), ExecuteRequest{
			Op:        workload.Op,
			Device:    device,
			Columns:   columns,
			Precision: workload.Precision,
		})
		if err != nil {
			return s.abortExecution(result, fallbackReasonForExecError(err), err)
		}
		for name, value := range response.Reductions {
			result.Reductions[name] = value
		}
		result.Transfer = response.Transfer
		result.Dispatch = response.Dispatch
		result.Readback = response.Readback
		result.BytesUploaded = response.BytesUploaded
	}

	s.ensureDatasetCached(dataset)
	s.applyDeviceResidency(dataset, device.ID)
	return s.finishExecution(result, nil)
}

// abortExecution turns a mid-flight failure back into a non-accelerated result
// so the caller still gets an inspectable report, and returns the error only
// when the session demands GPU execution.
func (s *Session) abortExecution(result ExecutionResult, reason FallbackReason, err error) (ExecutionResult, error) {
	result.Accelerated = false
	result.FallbackReason = reason
	result.Reductions = nil
	result.Counts = nil
	result.Transfer = 0
	result.Dispatch = 0
	result.Readback = 0
	result.BytesUploaded = 0
	return s.finishExecution(result, err)
}

func (s *Session) finishExecution(result ExecutionResult, err error) (ExecutionResult, error) {
	s.recordExecutionMetrics(result)
	if !result.Accelerated && strictGPURequired(s.cfg) {
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

// executionDevice picks the device carrying the largest share of the plan.
func (s *Session) executionDevice(plan ShardPlan) (Device, bool) {
	best := ""
	bestWeight := -1.0
	for _, assignment := range plan.Assignments {
		if assignment.Weight > bestWeight {
			bestWeight = assignment.Weight
			best = assignment.DeviceID
		}
	}
	if best == "" && len(plan.DeviceIDs) > 0 {
		best = plan.DeviceIDs[0]
	}
	for _, device := range s.devices {
		if device.ID == best {
			return cloneDevice(device), true
		}
	}
	return Device{}, false
}

func assignmentsForDevice(plan ShardPlan, deviceID string) []ShardAssignment {
	for _, assignment := range plan.Assignments {
		if assignment.DeviceID == deviceID {
			return []ShardAssignment{assignment}
		}
	}
	return nil
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

func (s *Session) ExecuteDataList(dl *insyra.DataList, workload WorkloadEstimate) (ExecutionResult, error) {
	if s == nil {
		return ExecutionResult{}, fmt.Errorf("accel: nil session")
	}
	if dl == nil {
		return ExecutionResult{}, fmt.Errorf("accel: nil datalist")
	}
	buffer, err := projectValues(dl.GetName(), dl.Data())
	if err != nil {
		return ExecutionResult{}, err
	}
	dataset := &Dataset{
		Name:    dl.GetName(),
		Lineage: "project:datalist",
		Rows:    buffer.Len,
		Buffers: []Buffer{buffer},
	}
	assignDatasetFingerprint(dataset)
	return s.ExecuteProjectedDataset(dataset, workload)
}

func (s *Session) ExecuteDataTable(dt *insyra.DataTable, workload WorkloadEstimate) (ExecutionResult, error) {
	if s == nil {
		return ExecutionResult{}, fmt.Errorf("accel: nil session")
	}
	if dt == nil {
		return ExecutionResult{}, fmt.Errorf("accel: nil datatable")
	}
	cols := make([]Buffer, 0, dt.NumCols())
	for i := 0; i < dt.NumCols(); i++ {
		col := dt.GetColByNumber(i)
		buf, err := projectValues(col.GetName(), col.Data())
		if err != nil {
			return ExecutionResult{}, err
		}
		cols = append(cols, buf)
	}
	dataset := &Dataset{
		Name:    dt.GetName(),
		Lineage: "project:datatable",
		Rows:    dt.NumRows(),
		Buffers: cols,
	}
	assignDatasetFingerprint(dataset)
	return s.ExecuteProjectedDataset(dataset, workload)
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
		entry.DeviceIDs = []string{deviceID}
		entry.DeviceResidentBytes = map[string]uint64{deviceID: entry.ResidentBytes}
		s.cache.entries[key] = entry
	}
	s.updateCacheMetrics()
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
	report.Metrics["execution.fallback"] = boolMetric(!result.Accelerated && result.FallbackReason != FallbackReasonNone)
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
