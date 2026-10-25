package nn

import (
	"fmt"

	"github.com/HazelnutParadise/insyra/accel"
)

// SetPrecision selects how the tape's arithmetic is defined. The default,
// accel.PrecisionExact, keeps every released behaviour: the host kernels, and
// AdamW with float64 intermediates. accel.PrecisionFloat32Rounded defines each
// result as the exact value of its operation rounded once to float32, so the
// same step returns the same bits on the host, on any GPU, and when a GPU is
// missing and the host runs it instead. Resident tensors need that mode, since
// a device cannot reproduce the host's default arithmetic.
//
// accel.PrecisionFloat32 is PrecisionFloat32Rounded for every operation of
// the tape, and in addition lets Custom record an operation on resident
// tensors, whose kernel is the caller's own and returns whatever the device
// computes. Use it only for a step that runs a kernel of your own through
// accel.Submit: its results then depend on the device, and the tape no longer
// promises the same bits on another device or on the host.
//
// In this release the mode covers MatMul, Add and their gradients, gradient
// accumulation, SGD, Adam and AdamW. Other operations keep their default arithmetic on the
// host and are refused on resident tensors.
func (t *Tape) SetPrecision(precision accel.Precision) error {
	switch precision {
	case "", accel.PrecisionExact:
		t.precision = accel.PrecisionExact
	case accel.PrecisionFloat32Rounded:
		t.precision = accel.PrecisionFloat32Rounded
	case accel.PrecisionFloat32:
		t.precision = accel.PrecisionFloat32
	default:
		return fmt.Errorf("tape precision %q is not supported; use %q, %q or %q", precision, accel.PrecisionExact, accel.PrecisionFloat32Rounded, accel.PrecisionFloat32)
	}
	return nil
}

// GetPrecision reports the precision the tape computes in.
func (t *Tape) GetPrecision() accel.Precision {
	if t.precision == "" {
		return accel.PrecisionExact
	}
	return t.precision
}

// rounded reports whether the built-in operations compute in the rounded
// float32 arithmetic, which a PrecisionFloat32 tape does too.
func (t *Tape) rounded() bool {
	return t.precision == accel.PrecisionFloat32Rounded || t.precision == accel.PrecisionFloat32
}

// Clear forgets the recorded operations and releases every device tensor the
// tape created for the step: operation outputs and gradients. Parameters and
// optimizer state stay, so training continues across a Clear. Tensors the
// caller supplied, including the inputs of the step, are the caller's to
// Release.
func (t *Tape) Clear() {
	t.ops = nil
	t.grads = make(map[*Tensor]*Tensor)
	releaseAll(t.withoutParameters(t.owned))
	releaseAll(t.gradOwned)
	for _, release := range t.releasers {
		release()
	}
	t.releasers = nil
	t.owned, t.gradOwned = nil, nil
	for _, parameter := range t.params {
		parameter.grad = nil
	}
}

// Close releases everything the tape holds on a device: what Clear releases,
// plus the parameters. It is safe to call more than once. A closed tape
// refuses new operations.
func (t *Tape) Close() {
	if t.closed {
		return
	}
	t.Clear()
	for _, parameter := range t.params {
		parameter.value.Release()
		parameter.releaseState()
	}
	t.closed = true
}

func (t *Tape) checkOpen() error {
	if t.closed {
		return fmt.Errorf("tape is closed")
	}
	return nil
}

// withoutParameters drops tensors that were marked as parameters after an
// operation produced them: a parameter belongs to the tape until Close, not
// to the step that created it.
func (t *Tape) withoutParameters(tensors []*Tensor) []*Tensor {
	kept := make([]*Tensor, 0, len(tensors))
	for _, tensor := range tensors {
		if t.marked[tensor] == nil {
			kept = append(kept, tensor)
		}
	}
	return kept
}

func releaseAll(tensors []*Tensor) {
	for _, tensor := range tensors {
		tensor.Release()
	}
}

// own registers a device tensor the tape created, so Clear releases it.
func (t *Tape) own(tensor *Tensor) *Tensor {
	if tensor != nil && tensor.device != nil {
		if t.inBackward {
			t.passOwned = append(t.passOwned, tensor)
		} else {
			t.owned = append(t.owned, tensor)
		}
	}
	return tensor
}

// residency says where an operation's operands live: all on the host, all on
// one device, or an error when they are mixed. A resident tensor on a tape
// that is not in PrecisionFloat32Rounded is an error too, because only that
// mode defines what the device computes.
func (t *Tape) residency(op string, tensors ...*Tensor) (bool, error) {
	if err := t.checkOpen(); err != nil {
		return false, err
	}
	var device *Tensor
	for _, tensor := range tensors {
		if tensor == nil {
			return false, fmt.Errorf("autodiff %s inputs must not be nil", op)
		}
		if tensor.device == nil {
			continue
		}
		if device == nil {
			device = tensor
		} else if device.Device().ID != tensor.Device().ID {
			return false, fmt.Errorf("autodiff %s operands live on different devices (%s and %s)", op, device.Device().ID, tensor.Device().ID)
		}
	}
	if device == nil {
		return false, nil
	}
	for _, tensor := range tensors {
		if tensor.device == nil {
			return false, fmt.Errorf("autodiff %s operands live on the device %s and on the host; move the host operand with To", op, device.Device().ID)
		}
	}
	if !t.rounded() {
		return false, fmt.Errorf("autodiff %s: tensors on a device need SetPrecision(accel.PrecisionFloat32Rounded)", op)
	}
	return true, nil
}

// matMulResident multiplies two resident 2-D tensors and records a VJP whose
// two products read the transposes in place.
func (t *Tape) matMulResident(a, b *Tensor) (*Tensor, error) {
	if a.dtype != DTypeFloat32 || b.dtype != DTypeFloat32 {
		return nil, fmt.Errorf("autodiff matmul on a device needs float32 operands, got %s and %s", dtypeName(a.dtype), dtypeName(b.dtype))
	}
	if len(a.shape) != 2 || len(b.shape) != 2 || a.shape[1] != b.shape[0] {
		return nil, fmt.Errorf("autodiff matmul on a device needs 2-D operands with matching inner dimensions, got %v and %v", a.shape, b.shape)
	}
	m, k, n := a.shape[0], a.shape[1], b.shape[1]
	output, err := t.residentMatMul(a, b, m, k, n, false, false, []int{m, n})
	if err != nil {
		return nil, err
	}
	t.record("MatMul", []*Tensor{a, b}, output, func(upstream *Tensor) ([]*Tensor, error) {
		if !upstream.onDevice() {
			return nil, fmt.Errorf("matmul upstream must be on the device")
		}
		dA, err := t.residentMatMul(upstream, b, m, n, k, false, true, []int{m, k})
		if err != nil {
			return nil, err
		}
		dB, err := t.residentMatMul(a, upstream, k, m, n, true, false, []int{k, n})
		if err != nil {
			return nil, err
		}
		return []*Tensor{dA, dB}, nil
	})
	return output, nil
}

func (t *Tape) residentMatMul(a, b *Tensor, m, k, n int, transA, transB bool, shape []int) (*Tensor, error) {
	buffer, err := accel.RoundedMatMul(a.device, b.device, m, k, n, transA, transB)
	if err != nil {
		return nil, err
	}
	output, err := newResidentTensor(shape, buffer)
	if err != nil {
		return nil, err
	}
	return t.own(output), nil
}

// trailingBroadcast reports whether b broadcasts over a's trailing dimensions,
// the shape of a bias added to a batch.
func trailingBroadcast(a, b []int) bool {
	if len(b) > len(a) {
		return false // the result would have more dimensions than a
	}
	stripped := b
	for len(stripped) > 0 && stripped[0] == 1 {
		stripped = stripped[1:]
	}
	if len(stripped) > len(a) {
		return false
	}
	offset := len(a) - len(stripped)
	for i, dimension := range stripped {
		if a[offset+i] != dimension {
			return false
		}
	}
	return true
}

func sizeOf(shape []int) int {
	count := 1
	for _, dimension := range shape {
		count *= dimension
	}
	return count
}

// addResident adds two resident tensors, b broadcasting over a's trailing
// dimensions or matching it exactly.
func (t *Tape) addResident(a, b *Tensor) (*Tensor, error) {
	if a.dtype != DTypeFloat32 || b.dtype != DTypeFloat32 {
		return nil, fmt.Errorf("autodiff add on a device needs float32 operands, got %s and %s", dtypeName(a.dtype), dtypeName(b.dtype))
	}
	if !trailingBroadcast(a.shape, b.shape) {
		return nil, fmt.Errorf("autodiff add on a device needs the second operand to match or broadcast over the trailing dimensions of the first, got %v and %v", a.shape, b.shape)
	}
	sum, err := accel.RoundedAdd(a.device, b.device)
	if err != nil {
		return nil, err
	}
	output, err := newResidentTensor(a.shape, sum)
	if err != nil {
		return nil, err
	}
	t.own(output)
	t.record("Add", []*Tensor{a, b}, output, func(upstream *Tensor) ([]*Tensor, error) {
		if !upstream.onDevice() {
			return nil, fmt.Errorf("add upstream must be on the device")
		}
		if sameShape(a.shape, b.shape) {
			return []*Tensor{upstream, upstream}, nil
		}
		columns := sizeOf(b.shape)
		if columns == 0 {
			empty, err := accel.NewZeroDeviceBuffer(upstream.Device(), 0)
			if err != nil {
				return nil, err
			}
			dB, err := newResidentTensor(b.shape, empty)
			if err != nil {
				return nil, err
			}
			return []*Tensor{upstream, t.own(dB)}, nil
		}
		reduced, err := accel.RoundedSumRows(upstream.device, sizeOf(a.shape)/columns, columns)
		if err != nil {
			return nil, err
		}
		dB, err := newResidentTensor(b.shape, reduced)
		if err != nil {
			return nil, err
		}
		return []*Tensor{upstream, t.own(dB)}, nil
	})
	return output, nil
}

// --- host arithmetic in PrecisionFloat32Rounded -----------------------------

// roundedZero turns a negative zero into +0: in this mode an exactly-zero
// result is +0, as when it is accumulated into a zeroed output.
func roundedZero(value float32) float32 {
	if value == 0 {
		return 0
	}
	return value
}

func addRounded(a, b *Tensor) (*Tensor, error) {
	output, err := Add(a, b)
	if err != nil {
		return nil, err
	}
	for i, value := range output.data {
		output.data[i] = roundedZero(value)
	}
	return output, nil
}

// reduceBroadcastRounded sums upstream down to targetShape in the cases the
// device also handles, each element the exact sum rounded once. Other shapes
// keep the default reduction.
func reduceBroadcastRounded(upstream *Tensor, targetShape []int) (*Tensor, error) {
	if sameShape(upstream.shape, targetShape) {
		return copyTensor(upstream)
	}
	if !trailingBroadcast(upstream.shape, targetShape) {
		return reduceBroadcastGradientTape(upstream, targetShape)
	}
	columns := sizeOf(targetShape)
	result, err := newZeroFloat32Tensor(targetShape)
	if err != nil {
		return nil, err
	}
	if columns == 0 {
		return result, nil // an empty target has nothing to sum
	}
	rows := len(upstream.data) / columns
	var fast float64ProductSum
	var exact exactAccumulator
	for column := 0; column < columns; column++ {
		fast.reset()
		for row := 0; row < rows; row++ {
			fast.add(upstream.data[row*columns+column], 1)
		}
		if value, ok := fast.result(); ok {
			result.data[column] = value
			continue
		}
		exact.reset()
		for row := 0; row < rows; row++ {
			exact.addProduct(upstream.data[row*columns+column], 1)
		}
		result.data[column] = exact.float32()
	}
	return result, nil
}

// sgdRounded updates value to value - rate*gradient, each element the exact
// value rounded once.
func sgdRounded(value, gradient *Tensor, rate float32) {
	var fast float64ProductSum
	var exact exactAccumulator
	for i := range value.data {
		fast.reset()
		fast.add(value.data[i], 1)
		fast.add(-rate, gradient.data[i])
		if result, ok := fast.result(); ok {
			value.data[i] = result
			continue
		}
		exact.reset()
		exact.addProduct(value.data[i], 1)
		exact.addProduct(-rate, gradient.data[i])
		value.data[i] = exact.float32()
	}
}

func matMulVJPRounded(a, b, upstream *Tensor) ([]*Tensor, error) {
	dA, err := matMulRounded(upstream, b, false, true)
	if err != nil {
		return nil, err
	}
	dB, err := matMulRounded(a, upstream, true, false)
	if err != nil {
		return nil, err
	}
	return []*Tensor{dA, dB}, nil
}

func addVJPRounded(a, b, upstream *Tensor) ([]*Tensor, error) {
	if err := requireFloat32(upstream, "add upstream"); err != nil {
		return nil, err
	}
	dA, err := reduceBroadcastRounded(upstream, a.shape)
	if err != nil {
		return nil, err
	}
	dB, err := reduceBroadcastRounded(upstream, b.shape)
	if err != nil {
		return nil, err
	}
	return []*Tensor{dA, dB}, nil
}

// addGradient accumulates gradient into grads[input].
func (t *Tape) addGradient(grads map[*Tensor]*Tensor, input, gradient *Tensor) error {
	if input == nil {
		return fmt.Errorf("gradient input is nil")
	}
	if err := requireFloat32Any(gradient, "gradient"); err != nil {
		return err
	}
	existing := grads[input]
	if existing == nil {
		stored := gradient
		if gradient.device == nil {
			var err error
			stored, err = copyTensor(gradient)
			if err != nil {
				return err
			}
		}
		grads[input] = stored
		return nil
	}
	if !sameShape(existing.shape, gradient.shape) {
		return fmt.Errorf("gradient shape %v does not match existing shape %v", gradient.shape, existing.shape)
	}
	if existing.device != nil || gradient.device != nil {
		if existing.device == nil || gradient.device == nil {
			return fmt.Errorf("gradient for a tensor mixes the host and a device")
		}
		sum, err := accel.RoundedAdd(existing.device, gradient.device)
		if err != nil {
			return err
		}
		accumulated, err := newResidentTensor(existing.shape, sum)
		if err != nil {
			return err
		}
		grads[input] = t.own(accumulated)
		return nil
	}
	for index := range existing.data {
		sum := existing.data[index] + gradient.data[index]
		if t.rounded() {
			sum = roundedZero(sum)
		}
		existing.data[index] = sum
	}
	return nil
}

// softmaxCrossEntropyResident computes the mean cross-entropy of resident
// logits against resident int64 labels. The labels were checked when moved;
// the largest one is compared with the class count here, so no kernel needs to
// report a bad label.
func (t *Tape) softmaxCrossEntropyResident(logits, labels *Tensor) (*Tensor, error) {
	if logits.dtype != DTypeFloat32 || labels.dtype != DTypeInt64 {
		return nil, fmt.Errorf("cross-entropy on a device needs float32 logits and int64 labels")
	}
	if len(logits.shape) != 2 || len(labels.shape) != 1 || logits.shape[0] != labels.shape[0] {
		return nil, fmt.Errorf("cross-entropy requires logits [N,C] and labels [N], got %v and %v", logits.shape, labels.shape)
	}
	rows, classes := logits.shape[0], logits.shape[1]
	if rows == 0 || classes == 0 {
		return nil, fmt.Errorf("cross-entropy requires non-empty logits")
	}
	if rows >= 1<<24 {
		return nil, fmt.Errorf("cross-entropy in PrecisionFloat32Rounded supports fewer than 2^24 rows, got %d", rows)
	}
	if labels.maxLabel >= int64(classes) {
		return nil, fmt.Errorf("cross-entropy label %d is outside [0,%d)", labels.maxLabel, classes)
	}
	buffer, state, err := accel.RoundedSoftmaxCrossEntropy(logits.device, labels.device, rows, classes)
	if err != nil {
		return nil, err
	}
	t.releasers = append(t.releasers, state.Release)
	loss, err := newResidentTensor(nil, buffer)
	if err != nil {
		return nil, err
	}
	t.own(loss)
	t.record("SoftmaxCrossEntropy", []*Tensor{logits, labels}, loss, func(upstream *Tensor) ([]*Tensor, error) {
		if !upstream.onDevice() || len(upstream.shape) != 0 {
			return nil, fmt.Errorf("cross-entropy upstream must be a scalar on the device")
		}
		grad, err := accel.RoundedSoftmaxCrossEntropyGrad(state, labels.device, upstream.device, rows, classes)
		if err != nil {
			return nil, err
		}
		gradient, err := newResidentTensor(logits.shape, grad)
		if err != nil {
			return nil, err
		}
		return []*Tensor{t.own(gradient), nil}, nil
	})
	return loss, nil
}
