package accel

import "sort"

const (
	saturationRows32  = 32_000
	saturationRows128 = 8_000
)

type ShardPlan struct {
	Accelerated      bool
	Backend          Backend
	DeviceIDs        []string
	Assignments      []ShardAssignment
	TotalBudgetBytes uint64
	Heterogeneous    bool
	MergePolicy      MergePolicy
	FallbackReason   FallbackReason
}

func (s *Session) PlanShardable() ShardPlan {
	return s.PlanShardableWorkload(WorkloadEstimate{Class: WorkloadClassColumnar})
}

func (s *Session) PlanShardableWorkload(workload WorkloadEstimate) ShardPlan {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.planShardableWorkloadLocked(workload)
}

// planShardableWorkloadLocked assumes s.mu is held.
func (s *Session) planShardableWorkloadLocked(workload WorkloadEstimate) ShardPlan {
	report := s.reportLocked()
	if len(s.devices) == 0 || !report.Accelerated {
		return ShardPlan{
			Accelerated:    false,
			Backend:        report.SelectedBackend,
			MergePolicy:    MergePolicyUnknown,
			FallbackReason: report.FallbackReason,
		}
	}
	if workload.Class != "" && workload.Class != WorkloadClassColumnar {
		return ShardPlan{
			Accelerated:    false,
			Backend:        report.SelectedBackend,
			MergePolicy:    MergePolicyUnknown,
			FallbackReason: FallbackReasonWorkloadUnsupported,
		}
	}

	candidates := shardableDevices(s.devices, s.cfg)
	if len(candidates) == 0 {
		return ShardPlan{
			Accelerated:    false,
			Backend:        report.SelectedBackend,
			MergePolicy:    MergePolicyUnknown,
			FallbackReason: report.FallbackReason,
		}
	}
	if shouldFallbackForProfitability(workload, s.cfg) {
		return ShardPlan{
			Accelerated:    false,
			Backend:        report.SelectedBackend,
			MergePolicy:    MergePolicyUnknown,
			FallbackReason: FallbackReasonWorkloadNotProfitable,
		}
	}

	strategy := effectiveShardStrategy(s.cfg.ShardStrategy)
	assignmentCount := shardCountForStrategy(strategy, len(candidates), workload.Rows, saturationPointForWorkload(workload))
	if assignmentCount < len(candidates) {
		candidates = candidates[:assignmentCount]
	}

	plan := ShardPlan{
		Accelerated:    true,
		Backend:        report.SelectedBackend,
		DeviceIDs:      make([]string, 0, len(candidates)),
		Assignments:    make([]ShardAssignment, 0, len(candidates)),
		FallbackReason: FallbackReasonNone,
		MergePolicy:    mergePolicyForDevices(candidates),
	}
	backendSet := map[Backend]struct{}{}
	totalWeight := 0.0
	weights := make([]float64, len(candidates))
	for idx, device := range candidates {
		plan.DeviceIDs = append(plan.DeviceIDs, device.ID)
		plan.TotalBudgetBytes += device.BudgetBytes
		backendSet[device.Backend] = struct{}{}
		weight := shardWeight(device)
		weights[idx] = weight
		totalWeight += weight
	}
	plan.Heterogeneous = len(backendSet) > 1

	remainingRows := workload.Rows
	remainingBytes := workload.Bytes
	rowOffset := 0
	var rowShares []int
	if workload.Rows > 0 && strategy == ShardStrategyAuto && len(candidates) > 1 {
		rowShares = minimumShardRows(workload.Rows, weights, saturationPointForWorkload(workload))
	}
	for idx, device := range candidates {
		sharePercent := 0.0
		if totalWeight > 0 {
			sharePercent = weights[idx] / totalWeight
		}
		assignment := ShardAssignment{
			DeviceID:     device.ID,
			Backend:      device.Backend,
			Weight:       weights[idx],
			SharePercent: sharePercent,
			BudgetBytes:  device.BudgetBytes,
		}
		if workload.Rows > 0 {
			if len(rowShares) > 0 {
				assignment.Rows = rowShares[idx]
			} else {
				assignment.Rows = proportionalIntShare(workload.Rows, idx, len(candidates), sharePercent, &remainingRows)
			}
			assignment.RowStart = rowOffset
			assignment.RowEnd = rowOffset + assignment.Rows
			rowOffset = assignment.RowEnd
		}
		if workload.Bytes > 0 {
			assignment.Bytes = proportionalUint64Share(workload.Bytes, idx, len(candidates), sharePercent, &remainingBytes)
		}
		plan.Assignments = append(plan.Assignments, assignment)
	}
	return plan
}

func effectiveShardStrategy(strategy ShardStrategy) ShardStrategy {
	switch strategy {
	case ShardStrategySingle, ShardStrategyForced:
		return strategy
	default:
		return ShardStrategyAuto
	}
}

func saturationPointForWorkload(workload WorkloadEstimate) int {
	if workload.Dimensions >= 128 {
		return saturationRows128
	}
	return saturationRows32
}

func shardCountForStrategy(strategy ShardStrategy, eligible, rows, saturationPoint int) int {
	if eligible <= 1 || strategy == ShardStrategySingle {
		return minInt(eligible, 1)
	}
	if strategy == ShardStrategyForced || rows <= 0 {
		return eligible
	}
	if saturationPoint <= 0 {
		return 1
	}
	count := rows / saturationPoint
	if count < 1 {
		return 1
	}
	if count > eligible {
		return eligible
	}
	return count
}

func minimumShardRows(total int, weights []float64, floor int) []int {
	rows := make([]int, len(weights))
	if len(weights) == 0 {
		return rows
	}
	minimum := floor * len(weights)
	if floor <= 0 || total < minimum {
		return nil
	}
	for i := range rows {
		rows[i] = floor
	}
	remaining := total - minimum
	totalWeight := 0.0
	for _, weight := range weights {
		totalWeight += weight
	}
	for i := range rows {
		if i == len(rows)-1 {
			rows[i] += remaining
			break
		}
		share := 0.0
		if totalWeight > 0 {
			share = weights[i] / totalWeight
		}
		portion := int(float64(total-minimum) * share)
		if portion > remaining {
			portion = remaining
		}
		rows[i] += portion
		remaining -= portion
	}
	return rows
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func shouldFallbackForProfitability(workload WorkloadEstimate, cfg Config) bool {
	if cfg.Mode == ModeStrictGPU || cfg.Strict {
		return false
	}
	if workload.Rows == 0 && workload.Bytes == 0 {
		return false
	}
	return workload.Rows > 0 && workload.Rows < 256 && workload.Bytes > 0 && workload.Bytes < 32*1024
}

func mergePolicyForDevices(devices []Device) MergePolicy {
	if len(devices) == 0 {
		return MergePolicyUnknown
	}
	backend := devices[0].Backend
	for _, device := range devices[1:] {
		if device.Backend != backend {
			return MergePolicyCPU
		}
	}
	if len(devices) > 1 {
		return MergePolicyBackendNative
	}
	return MergePolicyCPU
}

func shardWeight(device Device) float64 {
	weight := device.Score
	if weight <= 0 {
		weight = defaultDeviceScore(device)
	}
	switch device.MemoryClass {
	case MemoryClassDevice:
		weight *= 1.10
	case MemoryClassShared:
		weight *= 0.95
	}
	if device.BudgetBytes > 0 {
		weight *= 1 + minFloat64(float64(device.BudgetBytes)/(8*1024*1024*1024), 1.0)
	}
	if weight <= 0 {
		return 1
	}
	return weight
}

func proportionalIntShare(total, idx, count int, share float64, remaining *int) int {
	if idx == count-1 {
		out := *remaining
		*remaining = 0
		return out
	}
	portion := int(float64(total) * share)
	if portion < 0 {
		portion = 0
	}
	if portion > *remaining {
		portion = *remaining
	}
	*remaining -= portion
	return portion
}

func proportionalUint64Share(total uint64, idx, count int, share float64, remaining *uint64) uint64 {
	if idx == count-1 {
		out := *remaining
		*remaining = 0
		return out
	}
	portion := uint64(float64(total) * share)
	if portion > *remaining {
		portion = *remaining
	}
	*remaining -= portion
	return portion
}

func minFloat64(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func shardableDevices(devices []Device, cfg Config) []Device {
	if len(devices) == 0 {
		return nil
	}

	selected := make([]Device, 0, len(devices))
	preferredIDs := make(map[string]int, len(cfg.PreferredDevices))
	for idx, id := range cfg.PreferredDevices {
		preferredIDs[id] = idx
	}

	for _, device := range devices {
		if device.Type == DeviceTypeCPU || device.Backend == BackendCPU {
			continue
		}
		if len(preferredIDs) > 0 {
			if _, ok := preferredIDs[device.ID]; !ok {
				continue
			}
		}
		selected = append(selected, device)
	}

	sort.SliceStable(selected, func(i, j int) bool {
		left := selected[i]
		right := selected[j]
		if len(preferredIDs) > 0 {
			leftIdx, leftOK := preferredIDs[left.ID]
			rightIdx, rightOK := preferredIDs[right.ID]
			if leftOK && rightOK && leftIdx != rightIdx {
				return leftIdx < rightIdx
			}
			if leftOK != rightOK {
				return leftOK
			}
		}
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		return left.ID < right.ID
	})

	return selected
}
