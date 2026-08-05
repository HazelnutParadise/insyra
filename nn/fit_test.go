package nn

import (
	"bytes"
	"log"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	insyra "github.com/HazelnutParadise/insyra"
)

func TestSequentialFitMatchesHandWrittenLoop(t *testing.T) {
	const seed = int64(77)
	input := mustTestTensor(t, []int{9, 2}, []float32{
		-2, -1, -1, -2, -2, -2, 1, 2, 2, 1, 1, 1, 2, 2, 1, 2, -1, -1,
	})
	target := mustTestInt64Tensor(t, []int{9}, []int64{0, 0, 0, 1, 1, 1, 1, 0, 0})
	config := FitConfig{Epochs: 4, BatchSize: 3, Seed: seed, Optimizer: Adam{Rate: 0.01}, Loss: CrossEntropy{}, Quiet: true}

	fitModel, err := NewSequential(NewTape(123), Dense(2, 4), ReLU(), Dense(4, 2))
	if err != nil {
		t.Fatal(err)
	}
	fitResult, err := fitModel.Fit(input, target, config)
	if err != nil {
		t.Fatal(err)
	}

	handModel, err := NewSequential(NewTape(123), Dense(2, 4), ReLU(), Dense(4, 2))
	if err != nil {
		t.Fatal(err)
	}
	handTape := handModel.tape
	rng := rand.New(rand.NewSource(seed))
	handLosses := make([]float64, 0, config.Epochs)
	for epoch := 0; epoch < config.Epochs; epoch++ {
		order := rng.Perm(input.shape[0])
		var total float64
		batches := 0
		for start := 0; start < len(order); start += config.BatchSize {
			end := start + config.BatchSize
			if end > len(order) {
				end = len(order)
			}
			xBatch := testBatch(t, input, order[start:end])
			yBatch := testBatch(t, target, order[start:end])
			handTape.ops = nil
			handTape.grads = make(map[*Tensor]*Tensor)
			prediction, err := handModel.Forward(handTape, xBatch)
			if err != nil {
				t.Fatal(err)
			}
			loss, err := handTape.SoftmaxCrossEntropy(prediction, yBatch)
			if err != nil {
				t.Fatal(err)
			}
			if err := handTape.Backward(loss); err != nil {
				t.Fatal(err)
			}
			if err := handTape.Adam(0.01); err != nil {
				t.Fatal(err)
			}
			total += float64(loss.data[0])
			batches++
		}
		handLosses = append(handLosses, total/float64(batches))
	}

	if !reflect.DeepEqual(fitResult.TrainLosses, handLosses) {
		t.Fatalf("Fit losses = %.17g, hand losses = %.17g", fitResult.TrainLosses, handLosses)
	}
}

func TestSequentialFitDeterminismNoShuffleAndRefusals(t *testing.T) {
	input := mustTestTensor(t, []int{4, 1}, []float32{0, 1, 2, 3})
	target := mustTestTensor(t, []int{4, 1}, []float32{0, 1, 2, 3})
	config := FitConfig{Epochs: 3, BatchSize: 2, Seed: 9, Optimizer: SGDMomentum{Rate: 0.01, Momentum: 0.9}, Loss: MSE{}, Quiet: true}
	first, err := NewSequential(NewTape(44), Dense(1, 1))
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewSequential(NewTape(44), Dense(1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.Fit(input, target, config); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Fit(input, target, config); err != nil {
		t.Fatal(err)
	}
	for name, parameter := range first.NamedParameters() {
		if !sameFloat32(parameter.Value().data, second.NamedParameters()[name].Value().data) {
			t.Fatalf("parameter %s differs between identical Fit runs", name)
		}
	}

	var firstRows []float32
	model, err := NewSequential(NewTape(), Func(func(_ *Tape, x *Tensor) (*Tensor, error) {
		firstRows = append(firstRows, x.data[0])
		return x, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	noShuffle := config
	noShuffle.NoShuffle = true
	noShuffle.Epochs = 1
	noShuffle.BatchSize = 2
	if _, err := model.Fit(input, target, noShuffle); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstRows, []float32{0, 2}) {
		t.Fatalf("NoShuffle batch order = %v, want [0 2]", firstRows)
	}

	missingOptimizer := config
	missingOptimizer.Optimizer = nil
	if _, err := first.Fit(input, target, missingOptimizer); err == nil || !strings.Contains(err.Error(), "Optimizer") {
		t.Fatalf("missing optimizer error = %v", err)
	}
	missingLoss := config
	missingLoss.Loss = nil
	if _, err := first.Fit(input, target, missingLoss); err == nil || !strings.Contains(err.Error(), "Loss") {
		t.Fatalf("missing loss error = %v", err)
	}
	badEpochs := config
	badEpochs.Epochs = 0
	if _, err := first.Fit(input, target, badEpochs); err == nil || !strings.Contains(err.Error(), "Epochs") {
		t.Fatalf("invalid epochs error = %v", err)
	}
}

func TestSequentialFitProgressQuietValidationAndTrainingOnly(t *testing.T) {
	input := mustTestTensor(t, []int{4, 1}, []float32{0, 1, 2, 3})
	target := mustTestTensor(t, []int{4, 1}, []float32{0, 1, 2, 3})
	previousWriter := log.Writer()
	previousLevel := insyra.Config.GetLogLevel()
	defer func() {
		log.SetOutput(previousWriter)
		insyra.Config.SetLogLevel(previousLevel)
	}()
	insyra.Config.SetLogLevel(insyra.LogLevelInfo)
	var output bytes.Buffer
	log.SetOutput(&output)
	var progress []FitEpoch
	model, err := NewSequential(NewTape(10), Dense(1, 1), Dropout(0.5))
	if err != nil {
		t.Fatal(err)
	}
	config := FitConfig{
		Epochs: 2, BatchSize: 2, Seed: 5, Optimizer: SGD{Rate: 0.01}, Loss: MSE{},
		ValX: input, ValY: target, Progress: func(epoch FitEpoch) { progress = append(progress, epoch) },
	}
	result, err := model.Fit(input, target, config)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(output.String(), "Sequential.Fit:"); lines != 2 {
		t.Fatalf("default progress lines = %d, want 2: %s", lines, output.String())
	}
	if len(progress) != 2 || !progress[0].HasValLoss || len(result.ValLosses) != 2 {
		t.Fatalf("progress/validation = %#v, result validation = %v", progress, result.ValLosses)
	}
	prediction, err := model.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	validationTape := NewTape()
	wantLoss, err := validationTape.MSELoss(prediction, target)
	if err != nil {
		t.Fatal(err)
	}
	if result.ValLosses[len(result.ValLosses)-1] != float64(wantLoss.data[0]) {
		t.Fatalf("validation loss = %g, Predict loss = %g", result.ValLosses[len(result.ValLosses)-1], wantLoss.data[0])
	}

	output.Reset()
	progress = nil
	quiet := config
	quiet.Quiet = true
	if _, err := model.Fit(input, target, quiet); err != nil {
		t.Fatal(err)
	}
	if output.Len() != 0 || len(progress) != 2 {
		t.Fatalf("Quiet output/progress = %q/%d", output.String(), len(progress))
	}
}

func TestSequentialFitMicroConvergence(t *testing.T) {
	rng := rand.New(rand.NewSource(20260803))
	const sampleCount = 64
	features := make([]float32, sampleCount*2)
	labels := make([]int64, sampleCount)
	for row := 0; row < sampleCount; row++ {
		center := float64(-2)
		if row >= sampleCount/2 {
			center = 2
			labels[row] = 1
		}
		features[2*row] = float32(center + 0.35*rng.NormFloat64())
		features[2*row+1] = float32(center + 0.35*rng.NormFloat64())
	}
	input := mustTestTensor(t, []int{sampleCount, 2}, features)
	target := mustTestInt64Tensor(t, []int{sampleCount}, labels)
	model, err := NewSequential(NewTape(20260803), Dense(2, 8), ReLU(), Dense(8, 2))
	if err != nil {
		t.Fatal(err)
	}
	result, err := model.Fit(input, target, FitConfig{
		Epochs: 200, BatchSize: sampleCount, Seed: 20260803, NoShuffle: true,
		Optimizer: Adam{Rate: 0.01}, Loss: CrossEntropy{}, Quiet: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	prediction, err := model.Predict(input)
	if err != nil {
		t.Fatal(err)
	}
	correct := 0
	for row := range labels {
		best := 0
		for class := 1; class < prediction.shape[1]; class++ {
			if prediction.data[row*prediction.shape[1]+class] > prediction.data[row*prediction.shape[1]+best] {
				best = class
			}
		}
		if int64(best) == labels[row] {
			correct++
		}
	}
	if correct != sampleCount {
		t.Fatalf("Fit micro convergence accuracy = %d/%d after %d epochs, final loss=%g", correct, sampleCount, len(result.TrainLosses), result.TrainLosses[len(result.TrainLosses)-1])
	}
}

func testBatch(t *testing.T, input *Tensor, rows []int) *Tensor {
	t.Helper()
	batch, err := fitBatch(input, rows)
	if err != nil {
		t.Fatal(err)
	}
	return batch
}
