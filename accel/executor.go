package accel

import (
	"fmt"
	"sort"
)

type executionAllocator interface {
	Materialize(dataset *Dataset, plan ShardPlan) AllocationRecord
}

type AllocationRecord struct {
	DeviceIDs         []string
	DeviceResidentMap map[string]map[string]uint64
	BytesMoved        uint64
}

type ledgerAllocator struct{}

func (ledgerAllocator) Materialize(dataset *Dataset, plan ShardPlan) AllocationRecord {
	record := AllocationRecord{
		DeviceIDs:         append([]string(nil), plan.DeviceIDs...),
		DeviceResidentMap: map[string]map[string]uint64{},
	}
	if dataset == nil || len(plan.Assignments) == 0 {
		return record
	}

	for _, buffer := range dataset.Buffers {
		bufferBytes := estimateBufferResidentBytes(buffer)
		deviceBytes := make(map[string]uint64, len(plan.Assignments))
		remaining := bufferBytes
		for idx, assignment := range plan.Assignments {
			var portion uint64
			if idx == len(plan.Assignments)-1 {
				portion = remaining
			} else {
				portion = uint64(float64(bufferBytes) * assignment.SharePercent)
				if portion > remaining {
					portion = remaining
				}
				remaining -= portion
			}
			deviceBytes[assignment.DeviceID] = portion
			record.BytesMoved += portion
		}
		record.DeviceResidentMap[buffer.Name] = deviceBytes
	}

	sort.Strings(record.DeviceIDs)
	return record
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
	if workload.Rows <= 0 {
		workload.Rows = dataset.Rows
	}
	if workload.Bytes == 0 {
		workload.Bytes = estimateDatasetResidentBytes(dataset)
	}

	plan := s.PlanShardableWorkload(workload)
	result := ExecutionResult{
		Accelerated:    plan.Accelerated,
		FallbackReason: plan.FallbackReason,
		MergePolicy:    plan.MergePolicy,
		Assignments:    append([]ShardAssignment(nil), plan.Assignments...),
		DeviceIDs:      append([]string(nil), plan.DeviceIDs...),
	}
	if !plan.Accelerated {
		s.recordExecutionMetrics(result)
		if strictGPURequired(s.cfg) {
			return result, fmt.Errorf("accel: unable to execute projected dataset on acceleration path (%s)", plan.FallbackReason)
		}
		return result, nil
	}

	allocator := ledgerAllocator{}
	s.ensureDatasetCached(dataset)
	record := allocator.Materialize(dataset, plan)
	result.BytesMoved = record.BytesMoved
	s.applyAllocationRecord(dataset, record)
	s.recordExecutionMetrics(result)
	return result, nil
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

func (s *Session) applyAllocationRecord(dataset *Dataset, record AllocationRecord) {
	if s == nil || s.cache == nil || dataset == nil {
		return
	}
	for idx, buffer := range dataset.Buffers {
		key := cacheKey(dataset, buffer, idx)
		entry, ok := s.cache.entries[key]
		if !ok {
			continue
		}
		entry.DeviceIDs = append([]string(nil), record.DeviceIDs...)
		deviceBytes := record.DeviceResidentMap[buffer.Name]
		entry.DeviceResidentBytes = cloneDeviceResidentBytes(deviceBytes)
		s.cache.entries[key] = entry
	}
	s.updateCacheMetrics()
}

func (s *Session) recordExecutionMetrics(result ExecutionResult) {
	if s == nil || len(s.reports) == 0 {
		return
	}
	report := s.Report()
	if report.Metrics == nil {
		report.Metrics = map[string]float64{}
	}
	report.Metrics["execution.accelerated"] = boolMetric(result.Accelerated)
	report.Metrics["execution.fallback"] = boolMetric(!result.Accelerated && result.FallbackReason != FallbackReasonNone)
	report.Metrics["execution.device_participants"] = float64(len(result.DeviceIDs))
	report.Metrics["execution.assignments"] = float64(len(result.Assignments))
	report.Metrics["execution.bytes_moved"] = float64(result.BytesMoved)
	report.Metrics["execution.merge_cpu"] = boolMetric(result.MergePolicy == MergePolicyCPU)
	report.Metrics["execution.merge_backend_native"] = boolMetric(result.MergePolicy == MergePolicyBackendNative)
	s.reports[len(s.reports)-1] = cloneReport(report)
}

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

func cloneDeviceResidentBytes(input map[string]uint64) map[string]uint64 {
	if input == nil {
		return nil
	}
	cloned := make(map[string]uint64, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
