package nn

import (
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"reflect"
	"sort"
	"unicode/utf8"

	"github.com/HazelnutParadise/insyra/accel"
)

type optimizerStateTarget struct {
	name      string
	parameter *Parameter
}

type optimizerStateSlots struct {
	velocity []float32
	adamM    []float32
	adamV    []float32
	step     uint64
	hasAdam  bool
}

func nilOptimizerStateEndpoint(endpoint any) bool {
	if endpoint == nil {
		return true
	}
	value := reflect.ValueOf(endpoint)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func optimizerStateTargets(t *Tape, named map[string]*Parameter) ([]optimizerStateTarget, error) {
	if t == nil {
		return nil, fmt.Errorf("optimizer state: tape is nil")
	}
	if err := t.checkOpen(); err != nil {
		return nil, fmt.Errorf("optimizer state: %w", err)
	}
	if named == nil {
		return nil, fmt.Errorf("optimizer state: parameter mapping is nil")
	}
	if len(named) != len(t.params) {
		return nil, fmt.Errorf("optimizer state: mapping has %d parameters, tape has %d", len(named), len(t.params))
	}

	tapeParameters := make(map[*Parameter]struct{}, len(t.params))
	for _, parameter := range t.params {
		if parameter == nil {
			return nil, fmt.Errorf("optimizer state: tape contains a nil parameter")
		}
		if _, exists := tapeParameters[parameter]; exists {
			return nil, fmt.Errorf("optimizer state: tape contains an aliased parameter")
		}
		tapeParameters[parameter] = struct{}{}
	}

	targets := make([]optimizerStateTarget, 0, len(named))
	seenParameters := make(map[*Parameter]string, len(named))
	seenValues := make(map[*Tensor]string, len(named))
	for name, parameter := range named {
		if name == "" || !utf8.ValidString(name) {
			return nil, fmt.Errorf("optimizer state: parameter names must be non-empty valid UTF-8")
		}
		if parameter == nil {
			return nil, fmt.Errorf("optimizer state: parameter %q is nil", name)
		}
		if previous, exists := seenParameters[parameter]; exists {
			return nil, fmt.Errorf("optimizer state: names %q and %q alias the same parameter", previous, name)
		}
		seenParameters[parameter] = name
		if _, exists := tapeParameters[parameter]; !exists {
			return nil, fmt.Errorf("optimizer state: parameter %q does not belong to this tape", name)
		}
		if parameter.value == nil {
			return nil, fmt.Errorf("optimizer state: parameter %q has no value tensor", name)
		}
		if previous, exists := seenValues[parameter.value]; exists {
			return nil, fmt.Errorf("optimizer state: parameters %q and %q alias the same value tensor", previous, name)
		}
		seenValues[parameter.value] = name
		if t.marked[parameter.value] != parameter {
			return nil, fmt.Errorf("optimizer state: parameter %q is not registered with this tape", name)
		}
		if parameter.value.dtype != DTypeFloat32 {
			return nil, fmt.Errorf("optimizer state: parameter %q has dtype %s, want float32", name, parameter.value.dtype)
		}
		shape, _, count, err := makeLayout(parameter.value.shape)
		if err != nil {
			return nil, fmt.Errorf("optimizer state: parameter %q has invalid shape: %w", name, err)
		}
		if parameter.value.device == nil {
			if len(parameter.value.data) != count {
				return nil, fmt.Errorf("optimizer state: parameter %q has %d values, want %d", name, len(parameter.value.data), count)
			}
		} else {
			if parameter.value.device.Released() {
				return nil, fmt.Errorf("optimizer state: parameter %q has a released device buffer", name)
			}
			if parameter.value.device.Device().ID == "" || parameter.value.device.Len() != count {
				return nil, fmt.Errorf("optimizer state: parameter %q has an invalid device buffer for shape %v", name, shape)
			}
		}
		targets = append(targets, optimizerStateTarget{name: name, parameter: parameter})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].name < targets[j].name })
	return targets, nil
}

func optimizerStatePrecisionCode(precision accel.Precision) (int64, error) {
	switch precision {
	case "", accel.PrecisionExact:
		return 0, nil
	case accel.PrecisionFloat32Rounded:
		return 1, nil
	case accel.PrecisionFloat32:
		return 2, nil
	default:
		return 0, fmt.Errorf("optimizer state: unsupported tape precision %q", precision)
	}
}

func optimizerStateFinite(values []float32, name string, nonnegative bool) error {
	for index, value := range values {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return fmt.Errorf("optimizer state: %s has a non-finite value at index %d", name, index)
		}
		if nonnegative && value < 0 {
			return fmt.Errorf("optimizer state: %s has a negative value at index %d", name, index)
		}
	}
	return nil
}

func optimizerStateDeviceValues(buffer *accel.DeviceBuffer, parameter *Tensor, name string, count int) ([]float32, error) {
	if buffer == nil {
		return nil, fmt.Errorf("optimizer state: %s device buffer is nil", name)
	}
	if buffer.Released() {
		return nil, fmt.Errorf("optimizer state: %s device buffer was released", name)
	}
	if parameter == nil || parameter.device == nil || parameter.device.Released() {
		return nil, fmt.Errorf("optimizer state: %s has no live resident parameter", name)
	}
	deviceID := parameter.device.Device().ID
	if deviceID == "" || buffer.Device().ID != deviceID || buffer.Len() != count {
		return nil, fmt.Errorf("optimizer state: %s device buffer does not match its parameter", name)
	}
	values, err := buffer.ToHost()
	if err != nil {
		return nil, fmt.Errorf("optimizer state: read %s from device: %w", name, err)
	}
	if len(values) != count {
		return nil, fmt.Errorf("optimizer state: %s has %d values, want %d", name, len(values), count)
	}
	if err := optimizerStateFinite(values, name, false); err != nil {
		return nil, err
	}
	return values, nil
}

func optimizerStateReadSlots(target optimizerStateTarget) (optimizerStateSlots, error) {
	parameter := target.parameter
	count := parameter.value.Len()
	slots := optimizerStateSlots{step: parameter.adamStep}
	if slots.step == math.MaxUint64 {
		return optimizerStateSlots{}, fmt.Errorf("optimizer state: parameter %q has a maximal step with no representable next step", target.name)
	}

	if parameter.adamMDev != nil || parameter.adamVDev != nil {
		if parameter.adamMDev == nil || parameter.adamVDev == nil {
			return optimizerStateSlots{}, fmt.Errorf("optimizer state: parameter %q has only one resident Adam moment", target.name)
		}
		if parameter.adamM != nil || parameter.adamV != nil {
			return optimizerStateSlots{}, fmt.Errorf("optimizer state: parameter %q has both host and resident Adam moments", target.name)
		}
		var err error
		slots.adamM, err = optimizerStateDeviceValues(parameter.adamMDev, parameter.value, target.name+" adam_m", count)
		if err != nil {
			return optimizerStateSlots{}, err
		}
		slots.adamV, err = optimizerStateDeviceValues(parameter.adamVDev, parameter.value, target.name+" adam_v", count)
		if err != nil {
			return optimizerStateSlots{}, err
		}
		slots.hasAdam = true
	} else {
		if (parameter.adamM == nil) != (parameter.adamV == nil) {
			return optimizerStateSlots{}, fmt.Errorf("optimizer state: parameter %q has only one host Adam moment", target.name)
		}
		if parameter.adamM != nil {
			if len(parameter.adamM) != count || len(parameter.adamV) != count {
				return optimizerStateSlots{}, fmt.Errorf("optimizer state: parameter %q Adam moments do not match shape %v", target.name, parameter.value.shape)
			}
			slots.adamM, slots.adamV = parameter.adamM, parameter.adamV
			slots.hasAdam = true
		}
	}

	if parameter.velocity != nil {
		if len(parameter.velocity) != count {
			return optimizerStateSlots{}, fmt.Errorf("optimizer state: parameter %q velocity does not match shape %v", target.name, parameter.value.shape)
		}
		slots.velocity = parameter.velocity
	}
	if err := optimizerStateFinite(slots.adamM, target.name+" adam_m", false); err != nil {
		return optimizerStateSlots{}, err
	}
	if err := optimizerStateFinite(slots.adamV, target.name+" adam_v", true); err != nil {
		return optimizerStateSlots{}, err
	}
	if err := optimizerStateFinite(slots.velocity, target.name+" velocity", false); err != nil {
		return optimizerStateSlots{}, err
	}
	if (slots.step > 0) != slots.hasAdam {
		return optimizerStateSlots{}, fmt.Errorf("optimizer state: parameter %q has inconsistent Adam step and moments", target.name)
	}
	return slots, nil
}

func makeOptimizerStatePrefix(name string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(name)) + "/"
}

func optimizerStateIntTensor(shape []int, values []int64) (*Tensor, error) {
	tensor, err := newInt64Tensor(shape, values)
	if err != nil {
		return nil, err
	}
	return tensor, nil
}

// SaveOptimizerState writes a versioned binary SafeTensors checkpoint of Adam,
// AdamW and momentum state, including resident Adam moments. named must give
// every parameter on this tape exactly one stable, nonempty UTF-8 name, such as
// the map from Sequential.NamedParameters. Moments use Value's internal layout.
// Weights, gradients, RNG, optimizer arguments and freeze settings are separate.
// Validation errors write nothing; an I/O error can leave partial output.
// Do not mutate the tape, parameters or mapping concurrently with this call.
func (t *Tape) SaveOptimizerState(w io.Writer, named map[string]*Parameter) error {
	if nilOptimizerStateEndpoint(w) {
		return fmt.Errorf("optimizer state: writer is nil")
	}
	targets, err := optimizerStateTargets(t, named)
	if err != nil {
		return err
	}
	precision, err := optimizerStatePrecisionCode(t.GetPrecision())
	if err != nil {
		return err
	}

	tensors := map[string]*Tensor{}
	version, err := optimizerStateIntTensor(nil, []int64{1})
	if err != nil {
		return fmt.Errorf("optimizer state: create version descriptor: %w", err)
	}
	precisionTensor, err := optimizerStateIntTensor(nil, []int64{precision})
	if err != nil {
		return fmt.Errorf("optimizer state: create precision descriptor: %w", err)
	}
	tensors["version"], tensors["precision"] = version, precisionTensor

	for _, target := range targets {
		parameter := target.parameter
		prefix := makeOptimizerStatePrefix(target.name)
		shapeValues := make([]int64, len(parameter.value.shape))
		for index, dimension := range parameter.value.shape {
			shapeValues[index] = int64(dimension)
		}
		shape, err := optimizerStateIntTensor([]int{len(shapeValues)}, shapeValues)
		if err != nil {
			return fmt.Errorf("optimizer state: create %q shape descriptor: %w", target.name, err)
		}
		dtype, err := optimizerStateIntTensor(nil, []int64{0})
		if err != nil {
			return fmt.Errorf("optimizer state: create %q dtype descriptor: %w", target.name, err)
		}
		step, err := optimizerStateIntTensor([]int{2}, []int64{
			int64(uint32(parameter.adamStep)),
			int64(uint32(parameter.adamStep >> 32)),
		})
		if err != nil {
			return fmt.Errorf("optimizer state: create %q step descriptor: %w", target.name, err)
		}
		tensors[prefix+"shape"], tensors[prefix+"dtype"], tensors[prefix+"step"] = shape, dtype, step

		slots, err := optimizerStateReadSlots(target)
		if err != nil {
			return err
		}
		if slots.hasAdam {
			first, err := newFloat32Tensor(parameter.value.shape, slots.adamM)
			if err != nil {
				return fmt.Errorf("optimizer state: create %q first moment: %w", target.name, err)
			}
			second, err := newFloat32Tensor(parameter.value.shape, slots.adamV)
			if err != nil {
				return fmt.Errorf("optimizer state: create %q second moment: %w", target.name, err)
			}
			tensors[prefix+"adam_m"], tensors[prefix+"adam_v"] = first, second
		}
		if slots.velocity != nil {
			velocity, err := newFloat32Tensor(parameter.value.shape, slots.velocity)
			if err != nil {
				return fmt.Errorf("optimizer state: create %q velocity: %w", target.name, err)
			}
			tensors[prefix+"velocity"] = velocity
		}
	}
	if err := SaveSafeTensors(w, tensors); err != nil {
		return fmt.Errorf("save optimizer state: %w", err)
	}
	return nil
}

func optimizerStateDescriptor(tensors map[string]*Tensor, name string, shape []int) (*Tensor, error) {
	tensor := tensors[name]
	if tensor == nil {
		return nil, fmt.Errorf("optimizer state: missing tensor %q", name)
	}
	if tensor.dtype != DTypeInt64 || !sameShape(tensor.shape, shape) {
		return nil, fmt.Errorf("optimizer state: tensor %q must be I64 with shape %v", name, shape)
	}
	return tensor, nil
}

func optimizerStateLoadSlots(tensors map[string]*Tensor, target optimizerStateTarget) (optimizerStateSlots, map[string]struct{}, error) {
	parameter := target.parameter
	prefix := makeOptimizerStatePrefix(target.name)
	expected := map[string]struct{}{
		prefix + "shape": {},
		prefix + "dtype": {},
		prefix + "step":  {},
	}

	shapeTensor, err := optimizerStateDescriptor(tensors, prefix+"shape", []int{len(parameter.value.shape)})
	if err != nil {
		return optimizerStateSlots{}, nil, err
	}
	if len(shapeTensor.int64Data) != len(parameter.value.shape) {
		return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: parameter %q shape descriptor has the wrong length", target.name)
	}
	for index, dimension := range parameter.value.shape {
		if shapeTensor.int64Data[index] < 0 || uint64(shapeTensor.int64Data[index]) != uint64(dimension) {
			return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: parameter %q shape does not match target shape %v", target.name, parameter.value.shape)
		}
	}

	dtypeTensor, err := optimizerStateDescriptor(tensors, prefix+"dtype", nil)
	if err != nil {
		return optimizerStateSlots{}, nil, err
	}
	if len(dtypeTensor.int64Data) != 1 || dtypeTensor.int64Data[0] != 0 {
		return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: parameter %q has unsupported dtype descriptor", target.name)
	}

	stepTensor, err := optimizerStateDescriptor(tensors, prefix+"step", []int{2})
	if err != nil {
		return optimizerStateSlots{}, nil, err
	}
	if len(stepTensor.int64Data) != 2 {
		return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: parameter %q step descriptor must have two limbs", target.name)
	}
	for _, limb := range stepTensor.int64Data {
		if limb < 0 || uint64(limb) > math.MaxUint32 {
			return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: parameter %q has an invalid step limb %d", target.name, limb)
		}
	}
	step := uint64(uint32(stepTensor.int64Data[0])) | uint64(uint32(stepTensor.int64Data[1]))<<32
	if step == math.MaxUint64 {
		return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: parameter %q has a maximal step with no representable next step", target.name)
	}

	slots := optimizerStateSlots{step: step}
	first, hasFirst := tensors[prefix+"adam_m"]
	second, hasSecond := tensors[prefix+"adam_v"]
	if hasFirst {
		expected[prefix+"adam_m"] = struct{}{}
	}
	if hasSecond {
		expected[prefix+"adam_v"] = struct{}{}
	}
	if hasFirst != hasSecond {
		return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: parameter %q must have both Adam moments or neither", target.name)
	}
	if hasFirst {
		slots.hasAdam = true
		for name, tensor := range map[string]*Tensor{prefix + "adam_m": first, prefix + "adam_v": second} {
			if tensor.dtype != DTypeFloat32 || !sameShape(tensor.shape, parameter.value.shape) {
				return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: tensor %q must be F32 with shape %v", name, parameter.value.shape)
			}
		}
		slots.adamM = append([]float32{}, first.data...)
		slots.adamV = append([]float32{}, second.data...)
	}
	if (step > 0) != slots.hasAdam {
		return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: parameter %q has inconsistent Adam step and moments", target.name)
	}

	if velocity, exists := tensors[prefix+"velocity"]; exists {
		expected[prefix+"velocity"] = struct{}{}
		if velocity.dtype != DTypeFloat32 || !sameShape(velocity.shape, parameter.value.shape) {
			return optimizerStateSlots{}, nil, fmt.Errorf("optimizer state: tensor %q must be F32 with shape %v", prefix+"velocity", parameter.value.shape)
		}
		slots.velocity = append([]float32{}, velocity.data...)
	}
	if err := optimizerStateFinite(slots.adamM, target.name+" adam_m", false); err != nil {
		return optimizerStateSlots{}, nil, err
	}
	if err := optimizerStateFinite(slots.adamV, target.name+" adam_v", true); err != nil {
		return optimizerStateSlots{}, nil, err
	}
	if err := optimizerStateFinite(slots.velocity, target.name+" velocity", false); err != nil {
		return optimizerStateSlots{}, nil, err
	}
	return slots, expected, nil
}

// LoadOptimizerState restores the complete optimizer checkpoint saved for named.
// Names, shapes, dtypes and GetPrecision must match. All validation and device
// allocations finish before any state changes; an error leaves every parameter,
// gradient and optimizer slot unchanged. Uninitialized slots clear old state.
// Weights, graph history and caller policies are untouched. Do not mutate the
// tape, parameters or mapping concurrently with this call.
func (t *Tape) LoadOptimizerState(r io.Reader, named map[string]*Parameter) error {
	if nilOptimizerStateEndpoint(r) {
		return fmt.Errorf("optimizer state: reader is nil")
	}
	targets, err := optimizerStateTargets(t, named)
	if err != nil {
		return err
	}
	precision, err := optimizerStatePrecisionCode(t.GetPrecision())
	if err != nil {
		return err
	}

	entries, data, err := optimizerStateReadCheckpoint(r, targets)
	if err != nil {
		return err
	}
	tensors, err := loadSafeTensorData(entries, data)
	if err != nil {
		return fmt.Errorf("optimizer state: decode SafeTensors: %w", err)
	}

	expected := map[string]struct{}{"version": {}, "precision": {}}
	version, err := optimizerStateDescriptor(tensors, "version", nil)
	if err != nil {
		return err
	}
	if len(version.int64Data) != 1 || version.int64Data[0] != 1 {
		return fmt.Errorf("optimizer state: unsupported version")
	}
	precisionTensor, err := optimizerStateDescriptor(tensors, "precision", nil)
	if err != nil {
		return err
	}
	if len(precisionTensor.int64Data) != 1 || precisionTensor.int64Data[0] != precision {
		return fmt.Errorf("optimizer state: checkpoint precision does not match tape precision")
	}

	loaded := make(map[*Parameter]optimizerStateSlots, len(targets))
	for _, target := range targets {
		slots, keys, err := optimizerStateLoadSlots(tensors, target)
		if err != nil {
			return err
		}
		loaded[target.parameter] = slots
		for key := range keys {
			expected[key] = struct{}{}
		}
	}
	if len(tensors) != len(expected) {
		return fmt.Errorf("optimizer state: checkpoint has %d tensors, want exactly %d", len(tensors), len(expected))
	}
	for key := range tensors {
		if _, exists := expected[key]; !exists {
			return fmt.Errorf("optimizer state: unexpected tensor %q", key)
		}
	}

	staged := make(map[*Parameter][2]*accel.DeviceBuffer)
	var stagedBuffers []*accel.DeviceBuffer
	for _, target := range targets {
		parameter := target.parameter
		slots := loaded[parameter]
		if parameter.value.device == nil || !slots.hasAdam {
			continue
		}
		device := parameter.value.Device()
		first, err := accel.NewDeviceBuffer(device, slots.adamM)
		if err != nil {
			for _, buffer := range stagedBuffers {
				buffer.Release()
			}
			return fmt.Errorf("optimizer state: stage %q first moment: %w", target.name, err)
		}
		stagedBuffers = append(stagedBuffers, first)
		second, err := accel.NewDeviceBuffer(device, slots.adamV)
		if err != nil {
			for _, buffer := range stagedBuffers {
				buffer.Release()
			}
			return fmt.Errorf("optimizer state: stage %q second moment: %w", target.name, err)
		}
		stagedBuffers = append(stagedBuffers, second)
		staged[parameter] = [2]*accel.DeviceBuffer{first, second}
	}

	var oldBuffers []*accel.DeviceBuffer
	for _, target := range targets {
		parameter := target.parameter
		if parameter.adamMDev != nil {
			oldBuffers = append(oldBuffers, parameter.adamMDev)
		}
		if parameter.adamVDev != nil {
			oldBuffers = append(oldBuffers, parameter.adamVDev)
		}
	}

	// The decoded slices are already private copies. Commit only assignments;
	// keep allocation and other fallible work outside this loop.
	for _, target := range targets {
		parameter := target.parameter
		slots := loaded[parameter]
		parameter.velocity = slots.velocity
		parameter.adamM, parameter.adamV = nil, nil
		parameter.adamMDev, parameter.adamVDev = nil, nil
		if parameter.value.device != nil {
			if moments, exists := staged[parameter]; exists {
				parameter.adamMDev, parameter.adamVDev = moments[0], moments[1]
			}
		} else if slots.hasAdam {
			parameter.adamM, parameter.adamV = slots.adamM, slots.adamV
		}
		parameter.adamStep = slots.step
	}
	for _, buffer := range oldBuffers {
		buffer.Release()
	}
	return nil
}
