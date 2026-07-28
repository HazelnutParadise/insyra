package accel

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
)

type residentCache struct {
	entries        map[string]CacheEntry
	evictedBuffers uint64
	evictedBytes   uint64
	nextOrdinal    uint64
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
		s.cache.nextOrdinal++
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
			accessOrdinal: s.cache.nextOrdinal,
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
			if evictKey == "" || entry.LastAccess.Before(oldest) || (entry.LastAccess.Equal(oldest) && entryOlderThan(entry, s.cache.entries[evictKey], key, evictKey)) {
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

func entryOlderThan(candidate CacheEntry, incumbent CacheEntry, candidateKey string, incumbentKey string) bool {
	if candidate.accessOrdinal != incumbent.accessOrdinal {
		return candidate.accessOrdinal < incumbent.accessOrdinal
	}
	return candidateKey < incumbentKey
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

// fingerprintChunkBytes is the scratch buffer the value encoder fills before
// handing bytes to the hasher, so a column of any size costs one allocation.
const fingerprintChunkBytes = 32 << 10

// fingerprintHasher hashes a dataset's identity. Values go in as their binary
// representation rather than as text: rendering a 4 Mi float64 column with
// fmt.Sprintf("%v", ...) produced roughly 80 MB of decimal digits and cost more
// than seventy times the GPU work it was supposed to be book-keeping for.
type fingerprintHasher struct {
	digest *xxhash.Digest
	buf    []byte
}

func newFingerprintHasher() *fingerprintHasher {
	return &fingerprintHasher{
		digest: xxhash.New(),
		buf:    make([]byte, 0, fingerprintChunkBytes),
	}
}

// flush must run before any direct write so buffered bytes keep their position
// in the stream.
func (h *fingerprintHasher) flush() {
	if len(h.buf) > 0 {
		_, _ = h.digest.Write(h.buf)
		h.buf = h.buf[:0]
	}
}

func (h *fingerprintHasher) writeRaw(b []byte) {
	h.flush()
	_, _ = h.digest.Write(b)
}

func (h *fingerprintHasher) writeString(s string) {
	h.flush()
	_, _ = h.digest.WriteString(s)
}

func (h *fingerprintHasher) writeByte(b byte) {
	if len(h.buf)+1 > cap(h.buf) {
		h.flush()
	}
	h.buf = append(h.buf, b)
}

func (h *fingerprintHasher) writeUint64(v uint64) {
	if len(h.buf)+8 > cap(h.buf) {
		h.flush()
	}
	h.buf = binary.LittleEndian.AppendUint64(h.buf, v)
}

func (h *fingerprintHasher) sum() uint64 {
	h.flush()
	return h.digest.Sum64()
}

func datasetFingerprint(dataset *Dataset) string {
	if dataset == nil {
		return ""
	}

	hasher := newFingerprintHasher()
	hasher.writeString(dataset.Name)
	hasher.writeByte(0)
	hasher.writeUint64(uint64(dataset.Rows))
	for _, buffer := range dataset.Buffers {
		hasher.writeString(buffer.Name)
		hasher.writeByte(0)
		hasher.writeString(string(buffer.Type))
		hasher.writeByte(0)
		hasher.writeUint64(uint64(buffer.Len))
		if len(buffer.Validity) > 0 {
			hasher.writeRaw(buffer.Validity)
		} else {
			for _, isNull := range buffer.Nulls {
				if isNull {
					hasher.writeByte(1)
				} else {
					hasher.writeByte(0)
				}
			}
		}
		hasher.writeByte(0)
		writeBufferValues(hasher, buffer)
		hasher.writeByte(0)
	}
	return fmt.Sprintf("%x", hasher.sum())
}

func writeBufferValues(hasher *fingerprintHasher, buffer Buffer) {
	if buffer.Type == DataTypeString && (len(buffer.StringOffsets) > 0 || len(buffer.StringData) > 0) {
		for _, offset := range buffer.StringOffsets {
			hasher.writeUint64(uint64(offset))
		}
		hasher.writeRaw(buffer.StringData)
		return
	}

	switch values := buffer.Values.(type) {
	case []float64:
		for _, value := range values {
			hasher.writeUint64(math.Float64bits(value))
		}
	case []int64:
		for _, value := range values {
			hasher.writeUint64(uint64(value))
		}
	case []bool:
		for _, value := range values {
			if value {
				hasher.writeByte(1)
			} else {
				hasher.writeByte(0)
			}
		}
	case []string:
		// Length-prefixed, because concatenating the values would give
		// ["ab", "c"] and ["a", "bc"] the same bytes.
		for _, value := range values {
			hasher.writeUint64(uint64(len(value)))
			hasher.writeString(value)
		}
	default:
		// Untyped columns keep the per-element formatting path. projectValues
		// only produces these for genuinely mixed data, which is not eligible
		// for device execution anyway.
		hasher.writeString(strings.TrimSpace(fmt.Sprintf("%v", buffer.Values)))
	}
}

func cloneCacheEntry(entry CacheEntry) CacheEntry {
	cloned := entry
	cloned.DeviceIDs = append([]string(nil), entry.DeviceIDs...)
	cloned.DeviceResidentBytes = cloneDeviceResidentBytes(entry.DeviceResidentBytes)
	return cloned
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
