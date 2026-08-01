package ccl

import (
	"math"
	"testing"
)

// callSeq invokes a registered sequence function by name.
func callSeq(t *testing.T, name string, cols ...[]any) ([]any, error) {
	t.Helper()
	return callSequenceFunction(name, cols)
}

func seqApproxEqual(t *testing.T, got []any, want []any, tol float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] == nil && want[i] == nil {
			continue
		}
		if got[i] == nil || want[i] == nil {
			t.Errorf("[%d] got %v, want %v", i, got[i], want[i])
			continue
		}
		gf, gok := toFloat64(got[i])
		wf, wok := toFloat64(want[i])
		if !gok || !wok {
			if got[i] != want[i] {
				t.Errorf("[%d] got %v (%T), want %v (%T)", i, got[i], got[i], want[i], want[i])
			}
			continue
		}
		if math.Abs(gf-wf) > tol {
			t.Errorf("[%d] got %v, want %v (diff %g > tol %g)", i, gf, wf, math.Abs(gf-wf), tol)
		}
	}
}

func TestCCLSeq_LAG(t *testing.T) {
	out, err := callSeq(t, "LAG", []any{10.0, 20.0, 30.0, 40.0}, []any{1})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{nil, 10.0, 20.0, 30.0}, 1e-9)
}

func TestCCLSeq_LEAD(t *testing.T) {
	out, err := callSeq(t, "LEAD", []any{10.0, 20.0, 30.0, 40.0}, []any{1})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{20.0, 30.0, 40.0, nil}, 1e-9)
}

func TestCCLSeq_DIFF(t *testing.T) {
	out, err := callSeq(t, "DIFF", []any{10.0, 13.0, 18.0, 17.0, 25.0}, []any{1})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{nil, 3.0, 5.0, -1.0, 8.0}, 1e-9)
}

func TestCCLSeq_DIFF_DefaultPeriods(t *testing.T) {
	out, err := callSeq(t, "DIFF", []any{10.0, 13.0, 18.0})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{nil, 3.0, 5.0}, 1e-9)
}

func TestCCLSeq_PCT_CHANGE(t *testing.T) {
	out, err := callSeq(t, "PCT_CHANGE", []any{100.0, 110.0, 99.0}, []any{1})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{nil, 0.1, -0.1}, 1e-9)
}

func TestCCLSeq_CUMSUM(t *testing.T) {
	out, err := callSeq(t, "CUMSUM", []any{1.0, 2.0, 3.0, 4.0})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{1.0, 3.0, 6.0, 10.0}, 1e-9)
}

func TestCCLSeq_CUMPROD(t *testing.T) {
	out, err := callSeq(t, "CUMPROD", []any{2.0, 3.0, 4.0})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{2.0, 6.0, 24.0}, 1e-9)
}

func TestCCLSeq_CUMMAX(t *testing.T) {
	out, err := callSeq(t, "CUMMAX", []any{3.0, 1.0, 4.0, 1.0, 5.0, 9.0, 2.0})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{3.0, 3.0, 4.0, 4.0, 5.0, 9.0, 9.0}, 1e-9)
}

func TestCCLSeq_CUMMIN(t *testing.T) {
	out, err := callSeq(t, "CUMMIN", []any{3.0, 1.0, 4.0, 1.0, 5.0, 9.0, 2.0})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{3.0, 1.0, 1.0, 1.0, 1.0, 1.0, 1.0}, 1e-9)
}

func TestCCLSeq_ROLLING_SUM(t *testing.T) {
	out, err := callSeq(t, "ROLLING_SUM", []any{1.0, 2.0, 3.0, 4.0, 5.0}, []any{2})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{nil, 3.0, 5.0, 7.0, 9.0}, 1e-9)
}

func TestCCLSeq_ROLLING_MEAN(t *testing.T) {
	out, err := callSeq(t, "ROLLING_MEAN", []any{1.0, 2.0, 3.0, 4.0, 5.0}, []any{3})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{nil, nil, 2.0, 3.0, 4.0}, 1e-9)
}

func TestCCLSeq_ROLLING_MIN(t *testing.T) {
	out, err := callSeq(t, "ROLLING_MIN", []any{5.0, 1.0, 4.0, 2.0, 3.0}, []any{3})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{nil, nil, 1.0, 1.0, 2.0}, 1e-9)
}

func TestCCLSeq_ROLLING_MAX(t *testing.T) {
	out, err := callSeq(t, "ROLLING_MAX", []any{5.0, 1.0, 4.0, 2.0, 3.0}, []any{3})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{nil, nil, 5.0, 4.0, 4.0}, 1e-9)
}

func TestCCLSeq_ROLLING_STD(t *testing.T) {
	out, err := callSeq(t, "ROLLING_STD", []any{1.0, 2.0, 3.0, 4.0, 5.0}, []any{3})
	if err != nil {
		t.Fatal(err)
	}
	seqApproxEqual(t, out, []any{nil, nil, 1.0, 1.0, 1.0}, 1e-9)
}

func TestCCLSeq_BadArgs(t *testing.T) {
	if _, err := callSeq(t, "LAG", []any{1, 2, 3}); err == nil {
		t.Error("LAG with 1 arg should error")
	}
	if _, err := callSeq(t, "ROLLING_MEAN", []any{1, 2, 3}, []any{0}); err == nil {
		t.Error("ROLLING_MEAN with window=0 should error")
	}
	if _, err := callSeq(t, "DIFF", []any{1, 2}, []any{-1}); err == nil {
		t.Error("DIFF with periods=-1 should error")
	}
}

func TestCCLSeq_IsSequenceFunction(t *testing.T) {
	for _, name := range []string{"LAG", "LEAD", "DIFF", "PCT_CHANGE",
		"CUMSUM", "CUMPROD", "CUMMAX", "CUMMIN",
		"ROLLING_SUM", "ROLLING_MEAN", "ROLLING_MIN", "ROLLING_MAX", "ROLLING_STD"} {
		if !IsSequenceFunction(name) {
			t.Errorf("expected %s to be a sequence function", name)
		}
		if !IsSequenceFunction(name[:len(name)/2] + name[len(name)/2:]) {
			t.Errorf("expected case-insensitive lookup to work for %s", name)
		}
	}
	if IsSequenceFunction("SUM") {
		t.Error("SUM is an aggregate, not a sequence function")
	}
}
