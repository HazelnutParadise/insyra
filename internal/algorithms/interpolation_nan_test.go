package algorithms

import (
	"errors"
	"math"
	"testing"
)

// IN-21 of #340: NearestNeighborInterpolation answered data[0] for a NaN x,
// because every comparison against NaN is false so the search never moved off
// index 0. The other interpolations refuse a NaN, so the same call was an error
// in one place and a plausible-looking number in another.
func TestInterpolation_NaNIsRefusedEverywhere(t *testing.T) {
	data := []float64{10, 20, 30, 40}

	interpolations := map[string]func([]float64, float64) (float64, error){
		"NearestNeighbor": NearestNeighborInterpolation,
		"Linear":          LinearInterpolation,
		"Quadratic":       QuadraticInterpolation,
		"Lagrange":        LagrangeInterpolation,
		"Newton":          NewtonInterpolation,
	}
	for name, fn := range interpolations {
		t.Run(name, func(t *testing.T) {
			got, err := fn(data, math.NaN())
			if err == nil {
				t.Errorf("a NaN x gave %v with no error", got)
			}
			if !errors.Is(err, ErrOutOfBounds) {
				t.Errorf("a NaN x gave %v, want ErrOutOfBounds", err)
			}
		})
	}
}

// The ordinary answers are unchanged.
func TestNearestNeighborInterpolation_StillWorks(t *testing.T) {
	data := []float64{10, 20, 30, 40}

	tests := []struct {
		x    float64
		want float64
	}{
		{x: 0, want: 10},
		{x: 0.4, want: 10},
		{x: 0.6, want: 20},
		{x: 2, want: 30},
		{x: 3, want: 40},
	}
	for _, tt := range tests {
		got, err := NearestNeighborInterpolation(data, tt.x)
		if err != nil {
			t.Errorf("x=%v: %v", tt.x, err)
			continue
		}
		if got != tt.want {
			t.Errorf("x=%v: got %v, want %v", tt.x, got, tt.want)
		}
	}
}
