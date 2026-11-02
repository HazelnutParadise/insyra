package nn

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSafeTensorsScalarAndEmptyRoundTrip(t *testing.T) {
	for _, shape := range [][]int{nil, {0}, {0, 2}, {2, 0}} {
		count := 0
		if len(shape) == 0 {
			count = 1
		}
		f32, _ := NewTensor(shape, make([]float32, count))
		i64, _ := NewInt64Tensor(shape, make([]int64, count))
		boolean, _ := NewBoolTensor(shape, make([]bool, count))
		nonempty, _ := NewTensor([]int{2}, []float32{1, -2})
		original := map[string]*Tensor{"f32": f32, "i64": i64, "bool": boolean, "nonempty": nonempty}
		var encoded bytes.Buffer
		if err := SaveSafeTensors(&encoded, original); err != nil {
			t.Fatal(err)
		}
		loaded, err := LoadSafeTensors(bytes.NewReader(encoded.Bytes()))
		if err != nil {
			t.Fatalf("shape %v: %v", shape, err)
		}
		var roundtrip bytes.Buffer
		if err := SaveSafeTensors(&roundtrip, loaded); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded.Bytes(), roundtrip.Bytes()) {
			t.Fatalf("shape %v changed tensor shape, dtype or bytes", shape)
		}
	}
}

func TestSafeTensorsEmptyShapeRefusesPayload(t *testing.T) {
	header := []byte(`{"x":{"dtype":"F32","shape":[0],"data_offsets":[0,4]}}`)
	file := make([]byte, 8+len(header)+4)
	binary.LittleEndian.PutUint64(file, uint64(len(header)))
	copy(file[8:], header)
	if _, err := LoadSafeTensors(bytes.NewReader(file)); err == nil {
		t.Fatal("an empty tensor cannot carry a float32 payload")
	}
}

func TestSafeTensorsEmptyPythonReference(t *testing.T) {
	python := requireSafeTensorsReference(t)
	if python == "" {
		return
	}
	dir := t.TempDir()
	fixture := filepath.Join(dir, "python.safetensors")
	roundtrip := filepath.Join(dir, "go.safetensors")
	script := `import sys, torch
from safetensors.torch import save_file, load_file
if sys.argv[1] == "save":
    save_file({"f32": torch.empty((0, 2), dtype=torch.float32),
               "i64": torch.empty((2, 0), dtype=torch.int64),
               "bool": torch.empty((0,), dtype=torch.bool),
               "scalar": torch.tensor(1.25, dtype=torch.float32)}, sys.argv[2])
else:
    a, b = load_file(sys.argv[2]), load_file(sys.argv[3])
    assert a.keys() == b.keys()
    for k in a:
        assert a[k].dtype == b[k].dtype and a[k].shape == b[k].shape
        assert torch.equal(a[k], b[k])
`
	if out, err := exec.Command(python, "-c", script, "save", fixture).CombinedOutput(); err != nil {
		t.Fatalf("Python reference write: %v\n%s", err, out)
	}
	file, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	tensors, err := LoadSafeTensors(bytes.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := SaveSafeTensors(&encoded, tensors); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(roundtrip, encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(python, "-c", script, "compare", fixture, roundtrip).CombinedOutput(); err != nil {
		t.Fatalf("Python reference read: %v\n%s", err, out)
	}
}
