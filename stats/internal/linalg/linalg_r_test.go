package linalg

import (
	"math"
	"testing"
)

type linalgScalarCase struct {
	primitive string
	rFormula  string
	eval      func() float64
	want      float64
	tol       float64
}

type linalgVectorCase struct {
	primitive string
	rFormula  string
	eval      func() []float64
	want      []float64
	tol       float64
}

type linalgMatrixCase struct {
	primitive string
	rFormula  string
	eval      func() [][]float64
	want      [][]float64
	tol       float64
}

func runLinalgScalarCases(t *testing.T, cases []linalgScalarCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.primitive, func(t *testing.T) {
			got := tc.eval()
			if !rAlmostEqual(got, tc.want, tc.tol) {
				t.Fatalf("primitive=%s got=%.17g want=%.17g r_formula=%q", tc.primitive, got, tc.want, tc.rFormula)
			}
		})
	}
}

func runLinalgVectorCases(t *testing.T, cases []linalgVectorCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.primitive, func(t *testing.T) {
			got := tc.eval()
			if len(got) != len(tc.want) {
				t.Fatalf("primitive=%s got_len=%d want_len=%d r_formula=%q", tc.primitive, len(got), len(tc.want), tc.rFormula)
			}
			for i := range got {
				if !rAlmostEqual(got[i], tc.want[i], tc.tol) {
					t.Fatalf("primitive=%s idx=%d got=%.17g want=%.17g r_formula=%q", tc.primitive, i, got[i], tc.want[i], tc.rFormula)
				}
			}
		})
	}
}

func runLinalgMatrixCases(t *testing.T, cases []linalgMatrixCase) {
	t.Helper()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.primitive, func(t *testing.T) {
			got := tc.eval()
			if len(got) != len(tc.want) {
				t.Fatalf("primitive=%s got_rows=%d want_rows=%d r_formula=%q", tc.primitive, len(got), len(tc.want), tc.rFormula)
			}
			for i := range got {
				if len(got[i]) != len(tc.want[i]) {
					t.Fatalf("primitive=%s row=%d got_cols=%d want_cols=%d r_formula=%q", tc.primitive, i, len(got[i]), len(tc.want[i]), tc.rFormula)
				}
				for j := range got[i] {
					if !rAlmostEqual(got[i][j], tc.want[i][j], tc.tol) {
						t.Fatalf("primitive=%s idx=[%d][%d] got=%.17g want=%.17g r_formula=%q", tc.primitive, i, j, got[i][j], tc.want[i][j], tc.rFormula)
					}
				}
			}
		})
	}
}

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

	runLinalgVectorCases(t, []linalgVectorCase{
		{
			primitive: "GaussianElimination",
			rFormula:  "solve(matrix(c(4,2,1,0,5,3,2,1,3), nrow=3, byrow=TRUE), c(7,8,5))",
			eval:      func() []float64 { return GaussianElimination(A, b) },
			want:      []float64{0.98, 1.24, 0.6},
			tol:       tol,
		},
	})

	runLinalgMatrixCases(t, []linalgMatrixCase{
		{
			primitive: "InvertMatrix",
			rFormula:  "solve(matrix(c(4,2,1,0,5,3,2,1,3), nrow=3, byrow=TRUE))",
			eval:      func() [][]float64 { return InvertMatrix(A) },
			want: [][]float64{
				{0.24, -0.1, 0.02},
				{0.12, 0.2, -0.24},
				{-0.2, 0.0, 0.4},
			},
			tol: tol,
		},
	})

	runLinalgScalarCases(t, []linalgScalarCase{
		{
			primitive: "DeterminantGauss",
			rFormula:  "det(matrix(c(4,2,1,0,5,3,2,1,3), nrow=3, byrow=TRUE))",
			eval:      func() float64 { return DeterminantGauss(A) },
			want:      49.999999999999993,
			tol:       1e-10,
		},
	})
}

func rAlmostEqual(got, want, tol float64) bool {
	return math.Abs(got-want) <= tol
}
