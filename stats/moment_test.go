package stats_test

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

func TestCalculateMoment(t *testing.T) {
	data := []float64{2.0, 4.0, 7.0, 1.0, 8.0, 3.0, 9.0, 2.0}
	dl := insyra.NewDataList(data)
	tol := 1e-6

	rawExpect := []float64{
		4.5,     // Raw 1st moment (mean)
		28.5,    // Raw 2nd
		211.5,   // Raw 3rd
		1678.5,  // Raw 4th
		13744.5, // Raw 5th
	}

	centralExpect := []float64{
		0.0,      // Central 1st
		8.25,     // Central 2nd (variance)
		9.0,      // Central 3rd
		104.0625, // Central 4th
		217.5,    // Central 5th
	}

	for i := 1; i <= 5; i++ {
		raw, err := stats.CalculateMoment(dl, i, false)
		if err != nil {
			t.Fatalf("CalculateMoment raw order %d error: %v", i, err)
		}
		if !almostEqual(raw, rawExpect[i-1], tol) {
			t.Errorf("Raw moment %d = %f; want %f", i, raw, rawExpect[i-1])
		}

		central, err := stats.CalculateMoment(dl, i, true)
		if err != nil {
			t.Fatalf("CalculateMoment central order %d error: %v", i, err)
		}
		if !almostEqual(central, centralExpect[i-1], tol) {
			t.Errorf("Central moment %d = %f; want %f", i, central, centralExpect[i-1])
		}
	}
}

func TestCalculateMoment_EdgeCases(t *testing.T) {
	empty := insyra.NewDataList([]float64{})
	if _, err := stats.CalculateMoment(empty, 2, true); err == nil {
		t.Error("Expected error for empty input")
	}
	m, _ := stats.CalculateMoment(empty, 2, true)
	if !math.IsNaN(m) {
		t.Error("Expected NaN for empty input")
	}

	data := insyra.NewDataList([]float64{1.0, 2.0})
	if _, err := stats.CalculateMoment(data, 0, true); err == nil {
		t.Error("Expected error for moment order 0")
	}
	m, _ = stats.CalculateMoment(data, 0, true)
	if !math.IsNaN(m) {
		t.Error("Expected NaN for moment order 0")
	}
}
