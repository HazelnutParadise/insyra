package nn

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

// #266 (NN-2): a training run can be stopped between batches.

func fitContextData(t *testing.T) (*Tensor, *Tensor) {
	t.Helper()
	input := mustTestTensor(t, []int{8, 2}, []float32{
		-2, -1, -1, -2, -2, -2, 1, 2, 2, 1, 1, 1, 2, 2, -1, -1,
	})
	target := mustTestInt64Tensor(t, []int{8}, []int64{0, 0, 0, 1, 1, 1, 1, 0})
	return input, target
}

func fitContextModel(t *testing.T, layers ...Layer) *Sequential {
	t.Helper()
	if len(layers) == 0 {
		layers = []Layer{Dense(2, 4), ReLU(), Dense(4, 2)}
	}
	model, err := NewSequential(NewTape(123), layers...)
	if err != nil {
		t.Fatal(err)
	}
	return model
}

func parameterBits(model *Sequential) [][]uint32 {
	var bits [][]uint32
	for _, parameter := range model.Parameters() {
		values := parameter.Value().data
		row := make([]uint32, len(values))
		for i, value := range values {
			row[i] = math.Float32bits(value)
		}
		bits = append(bits, row)
	}
	return bits
}

func TestFitContextStopsAfterAnEpoch(t *testing.T) {
	input, target := fitContextData(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config := FitConfig{
		Epochs: 3, BatchSize: 3, Seed: 77, Optimizer: Adam{Rate: 0.01}, Loss: CrossEntropy{}, Quiet: true,
		Progress: func(epoch FitEpoch) {
			if epoch.Epoch == 1 {
				cancel()
			}
		},
	}
	stopped := fitContextModel(t)
	result, err := stopped.FitContext(ctx, input, target, config)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if result == nil || len(result.Epochs) != 1 || len(result.TrainLosses) != 1 {
		t.Fatalf("result = %+v, want exactly the one finished epoch", result)
	}

	config.Epochs = 1
	config.Progress = nil
	oneEpoch := fitContextModel(t)
	want, err := oneEpoch.Fit(input, target, config)
	if err != nil {
		t.Fatal(err)
	}
	if result.TrainLosses[0] != want.TrainLosses[0] {
		t.Errorf("stopped run's epoch loss = %.17g, one-epoch run's = %.17g", result.TrainLosses[0], want.TrainLosses[0])
	}
	if !reflect.DeepEqual(parameterBits(stopped), parameterBits(oneEpoch)) {
		t.Error("parameters after stopping at epoch 1 differ from a one-epoch run's")
	}
}

func TestFitContextStopsInsideAnEpoch(t *testing.T) {
	input, target := fitContextData(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forwards := 0
	counter := Func(func(_ *Tape, x *Tensor) (*Tensor, error) {
		forwards++
		if forwards == 2 {
			cancel()
		}
		return x, nil
	})
	model := fitContextModel(t, Dense(2, 4), counter, ReLU(), Dense(4, 2))
	before := parameterBits(model)
	progressCalls := 0
	config := FitConfig{
		Epochs: 2, BatchSize: 3, Seed: 77, Optimizer: SGD{Rate: 0.1}, Loss: CrossEntropy{}, Quiet: true,
		Progress: func(FitEpoch) { progressCalls++ },
	}
	result, err := model.FitContext(ctx, input, target, config)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if forwards != 2 {
		t.Errorf("forward passes = %d, want 2: the batch after the cancellation must not run", forwards)
	}
	if result == nil || len(result.Epochs) != 0 || progressCalls != 0 {
		t.Errorf("result = %+v with %d Progress calls, want no finished epoch", result, progressCalls)
	}
	if reflect.DeepEqual(parameterBits(model), before) {
		t.Error("parameters unchanged, but the two batches before the cancellation took optimizer steps")
	}
}

func TestFitContextAlreadyDoneChangesNothing(t *testing.T) {
	input, target := fitContextData(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	model := fitContextModel(t)
	before := parameterBits(model)
	rngBefore := model.tape.rng
	progressCalls := 0
	result, err := model.FitContext(ctx, input, target, FitConfig{
		Epochs: 2, BatchSize: 3, Seed: 1, Optimizer: SGD{Rate: 0.1}, Loss: CrossEntropy{}, Quiet: true,
		Progress: func(FitEpoch) { progressCalls++ },
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if result == nil || len(result.Epochs) != 0 || progressCalls != 0 {
		t.Errorf("result = %+v with %d Progress calls, want an empty result", result, progressCalls)
	}
	if !reflect.DeepEqual(parameterBits(model), before) || model.tape.rng != rngBefore {
		t.Error("a context that was already done changed the model or its tape")
	}
}

func TestFitContextReportsAnExpiredDeadline(t *testing.T) {
	input, target := fitContextData(t)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	_, err := fitContextModel(t).FitContext(ctx, input, target, FitConfig{
		Epochs: 1, BatchSize: 3, Optimizer: SGD{Rate: 0.1}, Loss: CrossEntropy{}, Quiet: true,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestFitContextRefusesANilContext(t *testing.T) {
	input, target := fitContextData(t)
	var ctx context.Context
	model := fitContextModel(t)
	before := parameterBits(model)
	result, err := model.FitContext(ctx, input, target, FitConfig{
		Epochs: 1, BatchSize: 3, Optimizer: SGD{Rate: 0.1}, Loss: CrossEntropy{}, Quiet: true,
	})
	if err == nil || !strings.Contains(err.Error(), "context") || result != nil {
		t.Fatalf("FitContext(nil ctx) = %v, %v; want a nil result and an error about the context", result, err)
	}
	if !reflect.DeepEqual(parameterBits(model), before) {
		t.Error("a nil context changed the model")
	}
}

// The check after an epoch's last batch stops the run before that epoch's
// validation and Progress call, so the epoch is not reported.
func TestFitContextStopsAfterAnEpochsLastBatch(t *testing.T) {
	input, target := fitContextData(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forwards := 0
	counter := Func(func(_ *Tape, x *Tensor) (*Tensor, error) {
		forwards++
		if forwards == 6 { // the last of epoch 2's three batches
			cancel()
		}
		return x, nil
	})
	model := fitContextModel(t, Dense(2, 4), counter, ReLU(), Dense(4, 2))
	progressCalls := 0
	result, err := model.FitContext(ctx, input, target, FitConfig{
		Epochs: 3, BatchSize: 3, Seed: 77, Optimizer: SGD{Rate: 0.1}, Loss: CrossEntropy{}, Quiet: true,
		Progress: func(FitEpoch) { progressCalls++ },
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if forwards != 6 || result == nil || len(result.Epochs) != 1 || progressCalls != 1 {
		t.Fatalf("forwards = %d, result = %+v, Progress calls = %d; want 6 forward passes and only epoch 1 reported", forwards, result, progressCalls)
	}
}

// A run whose every epoch finished has nothing left to stop: cancelling in
// the last epoch's Progress call returns the whole result and no error.
func TestFitContextFinishedRunIgnoresALateCancel(t *testing.T) {
	input, target := fitContextData(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result, err := fitContextModel(t).FitContext(ctx, input, target, FitConfig{
		Epochs: 2, BatchSize: 3, Seed: 77, Optimizer: SGD{Rate: 0.1}, Loss: CrossEntropy{}, Quiet: true,
		Progress: func(epoch FitEpoch) {
			if epoch.Epoch == 2 {
				cancel()
			}
		},
	})
	if err != nil || result == nil || len(result.Epochs) != 2 {
		t.Fatalf("FitContext = %+v, %v; want both epochs and no error", result, err)
	}
}
