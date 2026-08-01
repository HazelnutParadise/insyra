package stats

import (
	"math"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestZQuantile(t *testing.T) {
	got := zQuantile(0.975)
	want := 1.959963984540054
	if math.Abs(got-want) > 1e-12 {
		t.Fatalf("zQuantile mismatch: got %.15g want %.15g", got, want)
	}
}

func TestSigmoidStable(t *testing.T) {
	cases := []struct {
		x    float64
		want float64
	}{
		{0, 0.5},
		{1000, 1},
		{-1000, 0},
	}
	for _, tc := range cases {
		got := sigmoid(tc.x)
		if math.Abs(got-tc.want) > 1e-15 {
			t.Fatalf("sigmoid(%v) = %.15g, want %.15g", tc.x, got, tc.want)
		}
	}
}

func TestGatherRegressionInputsAndDesignMatrix(t *testing.T) {
	y := insyra.NewDataList(1.0, 2.0, 3.0)
	x1 := insyra.NewDataList(10.0, 20.0, 30.0)
	x2 := insyra.NewDataList(2.0, 4.0, 6.0)
	offset := insyra.NewDataList(0.1, 0.2, 0.3)

	gotY, gotXs, gotExtras, n, err := gatherRegressionInputs(y, []insyra.IDataList{x1, x2}, offset)
	if err != nil {
		t.Fatalf("gatherRegressionInputs returned error: %v", err)
	}
	if n != 3 || len(gotY) != 3 || len(gotXs) != 2 || len(gotExtras) != 1 {
		t.Fatalf("unexpected shapes: n=%d len(y)=%d len(xs)=%d len(extras)=%d", n, len(gotY), len(gotXs), len(gotExtras))
	}

	X := buildDesignMatrix(gotXs, n)
	want := [][]float64{
		{1, 10, 2},
		{1, 20, 4},
		{1, 30, 6},
	}
	for i := range n {
		for j := range want[i] {
			if got := X.At(i, j); got != want[i][j] {
				t.Fatalf("X[%d,%d] = %v, want %v", i, j, got, want[i][j])
			}
		}
	}
}
