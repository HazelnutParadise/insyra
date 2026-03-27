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
	entries map[string]CacheEntry
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

	report := s.Report()
	plan := s.PlanShardable()
	snapshot := CacheSnapshot{
		BudgetBytes: plan.TotalBudgetBytes,
	}
	if snapshot.BudgetBytes == 0 {
		snapshot.BudgetBytes = uint64(report.Metrics["memory.budget_bytes_selected"])
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
		snapshot.ResidentBytes += entry.ResidentBytes
	}
	return snapshot
}

func (s *Session) cacheDataset(dataset *Dataset) {
	if s == nil || s.cache == nil || dataset == nil {
		return
	}

	now := time.Now()
	deviceIDs := s.PlanShardable().DeviceIDs
	if len(deviceIDs) == 0 {
		deviceIDs = append([]string(nil), s.Report().SelectedDeviceIDs...)
	}

	for idx, buffer := range dataset.Buffers {
		key := cacheKey(dataset, buffer, idx)
		s.cache.entries[key] = CacheEntry{
			Key:           key,
			DatasetName:   dataset.Name,
			DatasetID:     dataset.Fingerprint,
			BufferName:    buffer.Name,
			Type:          buffer.Type,
			Len:           buffer.Len,
			ResidentBytes: estimateBufferResidentBytes(buffer),
			DeviceIDs:     append([]string(nil), deviceIDs...),
			LastAccess:    now,
		}
	}

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
	s.reports[len(s.reports)-1] = cloneReport(report)
}

func cacheKey(dataset *Dataset, buffer Buffer, idx int) string {
	return fmt.Sprintf("%s:%d:%s", dataset.Fingerprint, idx, buffer.Name)
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
		offsetBytes := uint64((len(values) + 1) * 4)
		stringBytes := uint64(0)
		for _, value := range values {
			stringBytes += uint64(len(value))
		}
		valueBytes = offsetBytes + stringBytes
	case []any:
		valueBytes = uint64(len(values) * 8)
	default:
		valueBytes = uint64(buffer.Len * 8)
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
		for _, isNull := range buffer.Nulls {
			if isNull {
				_, _ = hasher.Write([]byte{1})
			} else {
				_, _ = hasher.Write([]byte{0})
			}
		}
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(strings.TrimSpace(fmt.Sprintf("%v", buffer.Values))))
		_, _ = hasher.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hasher.Sum64())
}

func cloneCacheEntry(entry CacheEntry) CacheEntry {
	cloned := entry
	cloned.DeviceIDs = append([]string(nil), entry.DeviceIDs...)
	return cloned
}
