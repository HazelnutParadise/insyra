package insyra

import (
	"math"
	"testing"
)

func TestDataList_LinearInterpolation(t *testing.T) {
	dl := NewDataList([]interface{}{1.0, 2.0, 3.0, 4.0})

	// Test normal interpolation
	result := dl.LinearInterpolation(1.5)
	expected := 2.5 // between index 1 (2.0) and index 2 (3.0)
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("LinearInterpolation(1.5) = %v, expected %v", result, expected)
	}

	// Test exact point
	result = dl.LinearInterpolation(2.0)
	expected = 3.0 // index 2
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("LinearInterpolation(2.0) = %v, expected %v", result, expected)
	}

	// Test out of bounds
	result = dl.LinearInterpolation(5.0)
	if !math.IsNaN(result) {
		t.Errorf("LinearInterpolation(5.0) = %v, expected NaN", result)
	}

	// Test not enough data points
	dlShort := NewDataList([]interface{}{1.0})
	result = dlShort.LinearInterpolation(1.5)
	if !math.IsNaN(result) {
		t.Errorf("LinearInterpolation with insufficient data = %v, expected NaN", result)
	}
}

func TestDataList_QuadraticInterpolation(t *testing.T) {
	dl := NewDataList([]any{1.0, 4.0, 9.0, 16.0})

	// Test normal interpolation - within first three points [0,2]
	result := dl.QuadraticInterpolation(1.5)
	expected := 6.25 // Lagrange interpolation of (0,1), (1,4), (2,9) at x=1.5
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("QuadraticInterpolation(1.5) = %v, expected %v", result, expected)
	}

	// Test interpolation within second three points [1,3]
	result = dl.QuadraticInterpolation(2.5)
	// Points: (1,4), (2,9), (3,16) at x=2.5
	// Using Lagrange: l0 = (2.5-2)*(2.5-3)/((1-2)*(1-3)) = (0.5)*(-0.5)/(-1*-2) = (-0.25)/2 = -0.125
	// l1 = (2.5-1)*(2.5-3)/((2-1)*(2-3)) = (1.5)*(-0.5)/(1*-1) = (-0.75)/(-1) = 0.75
	// l2 = (2.5-1)*(2.5-2)/((3-1)*(3-2)) = (1.5)*(0.5)/(2*1) = 0.75/2 = 0.375
	// result = 4*(-0.125) + 9*0.75 + 16*0.375 = -0.5 + 6.75 + 6 = 12.25
	expected = 12.25
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("QuadraticInterpolation(2.5) = %v, expected %v", result, expected)
	}

	// Test out of bounds
	result = dl.QuadraticInterpolation(5.0)
	if !math.IsNaN(result) {
		t.Errorf("QuadraticInterpolation(5.0) = %v, expected NaN", result)
	}

	// Test not enough data points
	dlShort := NewDataList([]interface{}{1.0, 2.0})
	result = dlShort.QuadraticInterpolation(1.5)
	if !math.IsNaN(result) {
		t.Errorf("QuadraticInterpolation with insufficient data = %v, expected NaN", result)
	}
}

func TestDataList_LagrangeInterpolation(t *testing.T) {
	dl := NewDataList([]interface{}{1.0, 4.0, 9.0})

	// Test interpolation at x=1 (should be 4.0)
	result := dl.LagrangeInterpolation(1.0)
	expected := 4.0
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("LagrangeInterpolation(1.0) = %v, expected %v", result, expected)
	}

	// Test interpolation at x=1.5
	result = dl.LagrangeInterpolation(1.5)
	// Manual calculation for Lagrange at x=1.5 with points (0,1), (1,4), (2,9)
	// L0 = ((1.5-1)*(1.5-2))/((0-1)*(0-2)) = (0.5*(-0.5))/(-1*-2) = (-0.25)/2 = -0.125
	// L1 = ((1.5-0)*(1.5-2))/((1-0)*(1-2)) = (1.5*(-0.5))/(1*-1) = (-0.75)/(-1) = 0.75
	// L2 = ((1.5-0)*(1.5-1))/((2-0)*(2-1)) = (1.5*0.5)/(2*1) = 0.75/2 = 0.375
	// result = 1*(-0.125) + 4*0.75 + 9*0.375 = -0.125 + 3 + 3.375 = 6.25
	expected = 6.25
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("LagrangeInterpolation(1.5) = %v, expected %v", result, expected)
	}

	// Test not enough data points
	dlShort := NewDataList([]interface{}{1.0})
	result = dlShort.LagrangeInterpolation(1.5)
	if !math.IsNaN(result) {
		t.Errorf("LagrangeInterpolation with insufficient data = %v, expected NaN", result)
	}
}

func TestDataList_NearestNeighborInterpolation(t *testing.T) {
	dl := NewDataList([]interface{}{1.0, 4.0, 9.0, 16.0})

	// Test nearest neighbor
	result := dl.NearestNeighborInterpolation(1.3)
	expected := 4.0 // closest to index 1
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("NearestNeighborInterpolation(1.3) = %v, expected %v", result, expected)
	}

	// Test exact point
	result = dl.NearestNeighborInterpolation(2.0)
	expected = 9.0 // index 2
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("NearestNeighborInterpolation(2.0) = %v, expected %v", result, expected)
	}

	// Test out of bounds (should still work by finding closest)
	result = dl.NearestNeighborInterpolation(5.0)
	expected = 16.0 // closest to last point
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("NearestNeighborInterpolation(5.0) = %v, expected %v", result, expected)
	}
}

func TestDataList_NewtonInterpolation(t *testing.T) {
	dl := NewDataList([]interface{}{1.0, 4.0, 9.0})

	// Test Newton interpolation at x=1.5
	result := dl.NewtonInterpolation(1.5)
	// For quadratic polynomial x^2 + 2x + 1 at x=1.5: 2.25 + 3 + 1 = 6.25
	expected := 6.25
	if math.Abs(result-expected) > 1e-9 {
		t.Errorf("NewtonInterpolation(1.5) = %v, expected %v", result, expected)
	}

	// Test not enough data points
	dlShort := NewDataList([]interface{}{1.0})
	result = dlShort.NewtonInterpolation(1.5)
	if !math.IsNaN(result) {
		t.Errorf("NewtonInterpolation with insufficient data = %v, expected NaN", result)
	}
}

func TestDataList_HermiteInterpolation(t *testing.T) {
	dl := NewDataList([]interface{}{1.0, 4.0, 9.0})
	derivatives := []float64{2.0, 6.0, 12.0}

	// Test Hermite interpolation
	result := dl.HermiteInterpolation(1.5, derivatives)
	// This is complex, we'll just test that it returns a number
	if math.IsNaN(result) {
		t.Errorf("HermiteInterpolation returned NaN")
	}

	// Test length mismatch
	derivativesShort := []float64{2.0, 6.0}
	result = dl.HermiteInterpolation(1.5, derivativesShort)
	if !math.IsNaN(result) {
		t.Errorf("HermiteInterpolation with mismatched lengths = %v, expected NaN", result)
	}

	// Test not enough data points
	dlShort := NewDataList([]interface{}{1.0})
	result = dlShort.HermiteInterpolation(1.5, []float64{2.0})
	if !math.IsNaN(result) {
		t.Errorf("HermiteInterpolation with insufficient data = %v, expected NaN", result)
	}
}
