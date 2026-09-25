package nn

import (
	"reflect"
	"testing"
)

// Training batch normalization read momentum and epsilon by position, so
// setting epsilon meant writing momentum too, and a reader could not tell
// which number was which. They are named fields now; zero keeps torch's
// default.
func TestBatchNormTrainingTakesNamedOptions(t *testing.T) {
	run := func(opts ...BatchNormOptions) ([]float32, []float32, error) {
		tape := NewTape()
		input, _ := NewTensor([]int{2, 1, 1, 2}, []float32{1, 2, 3, 4})
		scale, _ := NewTensor([]int{1}, []float32{1})
		bias, _ := NewTensor([]int{1}, []float32{0})
		mean, _ := NewTensor([]int{1}, []float32{0})
		variance, _ := NewTensor([]int{1}, []float32{1})
		out, err := tape.BatchNormTraining(input, scale, bias, mean, variance, opts...)
		if err != nil {
			return nil, nil, err
		}
		return out.Data(), mean.Data(), nil
	}

	_, defaultMean, err := run()
	if err != nil {
		t.Fatal(err)
	}
	_, explicitMean, err := run(BatchNormOptions{Momentum: 0.1, Epsilon: 1e-5})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(defaultMean, explicitMean) {
		t.Fatalf("no options %v differs from torch's defaults %v", defaultMean, explicitMean)
	}
	_, fastMean, err := run(BatchNormOptions{Momentum: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(fastMean, defaultMean) {
		t.Fatal("Momentum had no effect on the running mean")
	}
	if _, _, err := run(BatchNormOptions{}, BatchNormOptions{}); err == nil {
		t.Fatal("two options structs were accepted")
	}
}
