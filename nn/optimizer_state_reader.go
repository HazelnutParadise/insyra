package nn

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

type optimizerStateReadObserver struct {
	reader io.Reader
	err    error
}

func (r *optimizerStateReadObserver) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if err != nil && !errors.Is(err, io.EOF) && r.err == nil {
		r.err = err
	}
	return n, err
}

type optimizerStateReadDescriptor struct {
	dtype    string
	shape    []int
	required bool
}

// optimizerStateReadCheckpoint bounds reads using the registered parameters,
// checks the format version, the precision and every wire descriptor before
// payload allocation, and reads one trailing byte at most. It is the only place
// the file's layout is validated, and it never mutates target parameters.
func optimizerStateReadCheckpoint(r io.Reader, targets []optimizerStateTarget, precision string) ([]safeTensorEntry, []byte, error) {
	if r == nil {
		return nil, nil, fmt.Errorf("optimizer state: reader is nil")
	}

	maxHeaderLength := 64 << 10
	for _, target := range targets {
		if target.parameter == nil || target.parameter.value == nil {
			return nil, nil, fmt.Errorf("optimizer state: parameter %q has no value tensor", target.name)
		}
		prefixLength := len(makeOptimizerStatePrefix(target.name))
		rank := len(target.parameter.value.shape)
		if prefixLength > maxInt()/6 || rank == maxInt() {
			return nil, nil, fmt.Errorf("optimizer state: header limit overflows for parameter %q", target.name)
		}
		prefixTerm := prefixLength * 6
		if rank+1 > maxInt()/24 {
			return nil, nil, fmt.Errorf("optimizer state: header limit overflows for parameter %q", target.name)
		}
		rankTerm := (rank + 1) * 24
		if prefixTerm > maxInt()-256 || rankTerm > maxInt()-256-prefixTerm {
			return nil, nil, fmt.Errorf("optimizer state: header limit overflows for parameter %q", target.name)
		}
		targetTerm := 256 + prefixTerm + rankTerm
		if targetTerm > maxInt()/6 {
			return nil, nil, fmt.Errorf("optimizer state: header limit overflows for parameter %q", target.name)
		}
		targetTerm *= 6
		if targetTerm > maxInt()-maxHeaderLength {
			return nil, nil, fmt.Errorf("optimizer state: header limit overflows for parameter %q", target.name)
		}
		maxHeaderLength += targetTerm
	}

	reader := &optimizerStateReadObserver{reader: r}
	var lengthBytes [8]byte
	if _, err := io.ReadFull(reader, lengthBytes[:]); err != nil {
		if reader.err != nil {
			err = reader.err
		}
		return nil, nil, fmt.Errorf("optimizer state: read checkpoint header length: %w", err)
	}
	if reader.err != nil {
		return nil, nil, fmt.Errorf("optimizer state: read checkpoint header length: %w", reader.err)
	}
	headerLength := binary.LittleEndian.Uint64(lengthBytes[:])
	if headerLength > uint64(maxHeaderLength) {
		return nil, nil, fmt.Errorf("optimizer state: SafeTensors header length %d exceeds limit %d", headerLength, maxHeaderLength)
	}
	header := make([]byte, int(headerLength))
	if _, err := io.ReadFull(reader, header); err != nil {
		if reader.err != nil {
			err = reader.err
		}
		return nil, nil, fmt.Errorf("optimizer state: read checkpoint header: %w", err)
	}
	if reader.err != nil {
		return nil, nil, fmt.Errorf("optimizer state: read checkpoint header: %w", reader.err)
	}
	entries, metadata, err := parseSafeTensorHeader(header)
	if err != nil {
		return nil, nil, fmt.Errorf("optimizer state: invalid SafeTensors header: %w", err)
	}
	// The version comes first, so a file from a newer release is reported as
	// such rather than by the first tensor this release does not know.
	version, ok := metadata[optimizerStateVersionKey]
	if !ok {
		return nil, nil, fmt.Errorf("optimizer state: not an optimizer-state file (no %s in its metadata)", optimizerStateVersionKey)
	}
	if version != optimizerStateVersion {
		return nil, nil, fmt.Errorf("optimizer state: unsupported version %q, this release reads version %s", version, optimizerStateVersion)
	}
	if metadata[optimizerStatePrecisionKey] != precision {
		return nil, nil, fmt.Errorf("optimizer state: checkpoint precision %q does not match tape precision %q", metadata[optimizerStatePrecisionKey], precision)
	}

	allowed := make(map[string]optimizerStateReadDescriptor)
	for _, target := range targets {
		prefix := makeOptimizerStatePrefix(target.name)
		shape := target.parameter.value.shape
		descriptors := []struct {
			name     string
			dtype    string
			shape    []int
			required bool
		}{
			{name: prefix + "shape", dtype: "I64", shape: []int{len(shape)}, required: true},
			{name: prefix + "step", dtype: "I64", required: true},
			{name: prefix + "adam_m", dtype: "F32", shape: shape},
			{name: prefix + "adam_v", dtype: "F32", shape: shape},
			{name: prefix + "velocity", dtype: "F32", shape: shape},
		}
		for _, descriptor := range descriptors {
			if _, exists := allowed[descriptor.name]; exists {
				return nil, nil, fmt.Errorf("optimizer state: duplicate expected tensor name %q", descriptor.name)
			}
			allowed[descriptor.name] = optimizerStateReadDescriptor{
				dtype:    descriptor.dtype,
				shape:    descriptor.shape,
				required: descriptor.required,
			}
		}
	}

	found := make(map[string]bool, len(allowed))
	regions := make([]safeTensorRegion, 0, len(entries))
	maxEnd := uint64(0)
	for _, entry := range entries {
		descriptor, exists := allowed[entry.name]
		if !exists {
			return nil, nil, fmt.Errorf("optimizer state: unknown tensor %q", entry.name)
		}
		found[entry.name] = true
		if entry.dtype != descriptor.dtype {
			return nil, nil, fmt.Errorf("optimizer state: tensor %q has wire dtype %s, want %s", entry.name, entry.dtype, descriptor.dtype)
		}
		if !sameShape(entry.shape, descriptor.shape) {
			return nil, nil, fmt.Errorf("optimizer state: tensor %q has shape %v, want %v", entry.name, entry.shape, descriptor.shape)
		}
		if entry.begin > entry.end {
			return nil, nil, fmt.Errorf("optimizer state: tensor %q has reversed data_offsets [%d, %d]", entry.name, entry.begin, entry.end)
		}
		if entry.end > uint64(maxInt()) {
			return nil, nil, fmt.Errorf("optimizer state: tensor %q data offset %d exceeds the Go slice size", entry.name, entry.end)
		}
		byteSize, supported := safeTensorDTypeSize(entry.dtype)
		if !supported || entry.count > math.MaxUint64/byteSize {
			return nil, nil, fmt.Errorf("optimizer state: tensor %q has an invalid element byte length", entry.name)
		}
		expectedLength := entry.count * byteSize
		if entry.end-entry.begin != expectedLength {
			return nil, nil, fmt.Errorf("optimizer state: tensor %q has %d data bytes, want %d", entry.name, entry.end-entry.begin, expectedLength)
		}
		regions = append(regions, safeTensorRegion{name: entry.name, begin: entry.begin, end: entry.end})
		if entry.end > maxEnd {
			maxEnd = entry.end
		}
	}
	for name, descriptor := range allowed {
		if descriptor.required && !found[name] {
			return nil, nil, fmt.Errorf("optimizer state: missing tensor %q", name)
		}
	}
	for _, target := range targets {
		prefix := makeOptimizerStatePrefix(target.name)
		hasFirst, hasSecond := found[prefix+"adam_m"], found[prefix+"adam_v"]
		if hasFirst != hasSecond {
			return nil, nil, fmt.Errorf("optimizer state: parameter %q must have both Adam moments or neither", target.name)
		}
	}
	if err := validateSafeTensorRegions(regions, maxEnd); err != nil {
		return nil, nil, fmt.Errorf("optimizer state: invalid data regions: %w", err)
	}

	data := make([]byte, int(maxEnd))
	if _, err := io.ReadFull(reader, data); err != nil {
		if reader.err != nil {
			err = reader.err
		}
		return nil, nil, fmt.Errorf("optimizer state: read checkpoint data: %w", err)
	}
	if reader.err != nil {
		return nil, nil, fmt.Errorf("optimizer state: read checkpoint data: %w", reader.err)
	}

	var trailing [1]byte
	n, err := io.ReadFull(reader, trailing[:])
	if n != 0 {
		return nil, nil, fmt.Errorf("optimizer state: checkpoint has trailing data")
	}
	if reader.err != nil {
		return nil, nil, fmt.Errorf("optimizer state: read checkpoint terminator: %w", reader.err)
	}
	if !errors.Is(err, io.EOF) {
		return nil, nil, fmt.Errorf("optimizer state: read checkpoint terminator: %w", err)
	}
	return entries, data, nil
}
