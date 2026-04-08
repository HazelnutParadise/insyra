package linalg

import (
	"math"
	"testing"
)

// R baseline values generated from:
// A <- matrix(c(4,2,1,0,5,3,2,1,3), nrow=3, byrow=TRUE)
// b <- c(7,8,5)
// solve(A,b); solve(A); det(A)
func TestLinalgPrimitivesAgainstR(t *testing.T) {
	const tol = 1e-12

	A := [][]float64{
		{4, 2, 1},
		{0, 5, 3},
		{2, 1, 3},
	}
	b := []float64{7, 8, 5}

	gotX := GaussianElimination(A, b)
	wantX := []float64{0.98, 1.24, 0.6}
	if len(gotX) != len(wantX) {
		t.Fatalf("GaussianElimination length mismatch")
	}
	for i := range gotX {
		if !rAlmostEqual(gotX[i], wantX[i], tol) {
			t.Fatalf("GaussianElimination[%d] mismatch: got %v want %v", i, gotX[i], wantX[i])
		}
	}

	gotInv := InvertMatrix(A)
	wantInv := [][]float64{
		{0.24, -0.1, 0.02},
		{0.12, 0.2, -0.24},
		{-0.2, 0.0, 0.4},
	}
	if len(gotInv) != len(wantInv) {
		t.Fatalf("InvertMatrix row mismatch")
	}
	for i := range gotInv {
		if len(gotInv[i]) != len(wantInv[i]) {
			t.Fatalf("InvertMatrix col mismatch at row %d", i)
		}
		for j := range gotInv[i] {
			if !rAlmostEqual(gotInv[i][j], wantInv[i][j], tol) {
				t.Fatalf("InvertMatrix[%d][%d] mismatch: got %v want %v", i, j, gotInv[i][j], wantInv[i][j])
			}
		}
	}

	gotDet := DeterminantGauss(A)
	if !rAlmostEqual(gotDet, 49.999999999999993, 1e-10) {
		t.Fatalf("DeterminantGauss mismatch: got %v want %v", gotDet, 49.999999999999993)
	}
}

func rAlmostEqual(got, want, tol float64) bool {
	return math.Abs(got-want) <= tol
}
