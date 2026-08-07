package nn

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra/internal/reftest"
)

//go:embed testdata/safetensors_fixture.py
var safeTensorsFixtureScript string

func TestLoadSafeTensorsRejectsMalformedStructure(t *testing.T) {
	validEntry := `{"x":{"dtype":"F32","shape":[1],"data_offsets":[0,4]}}`
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{name: "truncated header length", data: []byte{1, 2, 3}, want: "truncated"},
		{name: "header exceeds input", data: safeTensorsFileWithHeaderLength(100, nil), want: "header length"},
		{name: "bad JSON", data: safeTensorsFile(`{"x":`, nil), want: "invalid safetensors JSON"},
		{name: "overlapping regions", data: safeTensorsFile(`{"a":{"dtype":"F32","shape":[1],"data_offsets":[0,4]},"b":{"dtype":"F32","shape":[1],"data_offsets":[2,6]}}`, make([]byte, 6)), want: "overlaps"},
		{name: "gapped regions", data: safeTensorsFile(`{"x":{"dtype":"F32","shape":[1],"data_offsets":[1,5]}}`, make([]byte, 5)), want: "gap"},
		{name: "out of range offsets", data: safeTensorsFile(validEntry, make([]byte, 3)), want: "out-of-range"},
		{name: "element count mismatch", data: safeTensorsFile(`{"x":{"dtype":"F32","shape":[2],"data_offsets":[0,4]}}`, make([]byte, 4)), want: "want 8"},
		{name: "duplicate tensor name", data: safeTensorsFile(`{"x":{"dtype":"F32","shape":[1],"data_offsets":[0,4]},"x":{"dtype":"F32","shape":[1],"data_offsets":[4,8]}}`, make([]byte, 8)), want: "declared more than once"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("LoadSafeTensors panicked: %v", recovered)
				}
			}()
			_, err := LoadSafeTensors(bytes.NewReader(tc.data))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("LoadSafeTensors error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestLoadSafeTensorsReportsEveryUnsupportedDTypeAtOnce(t *testing.T) {
	header := `{"quant":{"dtype":"I8","shape":[2],"data_offsets":[0,2]},"double":{"dtype":"F64","shape":[1],"data_offsets":[2,10]}}`
	_, err := LoadSafeTensors(bytes.NewReader(safeTensorsFile(header, make([]byte, 10))))
	if err == nil {
		t.Fatal("LoadSafeTensors accepted unsupported dtypes")
	}
	message := err.Error()
	for _, want := range []string{"quant", "I8", "double", "F64"} {
		if !strings.Contains(message, want) {
			t.Errorf("error %q does not name unsupported item %q", message, want)
		}
	}
}

func TestLoadSafeTensorsToleratesMetadataAndLoadsExactValues(t *testing.T) {
	data := make([]byte, 8+8+2)
	binary.LittleEndian.PutUint32(data[0:4], 0x3fc00000)
	binary.LittleEndian.PutUint32(data[4:8], 0xc0200000)
	binary.LittleEndian.PutUint64(data[8:16], ^uint64(6))
	data[16], data[17] = 0, 1
	header := `{"__metadata__":{"format":"pt","note":"ignored"},"weights":{"dtype":"F32","shape":[2],"data_offsets":[0,8]},"indices":{"dtype":"I64","shape":[1],"data_offsets":[8,16]},"mask":{"dtype":"BOOL","shape":[2],"data_offsets":[16,18]}}`
	tensors, err := LoadSafeTensors(bytes.NewReader(safeTensorsFile(header, data)))
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	if got := tensors["weights"].Data(); !reflect.DeepEqual(got, []float32{1.5, -2.5}) {
		t.Fatalf("weights = %v, want [1.5 -2.5]", got)
	}
	indices, err := tensors["indices"].Int64Data()
	if err != nil || !reflect.DeepEqual(indices, []int64{-7}) {
		t.Fatalf("indices = %v, err %v, want [-7]", indices, err)
	}
	mask, err := tensors["mask"].BoolData()
	if err != nil || !reflect.DeepEqual(mask, []bool{false, true}) {
		t.Fatalf("mask = %v, err %v, want [false true]", mask, err)
	}
	if got := tensors["weights"].Shape(); !reflect.DeepEqual(got, []int{2}) {
		t.Fatalf("weights shape = %v, want [2]", got)
	}
}

func TestLoadSafeTensorsReferenceRoundTrip(t *testing.T) {
	python := requireSafeTensorsReference(t)
	if python == "" {
		return
	}
	path := filepath.Join(t.TempDir(), "fixture.safetensors")
	command := exec.Command(python, "-c", safeTensorsFixtureScript, path)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("safetensors fixture helper: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	var reference map[string]safeTensorFixtureReference
	if err := json.Unmarshal(stdout.Bytes(), &reference); err != nil {
		t.Fatalf("decode fixture stdout: %v\nstdout=%s\nstderr=%s", err, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String()))
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open safetensors fixture: %v", err)
	}
	tensors, err := LoadSafeTensors(file)
	closeErr := file.Close()
	if err != nil {
		t.Fatalf("LoadSafeTensors: %v", err)
	}
	if closeErr != nil {
		t.Fatalf("close safetensors fixture: %v", closeErr)
	}
	want := map[string]struct {
		shape []int
		dtype DType
		f32   []float32
		i64   []int64
		bools []bool
	}{
		"weights": {shape: []int{2, 3}, dtype: DTypeFloat32, f32: []float32{1.25, -2.5, 3.75, 4.5, 0, -6.25}},
		"indices": {shape: []int{2, 2}, dtype: DTypeInt64, i64: []int64{0, 3, 8, -1}},
		"mask":    {shape: []int{2, 3}, dtype: DTypeBool, bools: []bool{true, false, true, false, true, false}},
		"f16":     {shape: []int{7}, dtype: DTypeFloat32},
		"bf16":    {shape: []int{7}, dtype: DTypeFloat32},
	}
	if len(tensors) != len(want) {
		t.Fatalf("loaded tensor names = %v, want %v", mapKeys(tensors), mapKeys(want))
	}
	for name, expected := range want {
		got, ok := tensors[name]
		if !ok {
			t.Fatalf("missing tensor %q", name)
		}
		if got.DType() != expected.dtype || !reflect.DeepEqual(got.Shape(), expected.shape) {
			t.Fatalf("tensor %q metadata = dtype %s shape %v, want dtype %s shape %v", name, got.DType(), got.Shape(), expected.dtype, expected.shape)
		}
		switch expected.dtype {
		case DTypeFloat32:
			if referenceValues, ok := reference[name]; ok {
				assertSafeTensorReference(t, name, got.Data(), referenceValues)
				continue
			}
			if values := got.Data(); !reflect.DeepEqual(values, expected.f32) {
				t.Fatalf("tensor %q values = %v, want %v", name, values, expected.f32)
			}
		case DTypeInt64:
			values, err := got.Int64Data()
			if err != nil || !reflect.DeepEqual(values, expected.i64) {
				t.Fatalf("tensor %q values = %v, err %v, want %v", name, values, err, expected.i64)
			}
		case DTypeBool:
			values, err := got.BoolData()
			if err != nil || !reflect.DeepEqual(values, expected.bools) {
				t.Fatalf("tensor %q values = %v, err %v, want %v", name, values, err, expected.bools)
			}
		}
	}
}

type safeTensorFixtureReference struct {
	Shape  []int    `json:"shape"`
	Values []string `json:"values"`
	Bits   []uint32 `json:"bits"`
}

func assertSafeTensorReference(t *testing.T, name string, got []float32, reference safeTensorFixtureReference) {
	t.Helper()
	if len(got) != len(reference.Bits) || len(reference.Values) != len(reference.Bits) {
		t.Fatalf("tensor %q reference lengths = values %d, bits %d, loaded %d", name, len(reference.Values), len(reference.Bits), len(got))
	}
	if !reflect.DeepEqual(reference.Shape, []int{len(got)}) {
		t.Fatalf("tensor %q reference shape = %v, want [%d]", name, reference.Shape, len(got))
	}
	for index, value := range got {
		referenceValue := reference.Values[index]
		if referenceValue == "NaN" {
			if !math.IsNaN(float64(value)) {
				t.Fatalf("tensor %q value[%d] = %v, want NaN", name, index, value)
			}
			continue
		}
		if referenceValue == "+Inf" || referenceValue == "-Inf" {
			wantSign := 1
			if referenceValue == "-Inf" {
				wantSign = -1
			}
			if !math.IsInf(float64(value), wantSign) {
				t.Fatalf("tensor %q value[%d] = %v, want %s", name, index, value, referenceValue)
			}
			continue
		}
		parsed, err := strconv.ParseFloat(referenceValue, 32)
		if err != nil {
			t.Fatalf("tensor %q reference value[%d] %q: %v", name, index, referenceValue, err)
		}
		if math.Float32bits(value) != reference.Bits[index] || math.Float32bits(float32(parsed)) != reference.Bits[index] {
			t.Fatalf("tensor %q value[%d] bits = %#08x, reference bits %#08x", name, index, math.Float32bits(value), reference.Bits[index])
		}
	}
}

func requireSafeTensorsReference(t *testing.T) string {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		reftest.Missing(t, "python3", "the nn SafeTensors round-trip", err)
		return ""
	}
	command := exec.Command(python, "-c", "import torch, safetensors.torch")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		reftest.MissingOutput(t, "python3 with torch and safetensors", "the nn SafeTensors round-trip", err, stderr.Bytes())
		return ""
	}
	return python
}

func safeTensorsFile(header string, data []byte) []byte {
	result := make([]byte, 8+len(header)+len(data))
	binary.LittleEndian.PutUint64(result[:8], uint64(len(header)))
	copy(result[8:], header)
	copy(result[8+len(header):], data)
	return result
}

func safeTensorsFileWithHeaderLength(length uint64, data []byte) []byte {
	result := make([]byte, 8+len(data))
	binary.LittleEndian.PutUint64(result[:8], length)
	copy(result[8:], data)
	return result
}

func mapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
