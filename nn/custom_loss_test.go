package nn

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// #266 (NN-2): Fit trains with a loss the caller computes on the tape.

func customLossData(t *testing.T) (x, y, valX, valY *Tensor) {
	t.Helper()
	x = mustTestTensor(t, []int{6, 2}, []float32{0, 1, 1, 0, 1, 1, 0, 0, 2, 1, 1, 2})
	y = mustTestTensor(t, []int{6, 1}, []float32{1, 1, 2, 0, 3, 3})
	valX = mustTestTensor(t, []int{2, 2}, []float32{2, 2, 0, 1})
	valY = mustTestTensor(t, []int{2, 1}, []float32{4, 1})
	return x, y, valX, valY
}

func customLossRun(t *testing.T, loss LossSpec) (*FitResult, error) {
	t.Helper()
	x, y, valX, valY := customLossData(t)
	model, err := NewSequential(NewTape(21), Dense(2, 3), ReLU(), Dense(3, 1))
	if err != nil {
		t.Fatal(err)
	}
	return model.Fit(x, y, FitConfig{
		Epochs: 3, BatchSize: 4, Seed: 21, Optimizer: Adam{Rate: 0.05}, Loss: loss, Quiet: true,
		ValX: valX, ValY: valY,
	})
}

func TestCustomLossMatchesTheBuiltInItWraps(t *testing.T) {
	builtIn, err := customLossRun(t, MSE{})
	if err != nil {
		t.Fatal(err)
	}
	custom, err := customLossRun(t, CustomLoss{
		Name: "wrapped-mse",
		Loss: func(tape *Tape, prediction, target *Tensor) (*Tensor, error) {
			return tape.MSELoss(prediction, target)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(custom.TrainLosses, builtIn.TrainLosses) || !slices.Equal(custom.ValLosses, builtIn.ValLosses) {
		t.Fatalf("custom losses %v / %v, built-in %v / %v", custom.TrainLosses, custom.ValLosses, builtIn.TrainLosses, builtIn.ValLosses)
	}
}

func TestCustomLossComposedFromTapeOperationsTrains(t *testing.T) {
	x, y, _, _ := customLossData(t)
	half := mustTestTensor(t, nil, []float32{0.5})
	model, err := NewSequential(NewTape(21), Dense(2, 3), ReLU(), Dense(3, 1))
	if err != nil {
		t.Fatal(err)
	}
	before := parameterBits(model)
	result, err := model.Fit(x, y, FitConfig{
		Epochs: 20, BatchSize: 6, Seed: 21, Optimizer: Adam{Rate: 0.05}, Quiet: true,
		Loss: CustomLoss{Loss: func(tape *Tape, prediction, target *Tensor) (*Tensor, error) {
			mse, err := tape.MSELoss(prediction, target)
			if err != nil {
				return nil, err
			}
			return tape.Mul(mse, half)
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(parameterBits(model), before) {
		t.Error("parameters did not move: no gradient reached them")
	}
	if first, last := result.TrainLosses[0], result.TrainLosses[len(result.TrainLosses)-1]; !(last < first) {
		t.Errorf("training loss went from %v to %v, want it to fall", first, last)
	}
}

func TestCustomLossValidateRunsOnEveryBatchAndOnValidation(t *testing.T) {
	validations, losses := 0, 0
	_, err := customLossRun(t, CustomLoss{
		Name: "checked",
		Validate: func(prediction, target *Tensor) error {
			validations++
			return nil
		},
		Loss: func(tape *Tape, prediction, target *Tensor) (*Tensor, error) {
			losses++
			return tape.MSELoss(prediction, target)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 6 rows in batches of 4 is 2 batches per epoch, plus one validation per epoch, for 3 epochs.
	if validations != 9 || losses != 9 {
		t.Fatalf("Validate ran %d times and Loss %d times, want 9 each", validations, losses)
	}

	refused := errors.New("targets out of range")
	losses = 0
	_, err = customLossRun(t, CustomLoss{
		Name:     "checked",
		Validate: func(*Tensor, *Tensor) error { return refused },
		Loss: func(tape *Tape, prediction, target *Tensor) (*Tensor, error) {
			losses++
			return tape.MSELoss(prediction, target)
		},
	})
	if !errors.Is(err, refused) || !strings.Contains(err.Error(), "checked") || losses != 0 {
		t.Fatalf("err = %v with %d Loss calls; want the Validate error, named \"checked\", before any Loss call", err, losses)
	}
}

func TestCustomLossRefusesWhatFitCannotTrainOn(t *testing.T) {
	x, y, _, _ := customLossData(t)
	pair := mustTestTensor(t, []int{2}, []float32{1, 2})
	count := mustTestInt64Tensor(t, nil, []int64{1})
	outside := mustTestTensor(t, nil, []float32{0.25})
	returning := func(result *Tensor) func(*Tape, *Tensor, *Tensor) (*Tensor, error) {
		return func(*Tape, *Tensor, *Tensor) (*Tensor, error) { return result, nil }
	}
	cases := []struct {
		name string
		loss CustomLoss
		want string
	}{
		{"no Loss function", CustomLoss{Name: "huber"}, "huber"},
		{"a nil result", CustomLoss{Name: "huber", Loss: returning(nil)}, "huber"},
		{"a result that is not a scalar", CustomLoss{Name: "huber", Loss: returning(pair)}, "huber"},
		{"an int64 result", CustomLoss{Name: "huber", Loss: returning(count)}, "huber"},
		{"a scalar from outside the tape, unnamed", CustomLoss{Loss: returning(outside)}, "CustomLoss"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model, err := NewSequential(NewTape(21), Dense(2, 1))
			if err != nil {
				t.Fatal(err)
			}
			before := parameterBits(model)
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Fit panicked: %v", r)
				}
			}()
			_, err = model.Fit(x, y, FitConfig{Epochs: 1, BatchSize: 6, Seed: 1, Optimizer: SGD{Rate: 0.1}, Loss: tc.loss, Quiet: true})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want an error naming %q", err, tc.want)
			}
			if !reflect.DeepEqual(parameterBits(model), before) {
				t.Error("a refused loss changed the parameters")
			}
		})
	}
}
