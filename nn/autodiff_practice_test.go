package nn

import (
	"math"
	"testing"
)

func TestDropoutPractice(t *testing.T) {
	const probability = float32(0.25)
	inputData := make([]float32, 100_000)
	for index := range inputData {
		inputData[index] = 2
	}
	input := mustTestTensor(t, []int{len(inputData)}, inputData)
	first, err := NewTape(20260803).Dropout(input, probability)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewTape(20260803).Dropout(input, probability)
	if err != nil {
		t.Fatal(err)
	}
	different, err := NewTape(20260804).Dropout(input, probability)
	if err != nil {
		t.Fatal(err)
	}

	keepScale := float32(1 / (1 - probability))
	kept := 0
	differentCount := 0
	for index, value := range first.data {
		if value == 0 {
			continue
		}
		kept++
		if value != inputData[index]*keepScale {
			t.Fatalf("kept value[%d] = %g, want %g", index, value, inputData[index]*keepScale)
		}
	}
	for index := range first.data {
		if first.data[index] != second.data[index] {
			t.Fatalf("same-seed mask differs at %d", index)
		}
		if first.data[index] != different.data[index] {
			differentCount++
		}
	}
	keepFraction := float64(kept) / float64(len(inputData))
	if math.Abs(keepFraction-0.75) > 0.01 {
		t.Fatalf("keep fraction = %g, want near 0.75", keepFraction)
	}
	if differentCount == 0 {
		t.Fatal("different seeds produced the same mask")
	}
}

func TestDropoutGradientAndFrozenMaskFiniteDifference(t *testing.T) {
	const seed = int64(77)
	const probability = float32(0.4)
	values := []float32{0.2, -0.3, 0.7, 1.1, -0.8, 0.5, 0.4, -0.6}
	parameter := mustTestTensor(t, []int{len(values)}, values)
	tape := NewTape(seed)
	tracked, err := tape.Param(parameter)
	if err != nil {
		t.Fatal(err)
	}
	dropped, err := tape.Dropout(tracked.Value(), probability)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := tape.ReduceMean(dropped, []int{0}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	gradient, err := tape.Grad(tracked.Value())
	if err != nil {
		t.Fatal(err)
	}

	plus, minus := make([]float32, len(values)), make([]float32, len(values))
	copy(plus, values)
	copy(minus, values)
	const h = float32(1e-3)
	for index := range values {
		plus[index] += h
		minus[index] -= h
		plusLoss := dropoutFrozenMaskLoss(t, plus, seed, probability)
		minusLoss := dropoutFrozenMaskLoss(t, minus, seed, probability)
		finite := (plusLoss - minusLoss) / (2 * h)
		if math.Abs(float64(gradient.data[index]-finite)) > 2e-3 {
			t.Fatalf("gradient[%d] = %g, finite difference = %g", index, gradient.data[index], finite)
		}
		plus[index] = values[index]
		minus[index] = values[index]
	}

	keepScale := float32(1 / (1 - probability))
	for index, value := range dropped.data {
		want := float32(0)
		if value != 0 {
			want = keepScale / float32(len(values))
		}
		if gradient.data[index] != want {
			t.Fatalf("gradient[%d] = %g, want %g", index, gradient.data[index], want)
		}
	}

	identity := inputDataCopy(values)
	if len(identity) != len(values) {
		t.Fatal("identity copy changed shape")
	}
	for index := range values {
		if identity[index] != values[index] {
			t.Fatalf("eval identity[%d] = %g, want %g", index, identity[index], values[index])
		}
	}
}

func dropoutFrozenMaskLoss(t *testing.T, values []float32, seed int64, probability float32) float32 {
	t.Helper()
	input := mustTestTensor(t, []int{len(values)}, values)
	tape := NewTape(seed)
	dropped, err := tape.Dropout(input, probability)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := tape.ReduceMean(dropped, []int{0}, false)
	if err != nil {
		t.Fatal(err)
	}
	return loss.data[0]
}

func inputDataCopy(values []float32) []float32 {
	return append([]float32(nil), values...)
}

func TestAdamWHandComputed(t *testing.T) {
	parameter := mustTestTensor(t, []int{1}, []float32{2})
	tape := NewTape()
	tracked, err := tape.Param(parameter)
	if err != nil {
		t.Fatal(err)
	}
	loss, err := tape.ReduceMean(tracked.Value(), []int{0}, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	if err := tape.AdamW(0.1, 0.2); err != nil {
		t.Fatal(err)
	}
	if got, want := parameter.data[0], float32(1.86); math.Abs(float64(got-want)) > 1e-6 {
		t.Fatalf("AdamW parameter = %g, want %g", got, want)
	}
}

func TestStepLR(t *testing.T) {
	schedule, err := NewStepLR(1e-3, 0.5, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []float32{1e-3, 1e-3, 5e-4, 5e-4, 2.5e-4}
	for step, expected := range want {
		if got := schedule.LR(step); got != expected {
			t.Fatalf("step %d lr = %g, want %g", step, got, expected)
		}
	}
}
