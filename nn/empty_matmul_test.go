package nn

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"testing"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/accel"
)

func emptyMatrixFile(t *testing.T, dtype string, a, b []int) map[string]*Tensor {
	t.Helper()
	type entry struct {
		Dtype   string `json:"dtype"`
		Shape   []int  `json:"shape"`
		Offsets [2]int `json:"data_offsets"`
	}
	header, err := json.Marshal(map[string]entry{"a": {dtype, a, [2]int{}}, "b": {dtype, b, [2]int{}}})
	if err != nil {
		t.Fatal(err)
	}
	file := make([]byte, 8+len(header))
	binary.LittleEndian.PutUint64(file, uint64(len(header)))
	copy(file[8:], header)
	tensors, err := LoadSafeTensors(bytes.NewReader(file))
	if err != nil {
		t.Fatal(err)
	}
	return tensors
}

func matrixSizeMustError(t *testing.T, call func() (*Tensor, error)) {
	t.Helper()
	defer func() {
		if value := recover(); value != nil {
			t.Errorf("matrix output size panicked instead of returning error: %v", value)
		}
	}()
	result, err := call()
	if err == nil || result != nil {
		t.Errorf("result=%v error=%v; want nil result and size error", result, err)
	}
}

func TestEmptyMatrixOutputSizeErrors(t *testing.T) {
	oldHook, oldEnabled := registeredDeviceMatMul(), insyra.Config.GetAccelerationEnabled()
	t.Cleanup(func() { RegisterDeviceMatMul(oldHook); insyra.Config.SetAcceleration(oldEnabled) })
	insyra.Config.SetAcceleration(true)
	calls := 0
	RegisterDeviceMatMul(func([]float32, int, int, []float32, int, int) ([]float32, error) {
		calls++
		return nil, errors.New("unexpected device dispatch")
	})
	for _, dtype := range []string{"F32", "F16", "BF16"} {
		t.Run(dtype, func(t *testing.T) {
			tensors := emptyMatrixFile(t, dtype, []int{maxInt(), 0}, []int{0, 2})
			a, b := tensors["a"], tensors["b"]
			tests := map[string]func() (*Tensor, error){
				"MatMul":               func() (*Tensor, error) { return MatMul(a, b) },
				"CPU one worker":       func() (*Tensor, error) { return matMul2DWithWorkers(a, b, 1) },
				"CPU multiple workers": func() (*Tensor, error) { return matMul2DWithWorkers(a, b, 4) },
				"Gemm":                 func() (*Tensor, error) { return Gemm(a, b, nil) },
				"rounded":              func() (*Tensor, error) { return matMulRounded(a, b, false, false) },
			}
			for name, call := range tests {
				t.Run(name, func(t *testing.T) { matrixSizeMustError(t, call) })
			}
			for _, precision := range []accel.Precision{accel.PrecisionExact, accel.PrecisionFloat32Rounded} {
				t.Run(string(precision), func(t *testing.T) {
					tape := NewTape()
					defer tape.Close()
					if err := tape.SetPrecision(precision); err != nil {
						t.Fatal(err)
					}
					matrixSizeMustError(t, func() (*Tensor, error) { return tape.MatMul(a, b) })
				})
			}
		})
	}
	if calls != 0 {
		t.Errorf("invalid empty matrix output dispatched to device %d times", calls)
	}
	for _, transpose := range []bool{false, true} {
		tensors := emptyMatrixFile(t, "F32", []int{0, maxInt()}, []int{2, 0})
		if !transpose {
			tensors = emptyMatrixFile(t, "F32", []int{0, maxInt()}, []int{0, 2})
		}
		opts := GemmOptions{Alpha: 1, Beta: 1, TransA: true, TransB: transpose}
		matrixSizeMustError(t, func() (*Tensor, error) { return Gemm(tensors["a"], tensors["b"], nil, opts) })
		matrixSizeMustError(t, func() (*Tensor, error) { return matMulRounded(tensors["a"], tensors["b"], true, transpose) })
	}
	if strconv.IntSize == 64 {
		t.Run("runtime allocation limit", func(t *testing.T) {
			tensors := emptyMatrixFile(t, "F32", []int{maxInt() / 8, 0}, []int{0, 1})
			for name, call := range map[string]func() (*Tensor, error){"MatMul": func() (*Tensor, error) { return MatMul(tensors["a"], tensors["b"]) }, "Gemm": func() (*Tensor, error) { return Gemm(tensors["a"], tensors["b"], nil) }, "rounded": func() (*Tensor, error) { return matMulRounded(tensors["a"], tensors["b"], false, false) }} {
				t.Run(name, func(t *testing.T) { matrixSizeMustError(t, call) })
			}
		})
	}
}

func TestEmptyMatMulHugeShapes(t *testing.T) {
	if kind := os.Getenv("INSYRA_EMPTY_MATRIX_CHILD"); kind != "" {
		old := registeredDeviceMatMul()
		defer RegisterDeviceMatMul(old)
		RegisterDeviceMatMul(func([]float32, int, int, []float32, int, int) ([]float32, error) {
			return nil, errors.New("empty work must not dispatch")
		})
		as, bs, want := []int{maxInt(), 0}, []int{0, 0}, []int{maxInt(), 0}
		if kind == "batch" {
			as, bs, want = []int{maxInt(), 0, 0}, []int{0, 0}, []int{maxInt(), 0, 0}
		}
		if kind == "vector" {
			as, bs, want = []int{maxInt(), 0, 0}, []int{0}, []int{maxInt(), 0}
		}
		tensors := emptyMatrixFile(t, "F32", as, bs)
		var result *Tensor
		var err error
		switch kind {
		case "rounded":
			result, err = matMulRounded(tensors["a"], tensors["b"], false, false)
		case "gemm":
			result, err = Gemm(tensors["a"], tensors["b"], nil)
		default:
			result, err = MatMul(tensors["a"], tensors["b"])
		}
		if err != nil {
			t.Fatal(err)
		}
		if !sameShape(result.shape, want) || result.Len() != 0 {
			t.Fatalf("shape=%v length=%d, want %v empty", result.shape, result.Len(), want)
		}
		return
	}
	for _, kind := range []string{"default", "rounded", "batch", "gemm", "vector"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestEmptyMatMulHugeShapes$")
			cmd.Env = append(os.Environ(), "INSYRA_EMPTY_MATRIX_CHILD="+kind)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("huge empty %s did not finish safely: %v (%v)\n%s", kind, err, ctx.Err(), output)
			}
		})
	}
}

func TestEmptyMatrixArithmeticAndShapes(t *testing.T) {
	shapes := []struct{ a, b, want []int }{{[]int{2, 0}, []int{0, 3}, []int{2, 3}}, {[]int{0}, []int{0}, nil}, {[]int{0}, []int{0, 3}, []int{3}}, {[]int{2, 0}, []int{0}, []int{2}}, {[]int{0, 2, 0}, []int{1, 0, 3}, []int{0, 2, 3}}}
	for _, test := range shapes {
		tensors := emptyMatrixFile(t, "F32", test.a, test.b)
		result, err := MatMul(tensors["a"], tensors["b"])
		if err != nil {
			t.Fatal(err)
		}
		if !sameShape(result.shape, test.want) {
			t.Fatalf("shape=%v, want %v", result.shape, test.want)
		}
		for _, v := range result.data {
			if math.Float32bits(v) != 0 {
				t.Fatalf("empty dot returned bits %x", math.Float32bits(v))
			}
		}
	}
	tensors := emptyMatrixFile(t, "F32", []int{2, 0}, []int{0, 3})
	for _, alpha := range []float32{1, -1, float32(math.Inf(1)), float32(math.NaN())} {
		result, err := Gemm(tensors["a"], tensors["b"], nil, GemmOptions{Alpha: alpha})
		if err != nil {
			t.Fatal(err)
		}
		want := alpha * float32(0)
		for _, v := range result.data {
			if math.IsNaN(float64(want)) {
				if !math.IsNaN(float64(v)) {
					t.Fatal("Gemm lost NaN")
				}
			} else if math.Float32bits(v) != math.Float32bits(want) {
				t.Fatalf("Gemm bits=%x, want%x", math.Float32bits(v), math.Float32bits(want))
			}
		}
	}
	bad, _ := NewTensor([]int{2, 2}, make([]float32, 4))
	if _, err := Gemm(tensors["a"], tensors["b"], bad); err == nil {
		t.Fatal("zero-inner Gemm accepted incompatible bias")
	}
	for _, ta := range []bool{false, true} {
		for _, tb := range []bool{false, true} {
			as, bs := []int{2, 0}, []int{0, 3}
			if ta {
				as = []int{0, 2}
			}
			if tb {
				bs = []int{3, 0}
			}
			inputs := emptyMatrixFile(t, "F32", as, bs)
			result, err := matMulRounded(inputs["a"], inputs["b"], ta, tb)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result.shape, []int{2, 3}) {
				t.Fatal(result.shape)
			}
			for _, v := range result.data {
				if math.Float32bits(v) != 0 {
					t.Fatal("rounded zero changed")
				}
			}
		}
	}
}

// The reported host overflow must also remain an ordinary error on the
// separate resident path; this control does not claim all device sizes fit.
func TestEmptyResidentMatMulSizeRefusal(t *testing.T) {
	device := residentTestDevice(t)
	baseline := accel.ResidentBytes(device)
	inputs := emptyMatrixFile(t, "F32", []int{maxInt(), 0}, []int{0, 2})
	a, err := inputs["a"].To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Release()
	b, err := inputs["b"].To(device)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Release()
	tape := NewTape()
	defer tape.Close()
	if err := tape.SetPrecision(accel.PrecisionFloat32Rounded); err != nil {
		t.Fatal(err)
	}
	matrixSizeMustError(t, func() (*Tensor, error) { return tape.MatMul(a, b) })
	tape.Close()
	a.Release()
	b.Release()
	if got := accel.ResidentBytes(device); got != baseline {
		t.Fatalf("resident rejection leaked: got %d bytes, want %d", got, baseline)
	}
}
