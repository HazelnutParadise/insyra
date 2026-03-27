package accel

import "sort"

type ShardPlan struct {
	Accelerated      bool
	Backend          Backend
	DeviceIDs        []string
	TotalBudgetBytes uint64
	Heterogeneous    bool
	FallbackReason   FallbackReason
}

func (s *Session) PlanShardable() ShardPlan {
	report := s.Report()
	if len(s.devices) == 0 || !report.Accelerated {
		return ShardPlan{
			Accelerated:    false,
			Backend:        report.SelectedBackend,
			FallbackReason: report.FallbackReason,
		}
	}

	candidates := shardableDevices(s.devices, s.cfg)
	if len(candidates) == 0 {
		return ShardPlan{
			Accelerated:    false,
			Backend:        report.SelectedBackend,
			FallbackReason: report.FallbackReason,
		}
	}

	plan := ShardPlan{
		Accelerated:    true,
		Backend:        report.SelectedBackend,
		DeviceIDs:      make([]string, 0, len(candidates)),
		FallbackReason: FallbackReasonNone,
	}
	backendSet := map[Backend]struct{}{}
	for _, device := range candidates {
		plan.DeviceIDs = append(plan.DeviceIDs, device.ID)
		plan.TotalBudgetBytes += device.BudgetBytes
		backendSet[device.Backend] = struct{}{}
	}
	plan.Heterogeneous = len(backendSet) > 1
	return plan
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
