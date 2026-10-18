package nn

import (
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// #266 (NN-2): Fit trains with an optimizer the caller writes, and a parameter
// can be updated from its gradient outside the package.

func customOptimizerRun(t *testing.T, optimizer OptimizerSpec) (*Sequential, *FitResult, error) {
	t.Helper()
	x, y, valX, valY := customLossData(t)
	model, err := NewSequential(NewTape(21), Dense(2, 3), ReLU(), Dense(3, 1))
	if err != nil {
		t.Fatal(err)
	}
	result, err := model.Fit(x, y, FitConfig{
		Epochs: 3, BatchSize: 4, Seed: 21, Optimizer: optimizer, Loss: MSE{}, Quiet: true,
		ValX: valX, ValY: valY,
	})
	return model, result, err
}

// customSGD is plain SGD written only with the public API.
func customSGD(rate float32) CustomOptimizer {
	return CustomOptimizer{
		Name: "my-sgd",
		Step: func(params []*Parameter) error {
			for _, p := range params {
				if err := p.Update(func(value, grad []float32) {
					if grad == nil {
						return
					}
					for i := range value {
						value[i] -= rate * grad[i]
					}
				}); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func TestCustomOptimizerMatchesTheBuiltInItReimplements(t *testing.T) {
	builtInModel, builtIn, err := customOptimizerRun(t, SGD{Rate: 0.05})
	if err != nil {
		t.Fatal(err)
	}
	customModel, custom, err := customOptimizerRun(t, customSGD(0.05))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(custom.TrainLosses, builtIn.TrainLosses) || !slices.Equal(custom.ValLosses, builtIn.ValLosses) {
		t.Fatalf("custom losses %v / %v, built-in %v / %v", custom.TrainLosses, custom.ValLosses, builtIn.TrainLosses, builtIn.ValLosses)
	}
	if !reflect.DeepEqual(parameterBits(customModel), parameterBits(builtInModel)) {
		t.Fatal("parameters after a custom SGD differ from nn.SGD's")
	}
}

func TestCustomOptimizerStepSeesEveryParameterOncePerBatch(t *testing.T) {
	x, y, _, _ := customLossData(t)
	model, err := NewSequential(NewTape(21), Dense(2, 3), ReLU(), Dense(3, 1))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	_, err = model.Fit(x, y, FitConfig{
		Epochs: 2, BatchSize: 4, Seed: 21, Loss: MSE{}, Quiet: true,
		Optimizer: CustomOptimizer{Step: func(params []*Parameter) error {
			calls++
			if !slices.Equal(params, model.Parameters()) {
				t.Errorf("Step got %d parameters, want the model's %d in order", len(params), len(model.Parameters()))
			}
			return nil
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// 6 rows in batches of 4: two batches per epoch.
	if calls != 4 {
		t.Fatalf("Step called %d times, want 4", calls)
	}
}

func TestCustomOptimizerStepSliceDoesNotAliasTheTape(t *testing.T) {
	x, y, _, _ := customLossData(t)
	model, err := NewSequential(NewTape(21), Dense(2, 3), ReLU(), Dense(3, 1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = model.Fit(x, y, FitConfig{
		Epochs: 2, BatchSize: 6, Seed: 21, Loss: MSE{}, Quiet: true,
		Optimizer: CustomOptimizer{Step: func(params []*Parameter) error {
			for i := range params {
				params[i] = nil
			}
			return nil
		}},
	})
	if err != nil {
		t.Fatalf("clearing the slice Step was given broke the next batch: %v", err)
	}
	for _, p := range model.tape.params {
		if p == nil {
			t.Fatal("Step's slice aliases the tape's parameter list")
		}
	}
}

func TestCustomOptimizerErrors(t *testing.T) {
	x, y, _, _ := customLossData(t)
	newModel := func() *Sequential {
		model, err := NewSequential(NewTape(21), Dense(2, 3), ReLU(), Dense(3, 1))
		if err != nil {
			t.Fatal(err)
		}
		return model
	}

	t.Run("missing Step is refused before any batch", func(t *testing.T) {
		model := newModel()
		before := parameterBits(model)
		_, err := model.Fit(x, y, FitConfig{Epochs: 1, BatchSize: 6, Loss: MSE{}, Quiet: true, Optimizer: CustomOptimizer{Name: "lion"}})
		if err == nil || !strings.Contains(err.Error(), "lion") || !strings.Contains(err.Error(), "Step") {
			t.Fatalf("err = %v, want one naming lion and its missing Step", err)
		}
		if !reflect.DeepEqual(parameterBits(model), before) {
			t.Fatal("parameters moved although Fit refused the config")
		}
	})

	t.Run("an empty name reads CustomOptimizer", func(t *testing.T) {
		_, err := newModel().Fit(x, y, FitConfig{Epochs: 1, BatchSize: 6, Loss: MSE{}, Quiet: true, Optimizer: CustomOptimizer{}})
		if err == nil || !strings.Contains(err.Error(), "CustomOptimizer") {
			t.Fatalf("err = %v, want one naming CustomOptimizer", err)
		}
	})

	t.Run("a Step error stops Fit and is wrapped", func(t *testing.T) {
		sentinel := errors.New("diverged")
		_, err := newModel().Fit(x, y, FitConfig{
			Epochs: 1, BatchSize: 6, Loss: MSE{}, Quiet: true,
			Optimizer: CustomOptimizer{Name: "lion", Step: func([]*Parameter) error { return sentinel }},
		})
		if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "lion") {
			t.Fatalf("err = %v, want it to wrap the Step error and name lion", err)
		}
	})
}

func TestParameterUpdateWritesTheValueFromTheGradient(t *testing.T) {
	tape := NewTape()
	weight := mustTestTensor(t, []int{2}, []float32{1, 2})
	other := mustTestTensor(t, []int{2}, []float32{5, 6})
	p, err := tape.Param(weight)
	if err != nil {
		t.Fatal(err)
	}
	unreached, err := tape.Param(other)
	if err != nil {
		t.Fatal(err)
	}
	target := mustTestTensor(t, []int{2}, []float32{0, 0})
	loss, err := tape.MSELoss(weight, target)
	if err != nil {
		t.Fatal(err)
	}
	if err := tape.Backward(loss); err != nil {
		t.Fatal(err)
	}
	want, err := tape.Grad(weight)
	if err != nil {
		t.Fatal(err)
	}
	wantGrad, _ := want.Float32Data()

	if err := p.Update(func(value, grad []float32) {
		if !slices.Equal(grad, wantGrad) {
			t.Errorf("grad = %v, want %v", grad, wantGrad)
		}
		for i := range value {
			value[i] -= 0.5 * grad[i]
		}
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := p.Value().Float32Data()
	if wantValue := []float32{1 - 0.5*wantGrad[0], 2 - 0.5*wantGrad[1]}; !slices.Equal(got, wantValue) {
		t.Fatalf("value after Update = %v, want %v", got, wantValue)
	}

	// A parameter the loss never reached has no gradient: grad is nil.
	sawNil := false
	if err := unreached.Update(func(value, grad []float32) { sawNil = grad == nil }); err != nil {
		t.Fatal(err)
	}
	if !sawNil {
		t.Fatal("a parameter without a gradient got a non-nil grad")
	}

	// The next forward pass reads the updated value.
	loss, err = tape.MSELoss(weight, target)
	if err != nil {
		t.Fatal(err)
	}
	lossData, _ := loss.Float32Data()
	if wantLoss := (got[0]*got[0] + got[1]*got[1]) / 2; math.Abs(float64(lossData[0]-wantLoss)) > 1e-6 {
		t.Fatalf("loss after Update = %v, want %v", lossData[0], wantLoss)
	}
}

func TestParameterUpdateRefusals(t *testing.T) {
	var nilParam *Parameter
	if err := nilParam.Update(func(value, grad []float32) {}); err == nil {
		t.Fatal("Update on a nil parameter returned no error")
	}
	p, err := NewTape().Param(mustTestTensor(t, []int{1}, []float32{1}))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Update(nil); err == nil {
		t.Fatal("Update with a nil function returned no error")
	}
}
