package ccl

import (
	"math"
	"testing"
)

// ROUND rounds the number as it is written, so 1.005 reaches 1.01 although the
// float64 closest to it lies just below.
func TestRoundReadsTheWrittenNumber(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {1.005}})
	for _, c := range []struct {
		expr string
		want float64
	}{
		{"ROUND(1.005, 2)", 1.01},
		{"ROUND(A, 2)", 1.01},
		{"ROUND(-1.005, 2)", -1.01},
		{"ROUND(0.285, 2)", 0.29},
		{"ROUND(1.255, 2)", 1.26},
		{"ROUND(2.5)", 3},
		{"ROUND(-2.5)", -3},
		{"ROUND(0.5)", 1},
		{"ROUND(0.4)", 0},
		{"ROUND(3.14159, 2)", 3.14},
		{"ROUND(9.995, 2)", 10},
		{"ROUND(99.5)", 100},
		{"ROUND(0.0004, 2)", 0},
		{"ROUND(0.005, 2)", 0.01},
		{"ROUND(1234, -2)", 1200},
		{"ROUND(1250, -2)", 1300},
		{"ROUND(5, -1)", 10},
		{"ROUND(4, -1)", 0},
		{"ROUND(5, -2)", 0},
		{"ROUND(1.5, 10)", 1.5},
		{"ROUND(1e21)", 1e21},
		{"ROUND(1.5e300, -299)", 1.5e300},
		{"ROUND(1.55e300, -299)", 1.6e300},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if f, ok := toFloat64(got); !ok || f != c.want {
			t.Errorf("%s = %#v, want %v", c.expr, got, c.want)
		}
	}
	for _, c := range []struct {
		expr string
		sign int
	}{{"ROUND(1.79e308, -308)", 1}, {"ROUND(-1.79e308, -308)", -1}} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if f, _ := toFloat64(got); !math.IsInf(f, c.sign) {
			t.Errorf("%s = %v, want an infinity", c.expr, got)
		}
	}
	got, err := evalOne(t, ctx, "ROUND(-0.4)")
	if err != nil {
		t.Fatal(err)
	}
	if f, _ := toFloat64(got); math.Signbit(f) {
		t.Errorf("ROUND(-0.4) = %v, a negative zero", f)
	}
}

func TestRoundKeepsWhatIsNotANumber(t *testing.T) {
	for _, rule := range []roundRule{roundNearest, roundAway, roundToward} {
		if got := roundWritten(math.Copysign(0, -1), -1, rule); math.Signbit(got) {
			t.Errorf("%v of a negative zero stayed negative", rule)
		}
	}
	if got := roundWritten(math.NaN(), 2, roundNearest); !math.IsNaN(got) {
		t.Errorf("NaN became %v", got)
	}
	if got := roundWritten(math.Inf(1), 2, roundNearest); !math.IsInf(got, 1) {
		t.Errorf("+Inf became %v", got)
	}
}
