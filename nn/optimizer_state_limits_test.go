package nn

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

const optimizerStatePoisonLimit = 1 << 20

type optimizerStateBoundedPoisonReader struct {
	prefix      []byte
	offset      int
	poisonRead  int
	poisonLimit int
	poison      error
}

func newOptimizerStateBoundedPoisonReader(prefix []byte) *optimizerStateBoundedPoisonReader {
	return &optimizerStateBoundedPoisonReader{
		prefix:      prefix,
		poisonLimit: optimizerStatePoisonLimit,
		poison:      errors.New("bounded optimizer-state poison reader exhausted"),
	}
}

func (r *optimizerStateBoundedPoisonReader) Read(p []byte) (int, error) {
	if r.offset < len(r.prefix) {
		n := copy(p, r.prefix[r.offset:])
		r.offset += n
		return n, nil
	}
	if r.poisonRead == r.poisonLimit {
		return 0, r.poison
	}
	n := min(len(p), r.poisonLimit-r.poisonRead)
	clear(p[:n])
	r.poisonRead += n
	return n, nil
}

func (r *optimizerStateBoundedPoisonReader) bytesRead() int {
	return r.offset + r.poisonRead
}

func optimizerStateRequireAtomicReaderRejection(t *testing.T, tape *Tape, named map[string]*Parameter, reader *optimizerStateBoundedPoisonReader) {
	t.Helper()
	before := optimizerStateSnapshot(named, tape)
	err := tape.LoadOptimizerState(reader, named)
	if err == nil {
		t.Fatal("invalid optimizer state was accepted")
	}
	if after := optimizerStateSnapshot(named, tape); !reflect.DeepEqual(after, before) {
		t.Fatalf("rejected import partially mutated parameters:\nbefore: %#v\nafter:  %#v", before, after)
	}
	if errors.Is(err, reader.poison) {
		t.Fatalf("optimizer-state loader consumed poison input after %d bytes instead of rejecting its header: %v", reader.bytesRead(), err)
	}
}

func optimizerStateWithEditedHeader(t *testing.T, encoded []byte, edit func(map[string]json.RawMessage, int)) ([]byte, []byte) {
	t.Helper()
	if len(encoded) < 8 {
		t.Fatal("optimizer-state fixture is truncated")
	}
	headerLength := binary.LittleEndian.Uint64(encoded[:8])
	if headerLength > uint64(len(encoded)-8) {
		t.Fatal("optimizer-state fixture has an invalid header length")
	}
	headerEnd := 8 + int(headerLength)
	var header map[string]json.RawMessage
	if err := json.Unmarshal(encoded[8:headerEnd], &header); err != nil {
		t.Fatalf("decode optimizer-state fixture header: %v", err)
	}
	data := append([]byte(nil), encoded[headerEnd:]...)
	edit(header, len(data))
	headerJSON, err := json.MarshalIndent(header, "", "  ")
	if err != nil {
		t.Fatalf("encode edited optimizer-state header: %v", err)
	}
	result := make([]byte, 8+len(headerJSON)+len(data))
	binary.LittleEndian.PutUint64(result[:8], uint64(len(headerJSON)))
	copy(result[8:], headerJSON)
	copy(result[8+len(headerJSON):], data)
	return result, result[:8+len(headerJSON)]
}

func optimizerStateLimitedReadFixture(t *testing.T) (*Tape, map[string]*Parameter, []byte) {
	t.Helper()
	source, sourceNamed := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, source, sourceNamed, "adam", 1)
	encoded := optimizerStateExport(t, source, sourceNamed)

	target, targetNamed := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, target, targetNamed, "adam", 2)
	optimizerStateRunSteps(t, target, targetNamed, "momentum", 1)
	return target, targetNamed, encoded
}

func TestOptimizerStateRejectsInvalidHeadersBeforeReadingPayload(t *testing.T) {
	t.Run("oversized header length", func(t *testing.T) {
		target, named, _ := optimizerStateLimitedReadFixture(t)
		for _, length := range []uint64{1 << 20, 1 << 63} {
			prefix := make([]byte, 8)
			binary.LittleEndian.PutUint64(prefix, length)
			reader := newOptimizerStateBoundedPoisonReader(prefix)

			optimizerStateRequireAtomicReaderRejection(t, target, named, reader)
			if got := reader.bytesRead(); got > 8 {
				t.Fatalf("header length %d consumed %d bytes; want at most its 8-byte prefix", length, got)
			}
		}
	})

	t.Run("unknown tensor key", func(t *testing.T) {
		target, named, valid := optimizerStateLimitedReadFixture(t)
		malformed, header := optimizerStateWithEditedHeader(t, valid, func(entries map[string]json.RawMessage, dataLength int) {
			raw, err := json.Marshal(optimizerStateWireHeader{
				DType:       "F32",
				Shape:       []int{1},
				DataOffsets: [2]uint64{uint64(dataLength), uint64(dataLength + 4)},
			})
			if err != nil {
				t.Fatal(err)
			}
			entries["unexpected/tensor"] = raw
		})
		malformed = append(malformed, make([]byte, 4)...)
		if _, err := LoadSafeTensors(bytes.NewReader(malformed)); err != nil {
			t.Fatalf("unknown-key fixture is not a valid SafeTensors file: %v", err)
		}
		reader := newOptimizerStateBoundedPoisonReader(header)

		optimizerStateRequireAtomicReaderRejection(t, target, named, reader)
		if got := reader.bytesRead(); got > len(header) {
			t.Fatalf("unknown tensor key consumed payload: read %d bytes, header is %d bytes", got, len(header))
		}
	})

	t.Run("wrong slot shape", func(t *testing.T) {
		target, named, valid := optimizerStateLimitedReadFixture(t)
		malformed, headerPrefix := optimizerStateWithEditedHeader(t, valid, func(entries map[string]json.RawMessage, _ int) {
			key := optimizerStateKey(optimizerStateMatrixName, "adam_m")
			var entry optimizerStateWireHeader
			if err := json.Unmarshal(entries[key], &entry); err != nil {
				t.Fatal(err)
			}
			entry.Shape = []int{4}
			raw, err := json.Marshal(entry)
			if err != nil {
				t.Fatal(err)
			}
			entries[key] = raw
		})
		if _, err := LoadSafeTensors(bytes.NewReader(malformed)); err != nil {
			t.Fatalf("wrong-slot-shape fixture is not a valid SafeTensors file: %v", err)
		}
		reader := newOptimizerStateBoundedPoisonReader(headerPrefix)

		optimizerStateRequireAtomicReaderRejection(t, target, named, reader)
		if got := reader.bytesRead(); got > len(headerPrefix) {
			t.Fatalf("wrong slot shape consumed payload: read %d bytes, header is %d bytes", got, len(headerPrefix))
		}
	})
}

func TestOptimizerStateBoundedTrailingInput(t *testing.T) {
	target, named, valid := optimizerStateLimitedReadFixture(t)
	reader := newOptimizerStateBoundedPoisonReader(valid)
	optimizerStateRequireAtomicReaderRejection(t, target, named, reader)
	if got, want := reader.bytesRead(), len(valid)+1; got > want {
		t.Fatalf("trailing input consumed %d bytes, want at most checkpoint plus one byte (%d)", got, want)
	}
}

func TestOptimizerStateReadWhitespaceMetadataScalarAndEmptySlots(t *testing.T) {
	source := NewTape()
	t.Cleanup(source.Close)
	scalarValue, err := NewTensor(nil, []float32{0.375})
	if err != nil {
		t.Fatal(err)
	}
	emptyValue, err := NewTensor([]int{0, 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := source.Param(scalarValue)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := source.Param(emptyValue)
	if err != nil {
		t.Fatal(err)
	}
	named := map[string]*Parameter{"scalar": scalar, "empty": empty}

	scalarOutput, err := source.Mul(scalar.Value(), scalar.Value())
	if err != nil {
		t.Fatal(err)
	}
	scalarUpstream, err := NewTensor(nil, []float32{1})
	if err != nil {
		t.Fatal(err)
	}
	if err := source.BackwardFrom(scalarOutput, scalarUpstream); err != nil {
		t.Fatal(err)
	}
	if err := source.Adam(0.01); err != nil {
		t.Fatal(err)
	}
	source.Clear()
	emptyOutput, err := source.Add(empty.Value(), empty.Value())
	if err != nil {
		t.Fatal(err)
	}
	if err := source.BackwardFrom(emptyOutput, empty.Value()); err != nil {
		t.Fatal(err)
	}
	if err := source.Adam(0.01); err != nil {
		t.Fatal(err)
	}

	canonical := optimizerStateExport(t, source, named)
	withMetadata, _ := optimizerStateWithEditedHeader(t, canonical, func(entries map[string]json.RawMessage, _ int) {
		metadata, err := json.Marshal(map[string]string{"note": " whitespace is valid "})
		if err != nil {
			t.Fatal(err)
		}
		entries["__metadata__"] = metadata
	})
	if _, err := LoadSafeTensors(bytes.NewReader(withMetadata)); err != nil {
		t.Fatalf("whitespace and metadata fixture is not valid SafeTensors: %v", err)
	}

	target := NewTape()
	t.Cleanup(target.Close)
	restoredScalar, err := NewTensor(nil, []float32{0})
	if err != nil {
		t.Fatal(err)
	}
	restoredEmpty, err := NewTensor([]int{0, 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	restoredScalarParameter, err := target.Param(restoredScalar)
	if err != nil {
		t.Fatal(err)
	}
	restoredEmptyParameter, err := target.Param(restoredEmpty)
	if err != nil {
		t.Fatal(err)
	}
	restored := map[string]*Parameter{"scalar": restoredScalarParameter, "empty": restoredEmptyParameter}
	if err := target.LoadOptimizerState(bytes.NewReader(withMetadata), restored); err != nil {
		t.Fatalf("valid checkpoint with whitespace, metadata, scalar and empty slots was rejected: %v", err)
	}
	if got := optimizerStateExport(t, target, restored); !bytes.Equal(got, canonical) {
		t.Fatalf("round-tripped scalar and empty slots differ:\n got %s\nwant %s", fmt.Sprintf("%x", got), fmt.Sprintf("%x", canonical))
	}
}

func TestOptimizerStateReadRejectsTerminalIOError(t *testing.T) {
	target, named, valid := optimizerStateLimitedReadFixture(t)
	failure := errors.New("checkpoint source failed at EOF")
	before := optimizerStateSnapshot(named, target)
	readers := []io.Reader{
		io.MultiReader(bytes.NewReader(valid), optimizerStateTerminalErrorReader{failure}),
		&optimizerStateFinalBytesErrorReader{bytes.NewReader(valid), failure},
	}
	for _, reader := range readers {
		if err := target.LoadOptimizerState(reader, named); !errors.Is(err, failure) {
			t.Fatalf("terminal read error = %v, want %v", err, failure)
		}
		if !reflect.DeepEqual(optimizerStateSnapshot(named, target), before) {
			t.Fatal("terminal read error changed target state")
		}
	}
}

type optimizerStateTerminalErrorReader struct{ err error }

func (r optimizerStateTerminalErrorReader) Read([]byte) (int, error) { return 0, r.err }

func TestOptimizerStateReadBudgetScalesWithNamesAndEmptyRank(t *testing.T) {
	shape := make([]int, 256)
	for i := range shape {
		shape[i] = 1
	}
	shape[len(shape)-1] = 0
	name := strings.Repeat("β/weight", 4096)
	source, target := NewTape(), NewTape()
	t.Cleanup(source.Close)
	t.Cleanup(target.Close)
	makeParam := func(tape *Tape) *Parameter {
		value, err := NewTensor(shape, nil)
		if err != nil {
			t.Fatal(err)
		}
		parameter, err := tape.Param(value)
		if err != nil {
			t.Fatal(err)
		}
		return parameter
	}
	original, restored := makeParam(source), makeParam(target)
	original.adamM, original.adamV = []float32{}, []float32{}
	original.velocity, original.adamStep = []float32{}, 3
	named := map[string]*Parameter{name: original}
	canonical := optimizerStateExport(t, source, named)
	if len(canonical) <= 64<<10 {
		t.Fatal("fixture must exceed a fixed 64 KiB file limit")
	}
	if err := target.LoadOptimizerState(bytes.NewReader(canonical), map[string]*Parameter{name: restored}); err != nil {
		t.Fatalf("writer output for long names and high-rank empty parameter was refused: %v", err)
	}
	if got := optimizerStateExport(t, target, map[string]*Parameter{name: restored}); !bytes.Equal(got, canonical) {
		t.Fatal("long-name high-rank empty optimizer checkpoint changed")
	}
}

// A source can report an error with the final data bytes. ReadFull alone drops
// that error when the destination fills; optimizer imports must still refuse it.
type optimizerStateFinalBytesErrorReader struct {
	reader *bytes.Reader
	err    error
}

func (r *optimizerStateFinalBytesErrorReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.reader.Len() == 0 {
		return n, r.err
	}
	return n, err
}
