package accel

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"time"
)

type residentCache struct {
	entries        map[string]CacheEntry
	evictedBuffers uint64
	evictedBytes   uint64
}

func newResidentCache() *residentCache {
	return &residentCache{
		entries: map[string]CacheEntry{},
	}
}

func (s *Session) CacheSnapshot() CacheSnapshot {
	if s == nil || s.cache == nil {
		return CacheSnapshot{}
	}

	budgets := s.cacheBudgetByDevice()
	snapshot := CacheSnapshot{
		EvictedBuffers: s.cache.evictedBuffers,
		EvictedBytes:   s.cache.evictedBytes,
	}
	deviceUsage := make(map[string]*CacheDeviceUsage, len(budgets))
	for deviceID, budget := range budgets {
		deviceUsage[deviceID] = &CacheDeviceUsage{
			DeviceID:    deviceID,
			BudgetBytes: budget,
		}
		snapshot.BudgetBytes += budget
	}

	keys := make([]string, 0, len(s.cache.entries))
	for key := range s.cache.entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	snapshot.Entries = make([]CacheEntry, 0, len(keys))
	for _, key := range keys {
		entry := cloneCacheEntry(s.cache.entries[key])
		snapshot.Entries = append(snapshot.Entries, entry)
		snapshot.ResidentBuffers++
		if len(entry.DeviceResidentBytes) > 0 {
			for deviceID, residentBytes := range entry.DeviceResidentBytes {
				snapshot.ResidentBytes += residentBytes
				usage, ok := deviceUsage[deviceID]
				if !ok {
					usage = &CacheDeviceUsage{DeviceID: deviceID}
					deviceUsage[deviceID] = usage
				}
				usage.ResidentBuffers++
				usage.ResidentBytes += residentBytes
			}
			continue
		}
		snapshot.ResidentBytes += entry.ResidentBytes
		for _, deviceID := range entry.DeviceIDs {
			usage, ok := deviceUsage[deviceID]
			if !ok {
				usage = &CacheDeviceUsage{DeviceID: deviceID}
				deviceUsage[deviceID] = usage
			}
			usage.ResidentBuffers++
			usage.ResidentBytes += entry.ResidentBytes
		}
	}

	deviceIDs := make([]string, 0, len(deviceUsage))
	for deviceID := range deviceUsage {
		deviceIDs = append(deviceIDs, deviceID)
	}
	sort.Strings(deviceIDs)
	snapshot.DeviceUsage = make([]CacheDeviceUsage, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		snapshot.DeviceUsage = append(snapshot.DeviceUsage, *deviceUsage[deviceID])
	}
	return snapshot
}

func (s *Session) cacheDataset(dataset *Dataset) {
	if s == nil || s.cache == nil || dataset == nil {
		return
	}

	now := time.Now()

	for idx, buffer := range dataset.Buffers {
		key := cacheKey(dataset, buffer, idx)
		s.cache.entries[key] = CacheEntry{
			Key:           key,
			DatasetName:   dataset.Name,
			DatasetID:     dataset.Fingerprint,
			Lineage:       dataset.Lineage,
			BufferName:    buffer.Name,
			Type:          buffer.Type,
			Len:           buffer.Len,
			ResidentBytes: estimateBufferResidentBytes(buffer),
			LastAccess:    now,
		}
	}

	s.enforceCacheBudget()
	s.updateCacheMetrics()
}

func (s *Session) updateCacheMetrics() {
	if s == nil || len(s.reports) == 0 {
		return
	}

	snapshot := s.CacheSnapshot()
	report := s.Report()
	if report.Metrics == nil {
		report.Metrics = map[string]float64{}
	}
	report.Metrics["cache.resident_buffers"] = float64(snapshot.ResidentBuffers)
	report.Metrics["cache.resident_bytes"] = float64(snapshot.ResidentBytes)
	report.Metrics["cache.budget_bytes"] = float64(snapshot.BudgetBytes)
	report.Metrics["cache.evicted_buffers"] = float64(snapshot.EvictedBuffers)
	report.Metrics["cache.evicted_bytes"] = float64(snapshot.EvictedBytes)
	s.reports[len(s.reports)-1] = cloneReport(report)
}

func (s *Session) enforceCacheBudget() {
	if s == nil || s.cache == nil || len(s.cache.entries) == 0 {
		return
	}

	for {
		budgets := s.cacheBudgetByDevice()
		if len(budgets) == 0 {
			return
		}
		totalBudget := uint64(0)
		for _, budget := range budgets {
			totalBudget += budget
		}
		usage := s.cacheUsageByDevice()
		overBudget := make(map[string]struct{})
		for deviceID, budget := range budgets {
			if budget == 0 {
				continue
			}
			if usage[deviceID] > budget {
				overBudget[deviceID] = struct{}{}
			}
		}
		if len(overBudget) == 0 && (totalBudget == 0 || s.totalResidentBytes() <= totalBudget) {
			return
		}

		evictKey := ""
		var oldest time.Time
		for key, entry := range s.cache.entries {
			if len(overBudget) > 0 && !entryTouchesDevices(entry, overBudget) {
				continue
			}
			if evictKey == "" || entry.LastAccess.Before(oldest) {
				evictKey = key
				oldest = entry.LastAccess
			}
		}
		if evictKey == "" {
			return
		}

		entry := s.cache.entries[evictKey]
		s.cache.evictedBuffers++
		s.cache.evictedBytes += entry.ResidentBytes
		delete(s.cache.entries, evictKey)
	}
}

func (s *Session) cacheBudgetByDevice() map[string]uint64 {
	budgets := map[string]uint64{}
	if s == nil {
		return budgets
	}
	for _, device := range shardableDevices(s.devices, s.cfg) {
		budgets[device.ID] = device.BudgetBytes
	}
	return budgets
}

func (s *Session) cacheUsageByDevice() map[string]uint64 {
	usage := map[string]uint64{}
	if s == nil || s.cache == nil {
		return usage
	}
	for _, entry := range s.cache.entries {
		if len(entry.DeviceResidentBytes) > 0 {
			for deviceID, residentBytes := range entry.DeviceResidentBytes {
				usage[deviceID] += residentBytes
			}
			continue
		}
		if len(entry.DeviceIDs) == 0 {
			continue
		}
		for _, deviceID := range entry.DeviceIDs {
			usage[deviceID] += entry.ResidentBytes
		}
	}
	return usage
}

func (s *Session) totalResidentBytes() uint64 {
	if s == nil || s.cache == nil {
		return 0
	}
	total := uint64(0)
	for _, entry := range s.cache.entries {
		total += entry.ResidentBytes
	}
	return total
}

func entryTouchesDevices(entry CacheEntry, targets map[string]struct{}) bool {
	for _, deviceID := range entry.DeviceIDs {
		if _, ok := targets[deviceID]; ok {
			return true
		}
	}
	return false
}

func cacheKey(dataset *Dataset, buffer Buffer, idx int) string {
	return fmt.Sprintf("%s:%s:%d:%s", dataset.Fingerprint, dataset.Lineage, idx, buffer.Name)
}

func estimateBufferResidentBytes(buffer Buffer) uint64 {
	valueBytes := uint64(0)
	switch values := buffer.Values.(type) {
	case []bool:
		valueBytes = uint64(len(values))
	case []int64:
		valueBytes = uint64(len(values) * 8)
	case []float64:
		valueBytes = uint64(len(values) * 8)
	case []string:
		if len(buffer.StringOffsets) > 0 || len(buffer.StringData) > 0 {
			valueBytes = uint64(len(buffer.StringOffsets)*4) + uint64(len(buffer.StringData))
		} else {
			offsetBytes := uint64((len(values) + 1) * 4)
			stringBytes := uint64(0)
			for _, value := range values {
				stringBytes += uint64(len(value))
			}
			valueBytes = offsetBytes + stringBytes
		}
	case []any:
		valueBytes = uint64(len(values) * 8)
	default:
		valueBytes = uint64(buffer.Len * 8)
	}
	if len(buffer.Validity) > 0 {
		return valueBytes + uint64(len(buffer.Validity))
	}
	return valueBytes + validityBitmapBytes(buffer.Len)
}

func validityBitmapBytes(length int) uint64 {
	if length <= 0 {
		return 0
	}
	return uint64((length + 7) / 8)
}

func assignDatasetFingerprint(dataset *Dataset) {
	if dataset == nil {
		return
	}
	dataset.Fingerprint = datasetFingerprint(dataset)
}

func datasetFingerprint(dataset *Dataset) string {
	if dataset == nil {
		return ""
	}

	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(dataset.Name))
	_, _ = hasher.Write([]byte{0})
	_, _ = hasher.Write([]byte(strconv.Itoa(dataset.Rows)))
	_, _ = hasher.Write([]byte{0})
	for _, buffer := range dataset.Buffers {
		_, _ = hasher.Write([]byte(buffer.Name))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(buffer.Type))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(strconv.Itoa(buffer.Len)))
		_, _ = hasher.Write([]byte{0})
		if len(buffer.Validity) > 0 {
			_, _ = hasher.Write(buffer.Validity)
		} else {
			for _, isNull := range buffer.Nulls {
				if isNull {
					_, _ = hasher.Write([]byte{1})
				} else {
					_, _ = hasher.Write([]byte{0})
				}
			}
		}
		_, _ = hasher.Write([]byte{0})
		if buffer.Type == DataTypeString && (len(buffer.StringOffsets) > 0 || len(buffer.StringData) > 0) {
			for _, offset := range buffer.StringOffsets {
				_, _ = hasher.Write([]byte(strconv.FormatUint(uint64(offset), 10)))
				_, _ = hasher.Write([]byte{0})
			}
			_, _ = hasher.Write(buffer.StringData)
		} else {
			_, _ = hasher.Write([]byte(strings.TrimSpace(fmt.Sprintf("%v", buffer.Values))))
		}
		_, _ = hasher.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hasher.Sum64())
}

func cloneCacheEntry(entry CacheEntry) CacheEntry {
	cloned := entry
	cloned.DeviceIDs = append([]string(nil), entry.DeviceIDs...)
	cloned.DeviceResidentBytes = cloneDeviceResidentBytes(entry.DeviceResidentBytes)
	return cloned
}
