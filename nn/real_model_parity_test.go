package nn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const realModelParityScript = `
import json
import sys

import numpy as np
import onnxruntime as ort

model_path, feed_path = sys.argv[1], sys.argv[2]
with open(feed_path) as handle:
    payload = json.load(handle)
feed = {}
for item in payload:
    dtype = {"float32": np.float32, "int64": np.int64, "bool": np.bool_}[item["dtype"]]
    feed[item["name"]] = np.asarray(item["data"], dtype=dtype).reshape(item["shape"])
session = ort.InferenceSession(model_path, providers=["CPUExecutionProvider"])
result = []
for value in session.run(None, feed):
    array = np.asarray(value)
    if array.dtype == np.bool_:
        dtype = "bool"
    elif array.dtype == np.int64:
        dtype = "int64"
    else:
        dtype = "float32"
    result.append({"shape": list(array.shape), "dtype": dtype, "data": array.reshape(-1).tolist()})
print(json.dumps(result))
`

func TestRealModelParity(t *testing.T) {
	modelDir := os.Getenv("INSYRA_NN_REAL_MODELS_DIR")
	if modelDir == "" {
		t.Skip("INSYRA_NN_REAL_MODELS_DIR is not set")
	}
	models := []struct {
		name string
		file string
		feed func(*testing.T, *Model) map[string]*Tensor
	}{
		{name: "MobileNetV2", file: "mobilenetv2-12.onnx", feed: mobileNetFeed},
		{name: "MiniLM-L6-v2", file: "minilm-l6-v2.onnx", feed: miniLMFeed},
	}
	for _, tc := range models {
		if _, err := os.Stat(filepath.Join(modelDir, tc.file)); err != nil {
			t.Skipf("INSYRA_NN_REAL_MODELS_DIR is missing %s: %v", tc.file, err)
		}
	}
	python := requireONNXReference(t)
	if python == "" {
		return
	}
	for _, tc := range models {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(modelDir, tc.file)
			modelBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			model, err := LoadONNX(bytes.NewReader(modelBytes))
			if err != nil {
				t.Fatalf("load %s: %v", path, err)
			}
			inputs := tc.feed(t, model)
			feedPath := filepath.Join(t.TempDir(), "feed.json")
			if err := writeRealModelFeed(feedPath, model.Inputs(), inputs); err != nil {
				t.Fatalf("write %s feed: %v", tc.name, err)
			}
			outputs, err := model.Run(inputs)
			if err != nil {
				t.Fatalf("run %s in nn: %v", tc.name, err)
			}
			reference := runRealModelPython(t, python, path, feedPath)
			if len(reference) != len(model.Outputs()) {
				t.Fatalf("reference returned %d outputs, want %d", len(reference), len(model.Outputs()))
			}
			for index, spec := range model.Outputs() {
				output, present := outputs[spec.Name]
				if !present {
					t.Fatalf("nn output %q is missing", spec.Name)
				}
				assertParityOutput(t, output, reference[index])
			}
		})
	}
}

func mobileNetFeed(t *testing.T, model *Model) map[string]*Tensor {
	t.Helper()
	if len(model.Inputs()) != 1 || model.Inputs()[0].DType != DTypeFloat32 {
		t.Fatalf("MobileNet input contract = %+v", model.Inputs())
	}
	data := make([]float32, 3*224*224)
	for index := range data {
		data[index] = float32((index%251)-125) / 125
	}
	return map[string]*Tensor{model.Inputs()[0].Name: mustTestTensor(t, []int{1, 3, 224, 224}, data)}
}

func miniLMFeed(t *testing.T, model *Model) map[string]*Tensor {
	t.Helper()
	inputs := model.Inputs()
	if len(inputs) != 3 {
		t.Fatalf("MiniLM input contract = %+v", inputs)
	}
	tokens := make([]int64, 2*16)
	mask := make([]int64, len(tokens))
	types := make([]int64, len(tokens))
	for index := range tokens {
		tokens[index] = int64((index*7919 + 17) % 30522)
		mask[index] = 1
	}
	mask[3], mask[10], mask[15] = 0, 0, 0
	return map[string]*Tensor{
		"input_ids":      mustTestInt64Tensor(t, []int{2, 16}, tokens),
		"attention_mask": mustTestInt64Tensor(t, []int{2, 16}, mask),
		"token_type_ids": mustTestInt64Tensor(t, []int{2, 16}, types),
	}
}

func writeRealModelFeed(path string, specs []ValueInfo, inputs map[string]*Tensor) error {
	payload := make([]parityInput, 0, len(specs))
	for _, spec := range specs {
		input, present := inputs[spec.Name]
		if !present {
			return fmt.Errorf("input %q is missing", spec.Name)
		}
		var data any
		switch input.dtype {
		case DTypeFloat32:
			data = input.data
		case DTypeInt64:
			data = input.int64Data
		case DTypeBool:
			data = input.boolData
		default:
			return fmt.Errorf("input %q has unsupported dtype %s", spec.Name, input.dtype)
		}
		encoded, err := json.Marshal(data)
		if err != nil {
			return err
		}
		payload = append(payload, parityInput{Name: spec.Name, Shape: input.shape, DType: string(input.dtype), Data: encoded})
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(path, encoded, 0o600)
}

func runRealModelPython(t *testing.T, python, modelPath, feedPath string) []parityOutput {
	t.Helper()
	command := exec.Command(python, "-c", realModelParityScript, modelPath, feedPath)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("onnxruntime real-model helper: %v: %s", err, stderr.String())
	}
	var outputs []parityOutput
	if err := json.Unmarshal(stdout.Bytes(), &outputs); err != nil {
		t.Fatalf("decode real-model helper stdout: %v\nstdout=%s\nstderr=%s", err, stdout.String(), stderr.String())
	}
	return outputs
}
