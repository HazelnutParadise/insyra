package nn

import (
	"math"
	"testing"
)

func TestBatchNormalizationTrainingVJPFiniteDifference(t *testing.T) {
	inputValues := []float32{-1.2, 0.3, 1.7, -0.4, 0.8, 2.1, -0.6, 0.5}
	scaleValues := []float32{1.3, 0.7}
	biasValues := []float32{-0.2, 0.4}
	upstreamValues := []float32{0.2, -0.5, 0.7, 0.1, -0.3, 0.9, -0.8, 0.6}
	input := mustTestTensor(t, []int{2, 2, 2, 1}, inputValues)
	scale := mustTestTensor(t, []int{2}, scaleValues)
	bias := mustTestTensor(t, []int{2}, biasValues)
	runningMean := mustTestTensor(t, []int{2}, []float32{0.25, -0.5})
	runningVariance := mustTestTensor(t, []int{2}, []float32{1.5, 0.75})
	upstream := mustTestTensor(t, []int{2, 2, 2, 1}, upstreamValues)

	tape := NewTape()
	trackedInput, err := tape.Param(input)
	if err != nil {
		t.Fatal(err)
	}
	trackedScale, err := tape.Param(scale)
	if err != nil {
		t.Fatal(err)
	}
	trackedBias, err := tape.Param(bias)
	if err != nil {
		t.Fatal(err)
	}
	output, err := tape.BatchNormalizationTraining(input, scale, bias, runningMean, runningVariance, 0.2, 1e-5)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := tape.ReduceMean(mustTapeMul(t, tape, output, upstream), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	analytic := map[string][]float32{
		"input": trackedInput.Grad().data,
		"scale": trackedScale.Grad().data,
		"bias":  trackedBias.Grad().data,
	}
	base := map[string][]float32{
		"input": append([]float32(nil), inputValues...),
		"scale": append([]float32(nil), scaleValues...),
		"bias":  append([]float32(nil), biasValues...),
	}
	for name, values := range base {
		for index := range values {
			const step = float32(1e-3)
			plus := cloneCatalogValues(base)
			minus := cloneCatalogValues(base)
			plus[name][index] += step
			minus[name][index] -= step
			finite := (batchNormTrainingLoss(t, plus, upstreamValues) - batchNormTrainingLoss(t, minus, upstreamValues)) / (2 * step)
			if diff := math.Abs(float64(analytic[name][index] - finite)); diff > 3e-3 {
				t.Fatalf("%s gradient[%d] = %g, finite difference = %g, diff = %g", name, index, analytic[name][index], finite, diff)
			}
		}
	}
	if got := runningMean.data; !almostCatalogEqual(got, []float32{0.3, -0.34}, 1e-6) {
		t.Fatalf("running mean = %v", got)
	}
	if got := runningVariance.data; !almostCatalogEqual(got, []float32{1.572, 0.82}, 1e-5) {
		t.Fatalf("running variance = %v", got)
	}
}

func TestEmbeddingVJPFiniteDifferenceRepeatedIndices(t *testing.T) {
	values := []float32{0.2, -0.5, 1.1, 0.7, -0.3, 0.9, -1.2, 0.4, 0.8, -0.6, 0.1, 1.3}
	indicesValues := []int64{1, 3, 1, 0, 3}
	upstreamValues := []float32{0.3, -0.4, 0.8, 0.5, -0.7, 0.2, 0.1, 0.6, -0.9, 0.4, 0.2, -0.1, 0.7, -0.3, 0.9}
	table := mustTestTensor(t, []int{4, 3}, values)
	indices, err := NewInt64Tensor([]int{len(indicesValues)}, indicesValues)
	if err != nil {
		t.Fatal(err)
	}
	upstream := mustTestTensor(t, []int{len(indicesValues), 3}, upstreamValues)
	tape := NewTape()
	parameter, err := tape.Param(table)
	if err != nil {
		t.Fatal(err)
	}
	output, err := tape.Embedding(table, indices)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := tape.ReduceMean(mustTapeMul(t, tape, output, upstream), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	gradient := parameter.Grad().data
	for index := range values {
		const step = float32(1e-3)
		plus := append([]float32(nil), values...)
		minus := append([]float32(nil), values...)
		plus[index] += step
		minus[index] -= step
		finite := (embeddingLoss(t, plus, indicesValues, upstreamValues) - embeddingLoss(t, minus, indicesValues, upstreamValues)) / (2 * step)
		if diff := math.Abs(float64(gradient[index] - finite)); diff > 2e-3 {
			t.Fatalf("gradient[%d] = %g, finite difference = %g, diff = %g", index, gradient[index], finite, diff)
		}
	}
	if gradient[1*3] == 0 || gradient[3*3] == 0 {
		t.Fatalf("repeated-index rows did not accumulate: %v", gradient)
	}
}

func TestLayerCatalogLoadWeightsAndBatchNormPredict(t *testing.T) {
	tape := NewTape(7)
	model, err := NewSequential(tape,
		Conv2D(1, 2, 1),
		BatchNorm2D(2),
		ReLU(),
		MaxPool2D(1),
		GlobalAvgPool(),
		NewFlatten(),
		Dense(2, 2),
	)
	if err != nil {
		t.Fatal(err)
	}
	weights := map[string]*Tensor{
		"0.weight":              mustTestTensor(t, []int{2, 1, 1, 1}, []float32{1, -1}),
		"0.bias":                mustTestTensor(t, []int{2}, []float32{0, 0}),
		"1.weight":              mustTestTensor(t, []int{2}, []float32{1, 1}),
		"1.bias":                mustTestTensor(t, []int{2}, []float32{0, 0}),
		"1.running_mean":        mustTestTensor(t, []int{2}, []float32{0.5, -0.5}),
		"1.running_var":         mustTestTensor(t, []int{2}, []float32{0.25, 0.25}),
		"1.num_batches_tracked": mustTestTensor(t, []int{}, []float32{3}),
		"6.weight":              mustTestTensor(t, []int{2, 2}, []float32{1, 2, 3, 4}),
		"6.bias":                mustTestTensor(t, []int{2}, []float32{0.1, -0.2}),
	}
	if err := model.LoadWeights(weights); err != nil {
		t.Fatalf("LoadWeights: %v", err)
	}
	input := mustTestTensor(t, []int{1, 1, 1, 1}, []float32{1.0})
	got, err := model.Predict(input)
	if err != nil {
		t.Fatalf("Predict: %v", err)
	}
	if !almostCatalogEqual(got.data, []float32{1.1, 2.8}, 1e-3) {
		t.Fatalf("Predict = %v", got.data)
	}
}

func mustTapeMul(t *testing.T, tape *Tape, left, right *Tensor) *Tensor {
	t.Helper()
	value, err := tape.Mul(left, right)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func cloneCatalogValues(values map[string][]float32) map[string][]float32 {
	clone := make(map[string][]float32, len(values))
	for name, value := range values {
		clone[name] = append([]float32(nil), value...)
	}
	return clone
}

func batchNormTrainingLoss(t *testing.T, values map[string][]float32, upstream []float32) float32 {
	t.Helper()
	input := mustTestTensor(t, []int{2, 2, 2, 1}, values["input"])
	scale := mustTestTensor(t, []int{2}, values["scale"])
	bias := mustTestTensor(t, []int{2}, values["bias"])
	runningMean := mustTestTensor(t, []int{2}, []float32{0.25, -0.5})
	runningVariance := mustTestTensor(t, []int{2}, []float32{1.5, 0.75})
	upstreamTensor := mustTestTensor(t, []int{2, 2, 2, 1}, upstream)
	tape := NewTape()
	output, err := tape.BatchNormalizationTraining(input, scale, bias, runningMean, runningVariance, 0.2, 1e-5)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := ReduceMean(mustTapeMul(t, tape, output, upstreamTensor), nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return loss.data[0]
}

func embeddingLoss(t *testing.T, values []float32, indices []int64, upstream []float32) float32 {
	t.Helper()
	table := mustTestTensor(t, []int{4, 3}, values)
	indexTensor, err := NewInt64Tensor([]int{len(indices)}, indices)
	if err != nil {
		t.Fatal(err)
	}
	upstreamTensor := mustTestTensor(t, []int{len(indices), 3}, upstream)
	output, err := (&Tape{}).Embedding(table, indexTensor)
	if err != nil {
		t.Fatal(err)
	}
	product, err := Mul(output, upstreamTensor)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := ReduceMean(product, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	return loss.data[0]
}

func almostCatalogEqual(got, want []float32, tolerance float32) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if float32(math.Abs(float64(got[index]-want[index]))) > tolerance {
			return false
		}
	}
	return true
}
