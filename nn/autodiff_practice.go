package nn

import (
	"fmt"
	"math"
	"math/rand"
)

// Dropout applies training-time inverted dropout. The mask comes from the
// tape-owned seeded RNG and is retained by the VJP. Eval behavior is the
// identity by convention: callers simply do not call Dropout during eval.
func (t *Tape) Dropout(input *Tensor, probability float32) (*Tensor, error) {
	if err := requireFloat32(input, "autodiff dropout input"); err != nil {
		return nil, err
	}
	if math.IsNaN(float64(probability)) || math.IsInf(float64(probability), 0) || probability < 0 || probability >= 1 {
		return nil, fmt.Errorf("dropout probability must be in [0, 1), got %g", probability)
	}
	if t.rng == nil {
		t.rng = newDefaultTapeRNG()
	}
	keepScale := float32(1 / (1 - probability))
	mask := make([]bool, len(input.data))
	values := make([]float32, len(input.data))
	for index, value := range input.data {
		kept := probability == 0 || t.rng.Float32() >= probability
		mask[index] = kept
		if kept {
			values[index] = value * keepScale
		}
	}
	output, err := newFloat32Tensor(input.shape, values)
	if err != nil {
		return nil, err
	}
	t.record("Dropout", []*Tensor{input}, output, func(upstream *Tensor) ([]*Tensor, error) {
		if err := requireFloat32(upstream, "dropout upstream"); err != nil {
			return nil, err
		}
		gradient := make([]float32, len(upstream.data))
		for index, value := range upstream.data {
			if mask[index] {
				gradient[index] = value * keepScale
			}
		}
		result, err := newFloat32Tensor(input.shape, gradient)
		return []*Tensor{result}, err
	})
	return output, nil
}

func newDefaultTapeRNG() *rand.Rand {
	return rand.New(rand.NewSource(1))
}

// StepLR decays an initial learning rate by gamma every stepSize steps.
// LR(0) is the initial rate; decay applies at stepSize, 2*stepSize, ... .
type StepLR struct {
	initialRate float32
	gamma       float32
	stepSize    int
}

// NewStepLR creates a StepLR schedule with PyTorch-compatible step numbering.
func NewStepLR(initialRate, gamma float32, stepSize int) (*StepLR, error) {
	if math.IsNaN(float64(initialRate)) || math.IsInf(float64(initialRate), 0) || initialRate < 0 {
		return nil, fmt.Errorf("step lr initial rate must be finite and non-negative")
	}
	if math.IsNaN(float64(gamma)) || math.IsInf(float64(gamma), 0) || gamma < 0 {
		return nil, fmt.Errorf("step lr gamma must be finite and non-negative")
	}
	if stepSize <= 0 {
		return nil, fmt.Errorf("step lr step size must be positive")
	}
	return &StepLR{initialRate: initialRate, gamma: gamma, stepSize: stepSize}, nil
}

// LR returns the learning rate for a zero-based optimizer step.
func (s *StepLR) LR(step int) float32 {
	if s == nil || step <= 0 {
		if s == nil {
			return 0
		}
		return s.initialRate
	}
	decays := step / s.stepSize
	return s.initialRate * float32(math.Pow(float64(s.gamma), float64(decays)))
}
