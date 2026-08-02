package dl

import (
	"testing"
)

func TestCNNVJPsFiniteDifference(t *testing.T) {
	convOptions := ConvOptions{
		Pads:    []int{1, 0, 2, 1},
		Strides: []int{2, 1},
		Group:   2,
	}
	convInput := mustTestTensor(t, []int{1, 2, 3, 4}, patternedTestData(24))
	convWeights := mustTestTensor(t, []int{4, 1, 2, 3}, patternedTestData(24))
	convBias := mustTestTensor(t, []int{4}, []float32{0.2, -0.3, 0.4, -0.5})
	for _, test := range []struct {
		name      string
		parameter *Tensor
		build     func(*testing.T, *Tensor) (*Tape, *Tensor)
	}{
		{
			name:      "conv input grouped asymmetric padded strided",
			parameter: convInput,
			build: func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				input = mustTapeParameter(t, tape, input)
				output, err := tape.Conv(input, convWeights, convBias, convOptions)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:      "conv weights grouped asymmetric padded strided",
			parameter: convWeights,
			build: func(t *testing.T, weights *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				weights = mustTapeParameter(t, tape, weights)
				output, err := tape.Conv(convInput, weights, convBias, convOptions)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:      "conv bias grouped asymmetric padded strided",
			parameter: convBias,
			build: func(t *testing.T, bias *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				bias = mustTapeParameter(t, tape, bias)
				output, err := tape.Conv(convInput, convWeights, bias, convOptions)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			checkFiniteDifference(t, test.parameter, test.build)
		})
	}

	poolInput := mustTestTensor(t, []int{1, 1, 3, 4}, patternedTestData(12))
	t.Run("max pool", func(t *testing.T) {
		checkFiniteDifference(t, poolInput, func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
			tape := NewTape()
			input = mustTapeParameter(t, tape, input)
			output, err := tape.MaxPool(input, []int{2, 3}, PoolOptions{
				Pads:    []int{1, 1, 0, 2},
				Strides: []int{1, 2},
			})
			if err != nil {
				t.Fatal(err)
			}
			return tape, meanLoss(t, tape, output)
		})
	})

	for _, countIncludePad := range []bool{false, true} {
		t.Run("average pool count_include_pad="+boolString(countIncludePad), func(t *testing.T) {
			checkFiniteDifference(t, poolInput, func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				input = mustTapeParameter(t, tape, input)
				output, err := tape.AveragePool(input, []int{2, 3}, PoolOptions{
					Pads:            []int{1, 1, 0, 2},
					Strides:         []int{1, 2},
					CountIncludePad: countIncludePad,
				})
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			})
		})
	}

	t.Run("global average pool", func(t *testing.T) {
		checkFiniteDifference(t, poolInput, func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
			tape := NewTape()
			input = mustTapeParameter(t, tape, input)
			output, err := tape.GlobalAveragePool(input)
			if err != nil {
				t.Fatal(err)
			}
			return tape, meanLoss(t, tape, output)
		})
	})

	bnInput := mustTestTensor(t, []int{2, 2, 2, 2}, patternedTestData(16))
	bnScale := mustTestTensor(t, []int{2}, []float32{1.2, 0.7})
	bnBias := mustTestTensor(t, []int{2}, []float32{-0.2, 0.4})
	bnMean := mustTestTensor(t, []int{2}, []float32{0.1, -0.3})
	bnVariance := mustTestTensor(t, []int{2}, []float32{1.4, 0.8})
	for _, test := range []struct {
		name      string
		parameter *Tensor
		build     func(*testing.T, *Tensor) (*Tape, *Tensor)
	}{
		{
			name:      "batch normalization input",
			parameter: bnInput,
			build: func(t *testing.T, input *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				input = mustTapeParameter(t, tape, input)
				output, err := tape.BatchNormalization(input, bnScale, bnBias, bnMean, bnVariance, 1e-4)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:      "batch normalization scale",
			parameter: bnScale,
			build: func(t *testing.T, scale *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				scale = mustTapeParameter(t, tape, scale)
				output, err := tape.BatchNormalization(bnInput, scale, bnBias, bnMean, bnVariance, 1e-4)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
		{
			name:      "batch normalization bias",
			parameter: bnBias,
			build: func(t *testing.T, bias *Tensor) (*Tape, *Tensor) {
				tape := NewTape()
				bias = mustTapeParameter(t, tape, bias)
				output, err := tape.BatchNormalization(bnInput, bnScale, bias, bnMean, bnVariance, 1e-4)
				if err != nil {
					t.Fatal(err)
				}
				return tape, meanLoss(t, tape, output)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			checkFiniteDifference(t, test.parameter, test.build)
		})
	}
}

func TestMaxPoolVJPUsesFirstMaximum(t *testing.T) {
	input := mustTestTensor(t, []int{1, 1, 2, 2}, []float32{3, 3, 3, 3})
	tape := NewTape()
	parameter, err := tape.Param(input)
	if err != nil {
		t.Fatalf("Param: %v", err)
	}
	output, err := tape.MaxPool(input, []int{2, 2})
	if err != nil {
		t.Fatalf("MaxPool: %v", err)
	}
	loss := meanLoss(t, tape, output)
	if err := tape.Backward(loss); err != nil {
		t.Fatalf("Backward: %v", err)
	}
	assertAutodiffValue(t, "first maximum gradient", parameter.Grad().data, []float32{1, 0, 0, 0})
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
