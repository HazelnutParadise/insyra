package nn

import (
	"fmt"
	"math"
)

// softmaxCERow holds the per-row quantities of softmax cross-entropy in
// PrecisionFloat32Rounded. Every line is one correctly rounded operation or an
// exact sum rounded once, so a device that computes the same lines returns the
// same bits.
type softmaxCERow struct {
	e      []float32 // e[c] = exp(x[c] - max)
	d      []float32 // d[c] = x[c] - max
	s      float32   // exact sum of e, rounded once
	loss   float32   // log(s) - d[label]
	finite bool
}

// roundedSumOf returns the exact sum of values rounded once to float32, with an
// exactly-zero sum as +0.
func roundedSumOf(values []float32) float32 {
	var fast float64ProductSum
	for _, v := range values {
		fast.add(v, 1)
	}
	if result, ok := fast.result(); ok {
		return result
	}
	var exact exactAccumulator
	for _, v := range values {
		exact.addProduct(v, 1)
	}
	return exact.float32()
}

// mulRounded is a*b with an exactly-zero product reported as +0, as an exact
// product summed into a zeroed register is; a nonzero product that underflows
// keeps its sign, as IEEE gives it.
func mulRounded(a, b float32) float32 {
	product := a * b
	if (a == 0 || b == 0) && product == 0 {
		return 0
	}
	return product
}

func softmaxCEFiniteRow(x []float32) bool {
	for _, v := range x {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return false
		}
	}
	return true
}

func softmaxCERowOf(x []float32, label int) softmaxCERow {
	row := softmaxCERow{finite: softmaxCEFiniteRow(x)}
	if !row.finite {
		row.loss = float32(math.NaN())
		return row
	}
	maxValue := x[0]
	for _, v := range x[1:] {
		if v > maxValue {
			maxValue = v
		}
	}
	row.d = make([]float32, len(x))
	row.e = make([]float32, len(x))
	for c, v := range x {
		row.d[c] = sum2Rounded(v, 1, -1, maxValue)
		row.e[c] = expFloat32(row.d[c])
	}
	row.s = roundedSumOf(row.e)
	row.loss = sum2Rounded(logFloat32(row.s), 1, -1, row.d[label])
	return row
}

// softmaxCrossEntropyRoundedForward returns the mean loss of logits [N,C]
// against int64 labels [N]: the exact sum of the row losses rounded once,
// divided by the row count.
func softmaxCrossEntropyRoundedForward(logits, labels *Tensor) (*Tensor, error) {
	if err := checkSoftmaxCEInputs(logits, labels); err != nil {
		return nil, err
	}
	rows, classes := logits.shape[0], logits.shape[1]
	losses := make([]float32, rows)
	for r := 0; r < rows; r++ {
		losses[r] = softmaxCERowOf(logits.data[r*classes:(r+1)*classes], int(labels.int64Data[r])).loss
	}
	loss := roundedSumOf(losses) / float32(rows)
	return newFloat32Tensor(nil, []float32{loss})
}

// softmaxCrossEntropyRoundedVJP returns the gradient with respect to the
// logits: p[c] = e[c]/s, g[c] = p[c]*scale with scale = upstream/rows, and at
// the label g = g - scale, each one correctly rounded operation.
func softmaxCrossEntropyRoundedVJP(logits, labels, upstream *Tensor) (*Tensor, error) {
	if err := requireFloat32(upstream, "cross-entropy upstream"); err != nil {
		return nil, err
	}
	if len(upstream.shape) != 0 {
		return nil, fmt.Errorf("cross-entropy upstream must be scalar, got shape %v", upstream.shape)
	}
	rows, classes := logits.shape[0], logits.shape[1]
	gradient, err := newZeroFloat32Tensor(logits.shape)
	if err != nil {
		return nil, err
	}
	scale := upstream.data[0] / float32(rows)
	for r := 0; r < rows; r++ {
		label := int(labels.int64Data[r])
		row := softmaxCERowOf(logits.data[r*classes:(r+1)*classes], label)
		out := gradient.data[r*classes : (r+1)*classes]
		if !row.finite {
			for c := range out {
				out[c] = float32(math.NaN())
			}
			continue
		}
		for c := range out {
			out[c] = mulRounded(row.e[c]/row.s, scale)
		}
		out[label] = sum2Rounded(out[label], 1, -1, scale)
	}
	return gradient, nil
}

func checkSoftmaxCEInputs(logits, labels *Tensor) error {
	if err := requireFloat32(logits, "cross-entropy logits"); err != nil {
		return err
	}
	if labels == nil || labels.dtype != DTypeInt64 {
		return fmt.Errorf("cross-entropy labels must have dtype int64")
	}
	if labels.device != nil {
		return residentRefusal(labels, "cross-entropy labels")
	}
	if len(logits.shape) != 2 || len(labels.shape) != 1 || logits.shape[0] != labels.shape[0] {
		return fmt.Errorf("cross-entropy requires logits [N,C] and labels [N], got %v and %v", logits.shape, labels.shape)
	}
	rows, classes := logits.shape[0], logits.shape[1]
	if rows == 0 || classes == 0 {
		return fmt.Errorf("cross-entropy requires non-empty logits")
	}
	if rows >= 1<<24 {
		return fmt.Errorf("cross-entropy in PrecisionFloat32Rounded supports fewer than 2^24 rows, got %d", rows)
	}
	for r := 0; r < rows; r++ {
		if label := labels.int64Data[r]; label < 0 || label >= int64(classes) {
			return fmt.Errorf("cross-entropy label %d at row index %d is outside [0,%d)", label, r, classes)
		}
	}
	return nil
}
