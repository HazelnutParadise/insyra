package nn

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/HazelnutParadise/insyra/accel"
)

const (
	optimizerStateMatrixName = "dense/β:weight"
	optimizerStateScalarName = "bias::偏置/λ"
	optimizerStateChildEnv   = "INSYRA_OPTIMIZER_STATE_CHILD"
	optimizerStateChildKind  = "INSYRA_OPTIMIZER_STATE_KIND"
	optimizerStateChildMode  = "INSYRA_OPTIMIZER_STATE_PRECISION"
)

var optimizerStateNames = []string{optimizerStateMatrixName, optimizerStateScalarName}

func optimizerStatePrefix(name string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(name)) + "/"
}

func optimizerStateKey(name, slot string) string {
	return optimizerStatePrefix(name) + slot
}

func newOptimizerStateTape(t *testing.T, precision accel.Precision, weights map[string]*Tensor) (*Tape, map[string]*Parameter) {
	t.Helper()
	tape := NewTape()
	t.Cleanup(tape.Close)
	if err := tape.SetPrecision(precision); err != nil {
		t.Fatal(err)
	}

	parameters := make(map[string]*Parameter, len(optimizerStateNames))
	for _, name := range optimizerStateNames {
		value := weights[name]
		if value == nil {
			shape := []int{2, 2}
			data := []float32{0.7, -1.2, 0.3, 0.9}
			if name == optimizerStateScalarName {
				shape, data = []int{}, []float32{0.42}
			}
			var err error
			value, err = NewTensor(shape, data)
			if err != nil {
				t.Fatal(err)
			}
		}
		parameter, err := tape.Param(value)
		if err != nil {
			t.Fatal(err)
		}
		parameters[name] = parameter
	}
	return tape, parameters
}

func optimizerStateForwardBackward(t *testing.T, tape *Tape, named map[string]*Parameter) {
	t.Helper()
	matrixSquared, err := tape.Mul(named[optimizerStateMatrixName].Value(), named[optimizerStateMatrixName].Value())
	if err != nil {
		t.Fatal(err)
	}
	matrixLoss, err := tape.ReduceMean(matrixSquared, []int{0, 1}, false)
	if err != nil {
		t.Fatal(err)
	}
	scalarSquared, err := tape.Mul(named[optimizerStateScalarName].Value(), named[optimizerStateScalarName].Value())
	if err != nil {
		t.Fatal(err)
	}
	loss, err := tape.Add(matrixLoss, scalarSquared)
	if err != nil {
		t.Fatal(err)
	}
	upstream, err := NewTensor(nil, []float32{1})
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.BackwardFrom(loss, upstream); err != nil {
		t.Fatal(err)
	}
}

func optimizerStateUpdate(t *testing.T, tape *Tape, named map[string]*Parameter, optimizer string) {
	t.Helper()
	tape.Clear()
	optimizerStateForwardBackward(t, tape, named)
	var err error
	switch optimizer {
	case "adam":
		err = tape.Adam(0.025)
	case "adamw":
		err = tape.AdamW(0.025, 0.13)
	case "momentum":
		err = tape.SGDMomentum(0.025, 0.81)
	default:
		t.Fatalf("unknown optimizer %q", optimizer)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func optimizerStateRunSteps(t *testing.T, tape *Tape, named map[string]*Parameter, optimizer string, steps int) {
	t.Helper()
	for range steps {
		optimizerStateUpdate(t, tape, named, optimizer)
	}
}

func optimizerStateWeightBytes(t *testing.T, named map[string]*Parameter) []byte {
	t.Helper()
	weights := make(map[string]*Tensor, len(named))
	for name, parameter := range named {
		host, err := parameter.Value().ToHost()
		if err != nil {
			t.Fatal(err)
		}
		weights[name] = host
	}
	var encoded bytes.Buffer
	if err := SaveSafeTensors(&encoded, weights); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func optimizerStateExport(t *testing.T, tape *Tape, named map[string]*Parameter) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := tape.SaveOptimizerState(&encoded, named); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func optimizerStateAssertExportedSlots(t *testing.T, encoded []byte, named map[string]*Parameter) {
	t.Helper()
	tensors, err := LoadSafeTensors(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	for name, parameter := range named {
		step := tensors[optimizerStateKey(name, "step")]
		if step == nil || step.dtype != DTypeInt64 || len(step.shape) != 0 || !reflect.DeepEqual(step.int64Data, []int64{int64(parameter.adamStep)}) {
			t.Fatalf("%q exported step = %v, want the scalar %d", name, step, parameter.adamStep)
		}
		for slot, expected := range map[string][]float32{
			"velocity": parameter.velocity,
			"adam_m":   parameter.adamM,
			"adam_v":   parameter.adamV,
		} {
			tensor := tensors[optimizerStateKey(name, slot)]
			if len(expected) == 0 {
				if tensor != nil {
					t.Fatalf("%q unexpectedly exported %s", name, slot)
				}
				continue
			}
			if tensor == nil || tensor.dtype != DTypeFloat32 {
				t.Fatalf("%q %s tensor missing or not F32", name, slot)
			}
			sameBitsSlice(t, name+" "+slot, tensor.data, expected)
		}
	}
}

func TestOptimizerStateSubprocessContinuation(t *testing.T) {
	for _, precision := range []accel.Precision{accel.PrecisionExact, accel.PrecisionFloat32Rounded} {
		for _, optimizer := range []string{"adam", "adamw", "momentum"} {
			t.Run(string(precision)+"/"+optimizer, func(t *testing.T) {
				fullTape, fullNamed := newOptimizerStateTape(t, precision, nil)
				optimizerStateRunSteps(t, fullTape, fullNamed, optimizer, 8)
				wantState := optimizerStateExport(t, fullTape, fullNamed)
				optimizerStateAssertExportedSlots(t, wantState, fullNamed)
				wantWeights := optimizerStateWeightBytes(t, fullNamed)

				splitTape, splitNamed := newOptimizerStateTape(t, precision, nil)
				optimizerStateRunSteps(t, splitTape, splitNamed, optimizer, 3)
				dir := t.TempDir()
				for path, data := range map[string][]byte{
					"optimizer-state": optimizerStateExport(t, splitTape, splitNamed),
					"weights":         optimizerStateWeightBytes(t, splitNamed),
				} {
					if err := os.WriteFile(filepath.Join(dir, path), data, 0600); err != nil {
						t.Fatal(err)
					}
				}

				executable, err := os.Executable()
				if err != nil {
					t.Fatal(err)
				}
				command := exec.Command(executable, "-test.run=^TestOptimizerStateSubprocessChild$")
				command.Env = append(os.Environ(),
					optimizerStateChildEnv+"="+dir,
					optimizerStateChildKind+"="+optimizer,
					optimizerStateChildMode+"="+string(precision),
				)
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("new process failed: %v\n%s", err, output)
				}

				for path, want := range map[string][]byte{
					"resumed-state":   wantState,
					"resumed-weights": wantWeights,
				} {
					got, err := os.ReadFile(filepath.Join(dir, path))
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(got, want) {
						t.Fatalf("new-process %s differs from uninterrupted run", path)
					}
				}
			})
		}
	}
}

func TestOptimizerStateSubprocessChild(t *testing.T) {
	dir := os.Getenv(optimizerStateChildEnv)
	if dir == "" {
		t.Skip("subprocess helper")
	}
	precision := accel.Precision(os.Getenv(optimizerStateChildMode))
	weightsFile, err := os.ReadFile(filepath.Join(dir, "weights"))
	if err != nil {
		t.Fatal(err)
	}
	weights, err := LoadSafeTensors(bytes.NewReader(weightsFile))
	if err != nil {
		t.Fatal(err)
	}
	tape, named := newOptimizerStateTape(t, precision, weights)
	state, err := os.ReadFile(filepath.Join(dir, "optimizer-state"))
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.LoadOptimizerState(bytes.NewReader(state), named); err != nil {
		t.Fatal(err)
	}
	optimizerStateRunSteps(t, tape, named, os.Getenv(optimizerStateChildKind), 5)
	optimizerStateAssertExportedSlots(t, optimizerStateExport(t, tape, named), named)
	for path, data := range map[string][]byte{
		"resumed-state":   optimizerStateExport(t, tape, named),
		"resumed-weights": optimizerStateWeightBytes(t, named),
	} {
		if err := os.WriteFile(filepath.Join(dir, path), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

type optimizerParameterSnapshot struct {
	valueRef     *Tensor
	valueShape   []int
	valueDType   DType
	value        []uint32
	gradRef      *Tensor
	gradShape    []int
	gradDType    DType
	grad         []uint32
	mapGradRef   *Tensor
	mapGradShape []int
	mapGradDType DType
	mapGrad      []uint32
	velocity     []uint32
	adamM        []uint32
	adamV        []uint32
	adamStep     uint64
}

func optimizerStateBits(values []float32) []uint32 {
	if values == nil {
		return nil
	}
	bits := make([]uint32, len(values))
	for index, value := range values {
		bits[index] = math.Float32bits(value)
	}
	return bits
}

func optimizerStateSnapshot(named map[string]*Parameter, tape *Tape) map[string]optimizerParameterSnapshot {
	snapshot := make(map[string]optimizerParameterSnapshot, len(named))
	for name, parameter := range named {
		var grad, mapGrad []uint32
		var gradShape, mapGradShape []int
		var gradDType, mapGradDType DType
		var mapGradient *Tensor
		if parameter.grad != nil {
			grad = optimizerStateBits(parameter.grad.data)
			gradShape = append([]int(nil), parameter.grad.shape...)
			gradDType = parameter.grad.dtype
		}
		if mapGradient = tape.grads[parameter.value]; mapGradient != nil {
			mapGrad = optimizerStateBits(mapGradient.data)
			mapGradShape = append([]int(nil), mapGradient.shape...)
			mapGradDType = mapGradient.dtype
		}
		snapshot[name] = optimizerParameterSnapshot{
			valueRef:     parameter.value,
			valueShape:   append([]int(nil), parameter.value.shape...),
			valueDType:   parameter.value.dtype,
			value:        optimizerStateBits(parameter.value.data),
			gradRef:      parameter.grad,
			gradShape:    gradShape,
			gradDType:    gradDType,
			grad:         grad,
			mapGradRef:   mapGradient,
			mapGradShape: mapGradShape,
			mapGradDType: mapGradDType,
			mapGrad:      mapGrad,
			velocity:     optimizerStateBits(parameter.velocity),
			adamM:        optimizerStateBits(parameter.adamM),
			adamV:        optimizerStateBits(parameter.adamV),
			adamStep:     parameter.adamStep,
		}
	}
	return snapshot
}

func optimizerStateRequireAtomicRejection(t *testing.T, tape *Tape, named map[string]*Parameter, encoded []byte) {
	t.Helper()
	before := optimizerStateSnapshot(named, tape)
	err := tape.LoadOptimizerState(bytes.NewReader(encoded), named)
	if err == nil {
		t.Fatal("invalid optimizer state was accepted")
	}
	after := optimizerStateSnapshot(named, tape)
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("rejected import partially mutated parameters:\nbefore: %#v\nafter:  %#v", before, after)
	}
}

// optimizerStateMutateFile re-encodes a checkpoint after editing its tensors and
// its header metadata, which carries the version and the precision.
func optimizerStateMutateFile(t *testing.T, encoded []byte, mutate func(map[string]*Tensor), mutateMetadata func(map[string]string)) []byte {
	t.Helper()
	tensors, err := LoadSafeTensors(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("decode valid SafeTensors fixture: %v", err)
	}
	headerEnd := 8 + int(binary.LittleEndian.Uint64(encoded[:8]))
	_, metadata, err := parseSafeTensorHeader(encoded[8:headerEnd])
	if err != nil {
		t.Fatal(err)
	}
	if mutate != nil {
		mutate(tensors)
	}
	if mutateMetadata != nil {
		mutateMetadata(metadata)
	}
	var result bytes.Buffer
	if err := saveSafeTensors(&result, tensors, metadata); err != nil {
		t.Fatalf("re-encode SafeTensors mutation: %v", err)
	}
	return result.Bytes()
}

func optimizerStateReplaceI64(t *testing.T, tensors map[string]*Tensor, name string, values []int64) {
	t.Helper()
	shape := []int{len(values)}
	if existing := tensors[name]; existing != nil {
		shape = append([]int(nil), existing.shape...)
	}
	replacement, err := NewInt64Tensor(shape, values)
	if err != nil {
		t.Fatal(err)
	}
	tensors[name] = replacement
}

func optimizerStateReplaceF32(t *testing.T, tensors map[string]*Tensor, name string, values []float32) {
	t.Helper()
	shape := []int{len(values)}
	if existing := tensors[name]; existing != nil {
		shape = append([]int(nil), existing.shape...)
	}
	replacement, err := NewTensor(shape, values)
	if err != nil {
		t.Fatal(err)
	}
	tensors[name] = replacement
}

func TestOptimizerStateCorruptImportsAreTransactional(t *testing.T) {
	source, sourceNamed := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, source, sourceNamed, "adam", 4)
	adamState := optimizerStateExport(t, source, sourceNamed)
	momentumTape, momentumNamed := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, momentumTape, momentumNamed, "momentum", 4)
	momentumState := optimizerStateExport(t, momentumTape, momentumNamed)

	scalarPrefix := optimizerStatePrefix(optimizerStateScalarName)
	matrixMoments := optimizerStateKey(optimizerStateMatrixName, "adam_m")
	matrixSecondMoment := optimizerStateKey(optimizerStateMatrixName, "adam_v")
	matrixStep := optimizerStateKey(optimizerStateMatrixName, "step")
	matrixShape := optimizerStateKey(optimizerStateMatrixName, "shape")
	cases := []struct {
		name  string
		input []byte
		edit  func(map[string]*Tensor)
		meta  func(map[string]string)
		wire  string
	}{
		{
			name:  "unsupported version",
			input: adamState,
			meta: func(metadata map[string]string) {
				metadata[optimizerStateVersionKey] = "3"
			},
		},
		{
			name:  "missing version",
			input: adamState,
			meta: func(metadata map[string]string) {
				delete(metadata, optimizerStateVersionKey)
			},
		},
		{
			name:  "missing parameter name",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				for name := range tensors {
					if bytes.HasPrefix([]byte(name), []byte(scalarPrefix)) {
						delete(tensors, name)
					}
				}
			},
		},
		{
			name:  "extra parameter name",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				extra := optimizerStatePrefix("unexpected/name")
				optimizerStateReplaceI64(t, tensors, extra+"shape", []int64{1})
				step, _ := NewInt64Tensor(nil, []int64{0})
				tensors[extra+"step"] = step
			},
		},
		{
			name:  "shape mismatch",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				optimizerStateReplaceI64(t, tensors, matrixShape, []int64{2, 3})
			},
		},
		{
			name:  "step uses wrong wire dtype",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				step, _ := NewTensor(nil, []float32{4})
				tensors[matrixStep] = step
			},
		},
		{
			name:  "step is not a scalar",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				step, _ := NewInt64Tensor([]int{2}, []int64{4, 0})
				tensors[matrixStep] = step
			},
		},
		{
			name: "moment shape mismatch with same element count", input: adamState,
			edit: func(tensors map[string]*Tensor) {
				x := tensors[matrixMoments]
				tensors[matrixMoments], _ = NewTensor([]int{4}, x.data)
			},
		},
		{
			name: "zero step with initialized moments", input: adamState,
			edit: func(tensors map[string]*Tensor) {
				optimizerStateReplaceI64(t, tensors, matrixStep, []int64{0})
			},
		},
		{
			name: "positive step without moments", input: adamState,
			edit: func(tensors map[string]*Tensor) {
				delete(tensors, matrixMoments)
				delete(tensors, matrixSecondMoment)
			},
		},
		{
			name: "unknown optimizer slot", input: adamState,
			edit: func(tensors map[string]*Tensor) {
				tensors[optimizerStateKey(optimizerStateMatrixName, "unknown")] = tensors[matrixMoments]
			},
		},
		{
			name:  "precision mismatch",
			input: adamState,
			meta: func(metadata map[string]string) {
				metadata[optimizerStatePrecisionKey] = "float32"
			},
		},
		{
			name:  "non-finite first moment",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				optimizerStateReplaceF32(t, tensors, matrixMoments, []float32{float32(math.NaN()), 0, 0, 0})
			},
		},
		{
			name:  "non-finite second moment",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				optimizerStateReplaceF32(t, tensors, matrixSecondMoment, []float32{0, 0, float32(math.Inf(1)), 0})
			},
		},
		{
			name:  "negative second moment",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				optimizerStateReplaceF32(t, tensors, matrixSecondMoment, []float32{0, -math.SmallestNonzeroFloat32, 0, 0})
			},
		},
		{
			name:  "negative step",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				optimizerStateReplaceI64(t, tensors, matrixStep, []int64{-1})
			},
		},
		{
			name:  "missing first moment",
			input: adamState,
			edit: func(tensors map[string]*Tensor) {
				delete(tensors, matrixMoments)
			},
		},
		{
			name:  "non-finite velocity",
			input: momentumState,
			edit: func(tensors map[string]*Tensor) {
				optimizerStateReplaceF32(t, tensors, optimizerStateKey(optimizerStateMatrixName, "velocity"), []float32{0, 0, float32(math.NaN()), 0})
			},
		},
		{
			name:  "F16 moment payload",
			input: adamState,
			wire:  matrixMoments + "\x00F16",
		},
		{
			name:  "BF16 moment payload",
			input: adamState,
			wire:  matrixMoments + "\x00BF16",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			encoded := testCase.input
			if testCase.wire != "" {
				separator := bytes.IndexByte([]byte(testCase.wire), 0)
				wireName := testCase.wire[:separator]
				wireType := testCase.wire[separator+1:]
				encoded = optimizerStateRewriteWireDType(t, encoded, wireName, wireType)
			} else {
				encoded = optimizerStateMutateFile(t, encoded, testCase.edit, testCase.meta)
			}
			if _, err := LoadSafeTensors(bytes.NewReader(encoded)); err != nil {
				t.Fatalf("mutated fixture is not valid SafeTensors: %v", err)
			}

			target, named := newOptimizerStateTape(t, accel.PrecisionExact, nil)
			optimizerStateRunSteps(t, target, named, "adam", 2)
			optimizerStateRunSteps(t, target, named, "momentum", 1)
			optimizerStateRequireAtomicRejection(t, target, named, encoded)
		})
	}
}

func TestOptimizerStateMixedHistoryAndReorderedParameters(t *testing.T) {
	source, named := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, source, named, "adamw", 3)
	optimizerStateRunSteps(t, source, named, "momentum", 2)
	state := optimizerStateExport(t, source, named)
	weights, err := LoadSafeTensors(bytes.NewReader(optimizerStateWeightBytes(t, named)))
	if err != nil {
		t.Fatal(err)
	}
	target := NewTape()
	defer target.Close()
	reordered := make(map[string]*Parameter)
	for i := len(optimizerStateNames) - 1; i >= 0; i-- {
		name := optimizerStateNames[i]
		reordered[name], err = target.Param(weights[name])
		if err != nil {
			t.Fatal(err)
		}
		reordered[name].SetFrozen(true)
	}
	if err := target.LoadOptimizerState(bytes.NewReader(state), reordered); err != nil {
		t.Fatal(err)
	}
	for _, p := range reordered {
		if !p.GetFrozen() {
			t.Fatal("import changed caller freeze policy")
		}
		p.SetFrozen(false)
	}
	optimizerStateAssertExportedSlots(t, optimizerStateExport(t, target, reordered), reordered)
	for _, optimizer := range []string{"adam", "momentum", "adamw"} {
		optimizerStateUpdate(t, source, named, optimizer)
		optimizerStateUpdate(t, target, reordered, optimizer)
		if !bytes.Equal(optimizerStateWeightBytes(t, named), optimizerStateWeightBytes(t, reordered)) ||
			!bytes.Equal(optimizerStateExport(t, source, named), optimizerStateExport(t, target, reordered)) {
			t.Fatalf("mixed %s state resumed by position instead of name", optimizer)
		}
	}
}

func TestOptimizerStateMalformedFileIsAtomic(t *testing.T) {
	tape, named := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, tape, named, "adam", 2)
	valid := optimizerStateExport(t, tape, named)
	for _, input := range [][]byte{nil, {1, 2, 3}, bytes.Repeat([]byte{255}, 8), valid[:len(valid)-1]} {
		optimizerStateRequireAtomicRejection(t, tape, named, input)
	}
}

func TestOptimizerStateUninitializedImportClearsOldSlots(t *testing.T) {
	source, sourceNamed := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	uninitialized := optimizerStateExport(t, source, sourceNamed)

	target, targetNamed := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, target, targetNamed, "adam", 2)
	optimizerStateRunSteps(t, target, targetNamed, "momentum", 1)
	before := optimizerStateSnapshot(targetNamed, target)
	if err := target.LoadOptimizerState(bytes.NewReader(uninitialized), targetNamed); err != nil {
		t.Fatal(err)
	}
	for name, parameter := range targetNamed {
		old := before[name]
		after := optimizerStateSnapshot(map[string]*Parameter{name: parameter}, target)[name]
		if after.valueRef != old.valueRef ||
			!reflect.DeepEqual(after.valueShape, old.valueShape) ||
			after.valueDType != old.valueDType ||
			!reflect.DeepEqual(after.value, old.value) ||
			after.gradRef != old.gradRef ||
			!reflect.DeepEqual(after.gradShape, old.gradShape) ||
			after.gradDType != old.gradDType ||
			!reflect.DeepEqual(after.grad, before[name].grad) ||
			after.mapGradRef != old.mapGradRef ||
			!reflect.DeepEqual(after.mapGradShape, old.mapGradShape) ||
			after.mapGradDType != old.mapGradDType ||
			!reflect.DeepEqual(after.mapGrad, old.mapGrad) {
			t.Fatalf("loading uninitialized state changed %q values or gradients", name)
		}
		if len(parameter.velocity) != 0 || len(parameter.adamM) != 0 || len(parameter.adamV) != 0 || parameter.adamStep != 0 {
			t.Fatalf("loading uninitialized state did not clear %q optimizer slots", name)
		}
	}
}

func TestOptimizerStateEmptyTapeRoundTrip(t *testing.T) {
	tape := NewTape()
	defer tape.Close()
	named := map[string]*Parameter{}
	var encoded bytes.Buffer
	if err := tape.SaveOptimizerState(&encoded, named); err != nil {
		t.Fatal(err)
	}
	if err := tape.LoadOptimizerState(bytes.NewReader(encoded.Bytes()), named); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSafeTensors(bytes.NewReader(encoded.Bytes())); err != nil {
		t.Fatalf("empty-tape checkpoint is not a valid SafeTensors file: %v", err)
	}
}

func TestOptimizerStatePreservesBinaryFloatEdgesAndUint64Step(t *testing.T) {
	values := map[string]*Tensor{}
	matrixBits := []uint32{0x80000000, 0x00000001, 0x80000001, 0x3f800000}
	matrixValues := make([]float32, len(matrixBits))
	for index, bits := range matrixBits {
		matrixValues[index] = math.Float32frombits(bits)
	}
	matrix, err := NewTensor([]int{2, 2}, matrixValues)
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := NewTensor([]int{}, []float32{math.Float32frombits(0x80000000)})
	if err != nil {
		t.Fatal(err)
	}
	values[optimizerStateMatrixName], values[optimizerStateScalarName] = matrix, scalar
	source, sourceNamed := newOptimizerStateTape(t, accel.PrecisionExact, values)
	for _, name := range optimizerStateNames {
		parameter := sourceNamed[name]
		count := len(parameter.value.data)
		parameter.velocity = make([]float32, count)
		parameter.adamM = make([]float32, count)
		parameter.adamV = make([]float32, count)
		for index := range parameter.velocity {
			parameter.velocity[index] = math.Float32frombits(0x80000000)
			parameter.adamM[index] = math.Float32frombits(0x00000001)
			parameter.adamV[index] = math.Float32frombits(0x00000002)
		}
		parameter.adamStep = (uint64(1) << 53) + 12345
	}
	state := optimizerStateExport(t, source, sourceNamed)
	weightBytes := optimizerStateWeightBytes(t, sourceNamed)
	loadedWeights, err := LoadSafeTensors(bytes.NewReader(weightBytes))
	if err != nil {
		t.Fatal(err)
	}
	target, targetNamed := newOptimizerStateTape(t, accel.PrecisionExact, loadedWeights)
	if err := target.LoadOptimizerState(bytes.NewReader(state), targetNamed); err != nil {
		t.Fatal(err)
	}

	for _, name := range optimizerStateNames {
		sourceParameter, targetParameter := sourceNamed[name], targetNamed[name]
		sameBitsSlice(t, name+" value", targetParameter.value.data, sourceParameter.value.data)
		sameBitsSlice(t, name+" velocity", targetParameter.velocity, sourceParameter.velocity)
		sameBitsSlice(t, name+" first moment", targetParameter.adamM, sourceParameter.adamM)
		sameBitsSlice(t, name+" second moment", targetParameter.adamV, sourceParameter.adamV)
		if targetParameter.adamStep != sourceParameter.adamStep {
			t.Fatalf("%q step = %d, want exact uint64 %d", name, targetParameter.adamStep, sourceParameter.adamStep)
		}
	}
	if targetNamed[optimizerStateMatrixName].value.data[0] != 0 ||
		math.Float32bits(targetNamed[optimizerStateMatrixName].value.data[0]) != 0x80000000 ||
		math.Float32bits(targetNamed[optimizerStateMatrixName].value.data[1]) != 0x00000001 {
		t.Fatal("SafeTensors weights did not preserve signed zero and the smallest subnormal")
	}
}

func TestOptimizerStateRejectsInvalidMappingsAndClosedEndpoints(t *testing.T) {
	tape, named := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	optimizerStateRunSteps(t, tape, named, "adam", 2)
	validState := optimizerStateExport(t, tape, named)

	_, foreignNamed := newOptimizerStateTape(t, accel.PrecisionExact, nil)
	mappings := []struct {
		name  string
		named map[string]*Parameter
	}{
		{name: "nil map", named: nil},
		{name: "empty map", named: map[string]*Parameter{}},
		{name: "missing parameter", named: map[string]*Parameter{optimizerStateMatrixName: named[optimizerStateMatrixName]}},
		{name: "extra name", named: map[string]*Parameter{
			optimizerStateMatrixName: named[optimizerStateMatrixName],
			optimizerStateScalarName: named[optimizerStateScalarName],
			"extra":                  named[optimizerStateScalarName],
		}},
		{name: "empty name", named: map[string]*Parameter{
			optimizerStateMatrixName: named[optimizerStateMatrixName],
			"":                       named[optimizerStateScalarName],
		}},
		{name: "aliased parameter", named: map[string]*Parameter{
			optimizerStateMatrixName: named[optimizerStateMatrixName],
			optimizerStateScalarName: named[optimizerStateMatrixName],
		}},
		{name: "foreign parameter", named: map[string]*Parameter{
			optimizerStateMatrixName: foreignNamed[optimizerStateMatrixName],
			optimizerStateScalarName: named[optimizerStateScalarName],
		}},
	}
	for _, mapping := range mappings {
		t.Run(mapping.name, func(t *testing.T) {
			before := optimizerStateSnapshot(named, tape)
			var encoded bytes.Buffer
			if err := tape.SaveOptimizerState(&encoded, mapping.named); err == nil {
				t.Fatal("SaveOptimizerState accepted an invalid mapping")
			}
			if err := tape.LoadOptimizerState(bytes.NewReader(validState), mapping.named); err == nil {
				t.Fatal("LoadOptimizerState accepted an invalid mapping")
			}
			if after := optimizerStateSnapshot(named, tape); !reflect.DeepEqual(after, before) {
				t.Fatal("invalid mapping changed parameter values, gradients or optimizer state")
			}
		})
	}

	var nilTape *Tape
	if err := nilTape.SaveOptimizerState(&bytes.Buffer{}, named); err == nil {
		t.Fatal("nil tape saved optimizer state")
	}
	if err := nilTape.LoadOptimizerState(bytes.NewReader(validState), named); err == nil {
		t.Fatal("nil tape loaded optimizer state")
	}

	closedTape := NewTape()
	closedTape.Close()
	if err := closedTape.SaveOptimizerState(&bytes.Buffer{}, map[string]*Parameter{}); err == nil {
		t.Fatal("closed tape saved optimizer state")
	}
	if err := closedTape.LoadOptimizerState(bytes.NewReader(validState), named); err == nil {
		t.Fatal("closed tape loaded optimizer state")
	}

	if err := tape.SaveOptimizerState(nil, named); err == nil {
		t.Fatal("nil writer was accepted")
	}
	if err := tape.LoadOptimizerState(nil, named); err == nil {
		t.Fatal("nil reader was accepted")
	}
	var typedNilWriter *optimizerStateErrorWriter
	if err := tape.SaveOptimizerState(typedNilWriter, named); err == nil {
		t.Fatal("typed nil writer was accepted")
	}
	var typedNilReader *optimizerStateErrorReader
	if err := tape.LoadOptimizerState(typedNilReader, named); err == nil {
		t.Fatal("typed nil reader was accepted")
	}
	var nilBuffer *bytes.Buffer
	if err := tape.SaveOptimizerState(nilBuffer, named); err == nil {
		t.Fatal("typed nil bytes.Buffer writer was accepted")
	}
	var nilBytesReader *bytes.Reader
	if err := tape.LoadOptimizerState(nilBytesReader, named); err == nil {
		t.Fatal("typed nil bytes.Reader was accepted")
	}

	reader, readerWriter := io.Pipe()
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := readerWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tape.LoadOptimizerState(reader, named); err == nil {
		t.Fatal("closed reader was accepted")
	}
	if err := tape.LoadOptimizerState(&optimizerStateErrorReader{}, named); err == nil {
		t.Fatal("reader error was ignored")
	}
	if err := tape.SaveOptimizerState(&optimizerStateErrorWriter{}, named); err == nil {
		t.Fatal("writer error was ignored")
	}
	if err := tape.SaveOptimizerState(optimizerStateShortWriter{}, named); err == nil {
		t.Fatal("short write was ignored")
	}
	_, closedWriter := io.Pipe()
	if err := closedWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tape.SaveOptimizerState(closedWriter, named); err == nil {
		t.Fatal("closed writer was accepted")
	}
}

type optimizerStateErrorReader struct{}

func (*optimizerStateErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("closed optimizer-state reader")
}

type optimizerStateErrorWriter struct{}

func (*optimizerStateErrorWriter) Write([]byte) (int, error) {
	return 0, errors.New("closed optimizer-state writer")
}

type optimizerStateShortWriter struct{}

func (optimizerStateShortWriter) Write([]byte) (int, error) {
	return 0, nil
}

type optimizerStateWireHeader struct {
	DType       string    `json:"dtype"`
	Shape       []int     `json:"shape"`
	DataOffsets [2]uint64 `json:"data_offsets"`
}

func optimizerStateRewriteWireDType(t *testing.T, encoded []byte, name, dtype string) []byte {
	t.Helper()
	if len(encoded) < 8 {
		t.Fatal("state fixture is truncated")
	}
	headerLength := binary.LittleEndian.Uint64(encoded[:8])
	if headerLength > uint64(len(encoded)-8) {
		t.Fatal("state fixture has an invalid header length")
	}
	headerEnd := 8 + int(headerLength)
	entries, _, err := parseSafeTensorHeader(encoded[8:headerEnd])
	if err != nil {
		t.Fatal(err)
	}
	data := encoded[headerEnd:]
	payloads := make(map[string][]byte, len(entries))
	headers := make(map[string]optimizerStateWireHeader, len(entries))
	for _, entry := range entries {
		payload := append([]byte(nil), data[int(entry.begin):int(entry.end)]...)
		entryDType := entry.dtype
		if entry.name == name {
			if dtype != "F16" && dtype != "BF16" {
				t.Fatalf("unsupported test dtype %q", dtype)
			}
			entryDType = dtype
			payload = make([]byte, int(entry.count)*2)
		}
		payloads[entry.name] = payload
		headers[entry.name] = optimizerStateWireHeader{DType: entryDType, Shape: append([]int{}, entry.shape...)}
	}
	if _, ok := payloads[name]; !ok {
		t.Fatalf("wire fixture has no tensor %q", name)
	}

	names := make([]string, 0, len(payloads))
	for tensorName := range payloads {
		names = append(names, tensorName)
	}
	sort.Strings(names)
	var body bytes.Buffer
	for _, tensorName := range names {
		entry := headers[tensorName]
		entry.DataOffsets = [2]uint64{uint64(body.Len()), uint64(body.Len() + len(payloads[tensorName]))}
		headers[tensorName] = entry
		body.Write(payloads[tensorName])
	}
	header, err := json.Marshal(headers)
	if err != nil {
		t.Fatal(err)
	}
	result := make([]byte, 8+len(header)+body.Len())
	binary.LittleEndian.PutUint64(result[:8], uint64(len(header)))
	copy(result[8:], header)
	copy(result[8+len(header):], body.Bytes())
	return result
}

func TestOptimizerStateHalfWireFixtureIsValid(t *testing.T) {
	version, _ := NewInt64Tensor(nil, []int64{1})
	moment, _ := NewTensor([]int{2}, []float32{0, 0})
	var encoded bytes.Buffer
	if err := SaveSafeTensors(&encoded, map[string]*Tensor{"version": version, "moment": moment}); err != nil {
		t.Fatal(err)
	}
	for _, dtype := range []string{"F16", "BF16"} {
		wire := optimizerStateRewriteWireDType(t, encoded.Bytes(), "moment", dtype)
		if _, err := LoadSafeTensors(bytes.NewReader(wire)); err != nil {
			t.Fatalf("%s fixture must be valid SafeTensors before testing optimizer dtype refusal: %v", dtype, err)
		}
	}
}

func TestOptimizerStateEmptyParameterRetainsInitializedAdam(t *testing.T) {
	source := NewTape()
	defer source.Close()
	empty, _ := NewTensor([]int{0, 2}, nil)
	p, err := source.Param(empty)
	if err != nil {
		t.Fatal(err)
	}
	y, err := source.Add(empty, empty)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.BackwardFrom(y, empty); err != nil {
		t.Fatal(err)
	}
	if err := source.Adam(0.01); err != nil {
		t.Fatal(err)
	}
	named := map[string]*Parameter{"empty": p}
	state := optimizerStateExport(t, source, named)
	target := NewTape()
	defer target.Close()
	q, err := target.Param(empty)
	if err != nil {
		t.Fatal(err)
	}
	restored := map[string]*Parameter{"empty": q}
	if err := target.LoadOptimizerState(bytes.NewReader(state), restored); err != nil {
		t.Fatal(err)
	}
	if q.adamStep != 1 || !bytes.Equal(state, optimizerStateExport(t, target, restored)) {
		t.Fatal("zero-sized parameter lost initialized Adam slots")
	}
}
