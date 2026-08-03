package dl

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
)

// LoadSafeTensors reads a SafeTensors file into named tensors. The optional
// __metadata__ header entry is validated as the format's string-to-string map
// and then ignored because this API returns tensors only.
//
// SafeTensors input is untrusted: malformed headers and data return errors and
// never panic. F16 and BF16 are widened value-exactly to f32 because Tensor
// computation remains f32; F32, I64, and BOOL retain their native Tensor
// dtypes. Quantized and other unsupported dtypes remain refused by name.
func LoadSafeTensors(r io.Reader) (tensors map[string]*Tensor, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			tensors = nil
			err = fmt.Errorf("load safetensors: %v", recovered)
		}
	}()
	if r == nil {
		return nil, fmt.Errorf("load safetensors: reader is nil")
	}

	file, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read safetensors file: %w", err)
	}
	if len(file) < 8 {
		return nil, fmt.Errorf("safetensors file is truncated: need 8-byte header length, got %d bytes", len(file))
	}
	headerLength := binary.LittleEndian.Uint64(file[:8])
	if headerLength > uint64(len(file)-8) {
		return nil, fmt.Errorf("safetensors header length %d exceeds input size %d", headerLength, len(file)-8)
	}
	headerEnd := 8 + int(headerLength)
	entries, err := parseSafeTensorHeader(file[8:headerEnd])
	if err != nil {
		return nil, err
	}

	data := file[headerEnd:]
	regions := make([]safeTensorRegion, 0, len(entries))
	unsupported := make([]string, 0)
	for _, entry := range entries {
		if entry.begin > entry.end {
			return nil, fmt.Errorf("tensor %q has reversed data_offsets [%d, %d]", entry.name, entry.begin, entry.end)
		}
		if entry.end > uint64(len(data)) {
			return nil, fmt.Errorf("tensor %q has out-of-range data_offsets [%d, %d] for %d-byte data region", entry.name, entry.begin, entry.end, len(data))
		}
		byteSize, supported := safeTensorDTypeSize(entry.dtype)
		if !supported {
			unsupported = append(unsupported, fmt.Sprintf("tensor %q (%s)", entry.name, entry.dtype))
		} else {
			if entry.count > math.MaxUint64/byteSize {
				return nil, fmt.Errorf("tensor %q has an element byte length overflow for shape %v and dtype %s", entry.name, entry.shape, entry.dtype)
			}
			expected := entry.count * byteSize
			if entry.end-entry.begin != expected {
				return nil, fmt.Errorf("tensor %q has %d data bytes, want %d for %d elements of dtype %s", entry.name, entry.end-entry.begin, expected, entry.count, entry.dtype)
			}
		}
		regions = append(regions, safeTensorRegion{name: entry.name, begin: entry.begin, end: entry.end})
	}
	if err := validateSafeTensorRegions(regions, uint64(len(data))); err != nil {
		return nil, err
	}
	if len(unsupported) > 0 {
		return nil, fmt.Errorf("unsupported safetensors dtypes: %v", unsupported)
	}

	tensors = make(map[string]*Tensor, len(entries))
	for _, entry := range entries {
		payload := data[int(entry.begin):int(entry.end)]
		var tensor *Tensor
		switch entry.dtype {
		case "F32":
			values := make([]float32, int(entry.count))
			for index := range values {
				values[index] = math.Float32frombits(binary.LittleEndian.Uint32(payload[index*4:]))
			}
			tensor, err = newFloat32Tensor(entry.shape, values)
		case "F16":
			values := make([]float32, int(entry.count))
			for index := range values {
				values[index] = f16BitsToFloat32(binary.LittleEndian.Uint16(payload[index*2:]))
			}
			tensor, err = newFloat32Tensor(entry.shape, values)
		case "BF16":
			values := make([]float32, int(entry.count))
			for index := range values {
				values[index] = bf16BitsToFloat32(binary.LittleEndian.Uint16(payload[index*2:]))
			}
			tensor, err = newFloat32Tensor(entry.shape, values)
		case "I64":
			values := make([]int64, int(entry.count))
			for index := range values {
				values[index] = int64(binary.LittleEndian.Uint64(payload[index*8:]))
			}
			tensor, err = newInt64Tensor(entry.shape, values)
		case "BOOL":
			values := make([]bool, int(entry.count))
			for index, value := range payload {
				if value > 1 {
					return nil, fmt.Errorf("tensor %q has invalid BOOL byte %d at element %d", entry.name, value, index)
				}
				values[index] = value != 0
			}
			tensor, err = newBoolTensor(entry.shape, values)
		default:
			return nil, fmt.Errorf("tensor %q has unsupported dtype %s", entry.name, entry.dtype)
		}
		if err != nil {
			return nil, fmt.Errorf("tensor %q: %w", entry.name, err)
		}
		tensors[entry.name] = tensor
	}
	return tensors, nil
}

type safeTensorEntry struct {
	name  string
	dtype string
	shape []int
	count uint64
	begin uint64
	end   uint64
}

type safeTensorRegion struct {
	name       string
	begin, end uint64
}

type rawSafeTensorEntry struct {
	DType       json.RawMessage `json:"dtype"`
	Shape       json.RawMessage `json:"shape"`
	DataOffsets json.RawMessage `json:"data_offsets"`
}

func parseSafeTensorHeader(header []byte) ([]safeTensorEntry, error) {
	if len(header) == 0 || header[0] != '{' {
		return nil, fmt.Errorf("invalid safetensors JSON header: header must begin with '{'")
	}
	decoder := json.NewDecoder(bytes.NewReader(header))
	token, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("invalid safetensors JSON header: %w", err)
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return nil, fmt.Errorf("invalid safetensors JSON header: header is not an object")
	}

	entries := make([]safeTensorEntry, 0)
	seen := make(map[string]struct{})
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("invalid safetensors JSON header: %w", err)
		}
		name, ok := keyToken.(string)
		if !ok {
			return nil, fmt.Errorf("invalid safetensors JSON header: tensor name is not a string")
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, fmt.Errorf("tensor %q is declared more than once", name)
		}
		seen[name] = struct{}{}

		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return nil, fmt.Errorf("invalid safetensors JSON header for tensor %q: %w", name, err)
		}
		if name == "__metadata__" {
			var metadata map[string]string
			trimmed := bytes.TrimSpace(raw)
			if len(trimmed) == 0 || trimmed[0] != '{' || json.Unmarshal(trimmed, &metadata) != nil {
				return nil, fmt.Errorf("__metadata__ is not a string-to-string object")
			}
			continue
		}
		entry, err := parseSafeTensorEntry(name, raw)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if token, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("invalid safetensors JSON header: %w", err)
	} else if delimiter, ok := token.(json.Delim); !ok || delimiter != '}' {
		return nil, fmt.Errorf("invalid safetensors JSON header: object is not closed")
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("invalid safetensors JSON header: trailing JSON value")
		}
		return nil, fmt.Errorf("invalid safetensors JSON header: %w", err)
	}
	return entries, nil
}

func parseSafeTensorEntry(name string, raw json.RawMessage) (safeTensorEntry, error) {
	var result rawSafeTensorEntry
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return safeTensorEntry{}, fmt.Errorf("tensor %q has a non-object header entry", name)
	}
	if err := json.Unmarshal(trimmed, &result); err != nil {
		return safeTensorEntry{}, fmt.Errorf("invalid JSON for tensor %q: %w", name, err)
	}
	if len(result.DType) == 0 {
		return safeTensorEntry{}, fmt.Errorf("tensor %q has no dtype", name)
	}
	var dtype string
	if err := json.Unmarshal(result.DType, &dtype); err != nil || dtype == "" {
		return safeTensorEntry{}, fmt.Errorf("tensor %q has an invalid dtype", name)
	}
	if len(result.Shape) == 0 || bytes.TrimSpace(result.Shape)[0] != '[' {
		return safeTensorEntry{}, fmt.Errorf("tensor %q has an invalid shape", name)
	}
	var rawShape []int64
	if err := json.Unmarshal(result.Shape, &rawShape); err != nil {
		return safeTensorEntry{}, fmt.Errorf("tensor %q has an invalid shape: %w", name, err)
	}
	shape := make([]int, len(rawShape))
	count := uint64(1)
	for index, dimension := range rawShape {
		if dimension < 0 {
			return safeTensorEntry{}, fmt.Errorf("tensor %q has negative shape dimension %d at index %d", name, dimension, index)
		}
		if uint64(dimension) > uint64(maxInt()) {
			return safeTensorEntry{}, fmt.Errorf("tensor %q shape %v exceeds the Go slice size", name, rawShape)
		}
		shape[index] = int(dimension)
		if dimension != 0 {
			if count > uint64(maxInt())/uint64(dimension) {
				return safeTensorEntry{}, fmt.Errorf("tensor %q shape %v overflows element count", name, rawShape)
			}
			count *= uint64(dimension)
		}
	}
	if len(result.DataOffsets) == 0 || bytes.TrimSpace(result.DataOffsets)[0] != '[' {
		return safeTensorEntry{}, fmt.Errorf("tensor %q has invalid data_offsets", name)
	}
	var offsets []uint64
	if err := json.Unmarshal(result.DataOffsets, &offsets); err != nil || len(offsets) != 2 {
		return safeTensorEntry{}, fmt.Errorf("tensor %q has invalid data_offsets: want [begin, end]", name)
	}
	return safeTensorEntry{name: name, dtype: dtype, shape: shape, count: count, begin: offsets[0], end: offsets[1]}, nil
}

func safeTensorDTypeSize(dtype string) (uint64, bool) {
	switch dtype {
	case "F32":
		return 4, true
	case "F16", "BF16":
		return 2, true
	case "I64":
		return 8, true
	case "BOOL":
		return 1, true
	default:
		return 0, false
	}
}

func validateSafeTensorRegions(regions []safeTensorRegion, dataLength uint64) error {
	sort.Slice(regions, func(i, j int) bool {
		if regions[i].begin != regions[j].begin {
			return regions[i].begin < regions[j].begin
		}
		if regions[i].end != regions[j].end {
			return regions[i].end < regions[j].end
		}
		return regions[i].name < regions[j].name
	})
	cursor := uint64(0)
	lastName := ""
	for _, region := range regions {
		if region.begin < cursor {
			return fmt.Errorf("tensor %q overlaps tensor %q in the data region", region.name, lastName)
		}
		if region.begin > cursor {
			return fmt.Errorf("tensor %q leaves a data-region gap from byte %d to %d", region.name, cursor, region.begin)
		}
		if region.end > cursor {
			cursor = region.end
			lastName = region.name
		}
	}
	if cursor != dataLength {
		if lastName == "" {
			return fmt.Errorf("safetensors data region has %d unindexed bytes", dataLength)
		}
		return fmt.Errorf("tensor %q leaves a trailing data-region gap from byte %d to %d", lastName, cursor, dataLength)
	}
	return nil
}
