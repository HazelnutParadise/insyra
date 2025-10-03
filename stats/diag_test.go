package stats

import (
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestDiag(t *testing.T) {
	// Test extracting diagonal from matrix
	matrix := mat.NewDense(3, 3, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9})
	result := Diag(matrix, 0, 0, false)
	diag, ok := result.([]float64)
	if !ok {
		t.Errorf("Expected []float64, got %T", result)
	}
	expected := []float64{1, 5, 9}
	for i, v := range expected {
		if diag[i] != v {
			t.Errorf("Expected %v, got %v", expected, diag)
		}
	}

	// Test creating diagonal matrix from slice
	vec := []float64{1, 2, 3}
	result = Diag(vec, 0, 0, false)
	diagMat, ok := result.(*mat.Dense)
	if !ok {
		t.Errorf("Expected *mat.Dense, got %T", result)
	}
	r, c := diagMat.Dims()
	if r != 3 || c != 3 {
		t.Errorf("Expected 3x3 matrix, got %dx%d", r, c)
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i == j {
				if diagMat.At(i, j) != float64(i+1) {
					t.Errorf("Diagonal element wrong")
				}
			} else {
				if diagMat.At(i, j) != 0 {
					t.Errorf("Off-diagonal should be 0")
				}
			}
		}
	}

	// Test creating identity matrix
	result = Diag(3, 0, 0, false)
	idMat, ok := result.(*mat.Dense)
	if !ok {
		t.Errorf("Expected *mat.Dense, got %T", result)
	}
	r, c = idMat.Dims()
	if r != 3 || c != 3 {
		t.Errorf("Expected 3x3 matrix, got %dx%d", r, c)
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i == j {
				if idMat.At(i, j) != 1.0 {
					t.Errorf("Identity diagonal should be 1")
				}
			} else {
				if idMat.At(i, j) != 0 {
					t.Errorf("Off-diagonal should be 0")
				}
			}
		}
	}

	// Test creating identity matrix from float64
	result = Diag(3.0, 0, 0, false)
	idMat2, ok := result.(*mat.Dense)
	if !ok {
		t.Errorf("Expected *mat.Dense, got %T", result)
	}
	r, c = idMat2.Dims()
	if r != 3 || c != 3 {
		t.Errorf("Expected 3x3 matrix, got %dx%d", r, c)
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i == j {
				if idMat2.At(i, j) != 1.0 {
					t.Errorf("Identity diagonal should be 1")
				}
			} else {
				if idMat2.At(i, j) != 0 {
					t.Errorf("Off-diagonal should be 0")
				}
			}
		}
	}

	// Test creating identity matrix from nil
	result = Diag(nil, 3, 3, false)
	idMat3, ok := result.(*mat.Dense)
	if !ok {
		t.Errorf("Expected *mat.Dense, got %T", result)
	}
	r, c = idMat3.Dims()
	if r != 3 || c != 3 {
		t.Errorf("Expected 3x3 matrix, got %dx%d", r, c)
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i == j {
				if idMat3.At(i, j) != 1.0 {
					t.Errorf("Identity diagonal should be 1")
				}
			} else {
				if idMat3.At(i, j) != 0 {
					t.Errorf("Off-diagonal should be 0")
				}
			}
		}
	}
}
