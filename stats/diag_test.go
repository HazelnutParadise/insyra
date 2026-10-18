package stats

import (
	"reflect"
	"testing"

	"gonum.org/v1/gonum/mat"
)

func TestDiagOf(t *testing.T) {
	// 3x3 -> [1 5 9]
	matrix := mat.NewDense(3, 3, []float64{1, 2, 3, 4, 5, 6, 7, 8, 9})
	diag, err := DiagOf(matrix)
	if err != nil {
		t.Fatalf("DiagOf(3x3) returned error: %v", err)
	}
	expected := []float64{1, 5, 9}
	for i, v := range expected {
		if diag[i] != v {
			t.Errorf("Expected %v, got %v", expected, diag)
		}
	}

	// 2x3 NewDense(2,3,{1..6}) -> [1 5]
	matrix2 := mat.NewDense(2, 3, []float64{1, 2, 3, 4, 5, 6})
	diag2, err := DiagOf(matrix2)
	if err != nil {
		t.Fatalf("DiagOf(2x3) returned error: %v", err)
	}
	expected2 := []float64{1, 5}
	for i, v := range expected2 {
		if diag2[i] != v {
			t.Errorf("Expected %v, got %v", expected2, diag2)
		}
	}

	// 3x2 -> [1 4]
	matrix3 := mat.NewDense(3, 2, []float64{1, 2, 3, 4, 5, 6})
	diag3, err := DiagOf(matrix3)
	if err != nil {
		t.Fatalf("DiagOf(3x2) returned error: %v", err)
	}
	expected3 := []float64{1, 4}
	for i, v := range expected3 {
		if diag3[i] != v {
			t.Errorf("Expected %v, got %v", expected3, diag3)
		}
	}

	// a *mat.SymDense also works
	sym := mat.NewSymDense(3, []float64{1, 2, 3, 2, 5, 6, 3, 6, 9})
	diag4, err := DiagOf(sym)
	if err != nil {
		t.Fatalf("DiagOf(SymDense) returned error: %v", err)
	}
	expected4 := []float64{1, 5, 9}
	for i, v := range expected4 {
		if diag4[i] != v {
			t.Errorf("Expected %v, got %v", expected4, diag4)
		}
	}

	// DiagOf(nil) error
	_, err = DiagOf(nil)
	if err == nil {
		t.Error("DiagOf(nil) expected error, got nil")
	}

	// var d *mat.Dense; DiagOf(d) error (no panic)
	var d *mat.Dense
	_, err = DiagOf(d)
	if err == nil {
		t.Error("DiagOf(nil *mat.Dense) expected error, got nil")
	}
}

func TestDiagMatrix(t *testing.T) {
	// [1 2 3] -> 3x3 with that diagonal and zeros elsewhere
	v := []float64{1, 2, 3}
	m, err := DiagMatrix(v)
	if err != nil {
		t.Fatalf("DiagMatrix returned error: %v", err)
	}
	r, c := m.Dims()
	if r != 3 || c != 3 {
		t.Errorf("Expected 3x3 matrix, got %dx%d", r, c)
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i == j {
				if m.At(i, j) != float64(i+1) {
					t.Errorf("Diagonal element [%d][%d] wrong: got %v, want %v", i, j, m.At(i, j), float64(i+1))
				}
			} else {
				if m.At(i, j) != 0 {
					t.Errorf("Off-diagonal [%d][%d] should be 0, got %v", i, j, m.At(i, j))
				}
			}
		}
	}

	// nil and []float64{} -> error
	_, err = DiagMatrix(nil)
	if err == nil {
		t.Error("DiagMatrix(nil) expected error, got nil")
	}
	_, err = DiagMatrix([]float64{})
	if err == nil {
		t.Error("DiagMatrix([]float64{}) expected error, got nil")
	}

	// the input slice is unchanged afterwards
	vCopy := make([]float64, len(v))
	copy(vCopy, v)
	_, err = DiagMatrix(v)
	if err != nil {
		t.Fatalf("DiagMatrix(v) returned error: %v", err)
	}
	if !reflect.DeepEqual(v, vCopy) {
		t.Errorf("Input slice was modified: %v", v)
	}
}

func TestDiagMatrixSize(t *testing.T) {
	// ([1 1], 2, 3) -> [[1 0 0] [0 1 0]]
	v := []float64{1, 1}
	mat2, err := DiagMatrixSize(v, 2, 3)
	if err != nil {
		t.Fatalf("DiagMatrixSize([1,1],2,3) returned error: %v", err)
	}
	r, c := mat2.Dims()
	if r != 2 || c != 3 {
		t.Errorf("Expected 2x3 matrix, got %dx%d", r, c)
	}
	expected := [][]float64{{1, 0, 0}, {0, 1, 0}}
	for i := 0; i < 2; i++ {
		for j := 0; j < 3; j++ {
			if mat2.At(i, j) != expected[i][j] {
				t.Errorf("mat[%d][%d] = %v, want %v", i, j, mat2.At(i, j), expected[i][j])
			}
		}
	}

	// ([7], 3, 3) -> 7 then zeros on the diagonal
	v2 := []float64{7}
	mat3, err := DiagMatrixSize(v2, 3, 3)
	if err != nil {
		t.Fatalf("DiagMatrixSize([7],3,3) returned error: %v", err)
	}
	if mat3.At(0, 0) != 7 {
		t.Errorf("Expected mat[0][0]=7, got %v", mat3.At(0, 0))
	}
	for i := 1; i < 3; i++ {
		if mat3.At(i, i) != 0 {
			t.Errorf("Expected mat[%d][%d]=0, got %v", i, i, mat3.At(i, i))
		}
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i != j && mat3.At(i, j) != 0 {
				t.Errorf("Expected mat[%d][%d]=0, got %v", i, j, mat3.At(i, j))
			}
		}
	}

	// ([1 2 3], 2, 2) -> error, nil matrix
	_, err = DiagMatrixSize([]float64{1, 2, 3}, 2, 2)
	if err == nil {
		t.Error("DiagMatrixSize([1,2,3],2,2) expected error, got nil")
	}

	// (v, 0, 3), (v, 3, -1) -> error
	_, err = DiagMatrixSize([]float64{1}, 0, 3)
	if err == nil {
		t.Error("DiagMatrixSize(v,0,3) expected error, got nil")
	}
	_, err = DiagMatrixSize([]float64{1}, 3, -1)
	if err == nil {
		t.Error("DiagMatrixSize(v,3,-1) expected error, got nil")
	}
}

func TestIdentityMatrix(t *testing.T) {
	// 3 -> 3x3 identity
	m, err := IdentityMatrix(3)
	if err != nil {
		t.Fatalf("IdentityMatrix(3) returned error: %v", err)
	}
	r, c := m.Dims()
	if r != 3 || c != 3 {
		t.Errorf("Expected 3x3 matrix, got %dx%d", r, c)
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if i == j {
				if m.At(i, j) != 1.0 {
					t.Errorf("Identity diagonal should be 1 at [%d][%d], got %v", i, j, m.At(i, j))
				}
			} else {
				if m.At(i, j) != 0 {
					t.Errorf("Off-diagonal should be 0 at [%d][%d], got %v", i, j, m.At(i, j))
				}
			}
		}
	}

	// 0 and -2 -> error
	_, err = IdentityMatrix(0)
	if err == nil {
		t.Error("IdentityMatrix(0) expected error, got nil")
	}
	_, err = IdentityMatrix(-2)
	if err == nil {
		t.Error("IdentityMatrix(-2) expected error, got nil")
	}
}
