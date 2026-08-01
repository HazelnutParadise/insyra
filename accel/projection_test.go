package accel

import (
	"math"
	"testing"
)

// Celsius exercises the named-type path: reflect matches it by Kind, a plain
// type switch does not, so it is the case a naive rewrite would break.
type Celsius float64

type Count int32

func TestProjectNamedNumericTypes(t *testing.T) {
	buf, err := projectValues("temps", []any{Celsius(1.5), Celsius(-2.25)})
	if err != nil {
		t.Fatalf("named float type must still project: %v", err)
	}
	if buf.Type != DataTypeFloat64 {
		t.Fatalf("expected float64 column, got %q", buf.Type)
	}
	values, ok := buf.Values.([]float64)
	if !ok {
		t.Fatalf("expected []float64, got %T", buf.Values)
	}
	if values[0] != 1.5 || values[1] != -2.25 {
		t.Fatalf("unexpected values: %v", values)
	}

	counts, err := projectValues("counts", []any{Count(3), Count(-4)})
	if err != nil {
		t.Fatalf("named int type must still project: %v", err)
	}
	if counts.Type != DataTypeInt64 {
		t.Fatalf("expected int64 column, got %q", counts.Type)
	}
	if got := counts.Values.([]int64); got[0] != 3 || got[1] != -4 {
		t.Fatalf("unexpected values: %v", got)
	}
}

func TestProjectMixedIntAndFloatBecomesFloat64(t *testing.T) {
	buf, err := projectValues("mixed", []any{1, 2.5, int64(3)})
	if err != nil {
		t.Fatalf("project failed: %v", err)
	}
	if buf.Type != DataTypeFloat64 {
		t.Fatalf("expected float64 column, got %q", buf.Type)
	}
	if got := buf.Values.([]float64); got[0] != 1 || got[1] != 2.5 || got[2] != 3 {
		t.Fatalf("unexpected values: %v", got)
	}
}

func TestProjectAllIntegerKindsBecomeInt64(t *testing.T) {
	buf, err := projectValues("ints", []any{int8(1), int16(2), int32(3), int64(4), uint8(5), uint16(6), uint32(7), uint(8)})
	if err != nil {
		t.Fatalf("project failed: %v", err)
	}
	if buf.Type != DataTypeInt64 {
		t.Fatalf("expected int64 column, got %q", buf.Type)
	}
	want := []int64{1, 2, 3, 4, 5, 6, 7, 8}
	got := buf.Values.([]int64)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("at %d: expected %d, got %d", i, want[i], got[i])
		}
	}
}

func TestProjectFloat32Narrowing(t *testing.T) {
	buf, err := projectValues("f32", []any{float32(1.5), float32(0.1)})
	if err != nil {
		t.Fatalf("project failed: %v", err)
	}
	got := buf.Values.([]float64)
	if got[0] != float64(float32(1.5)) || got[1] != float64(float32(0.1)) {
		t.Fatalf("float32 must widen exactly as a Go conversion does, got %v", got)
	}
}

func TestProjectUint64AboveMaxInt64Wraps(t *testing.T) {
	// Documented as deliberate rather than accidental: reflect's int64(rv.Uint())
	// wraps, and a type switch must wrap the same way.
	buf, err := projectValues("big", []any{uint64(math.MaxInt64) + 1})
	if err != nil {
		t.Fatalf("project failed: %v", err)
	}
	if got := buf.Values.([]int64); got[0] != math.MinInt64 {
		t.Fatalf("expected wrap to MinInt64, got %d", got[0])
	}
}

func TestProjectNonNumericValueDemotesColumnToAny(t *testing.T) {
	// Inference sees the struct first and demotes the whole column, so the
	// converter's error branch is unreachable from here. Pinning the real
	// behaviour rather than the behaviour the error message implies.
	type opaque struct{ n int }
	buf, err := projectValues("bad", []any{1.0, opaque{2}})
	if err != nil {
		t.Fatalf("a non-numeric value demotes the column, it does not error: %v", err)
	}
	if buf.Type != DataTypeAny {
		t.Fatalf("expected an any column, got %q", buf.Type)
	}
}

func TestNumericConvertersRejectNonNumericValues(t *testing.T) {
	// The converters' own contract, which projectValues relies on defensively.
	if _, ok := toFloat64("nope"); ok {
		t.Fatal("toFloat64 must reject a string")
	}
	if _, ok := toFloat64(struct{}{}); ok {
		t.Fatal("toFloat64 must reject a struct")
	}
	if _, ok := toInt64(1.5); ok {
		t.Fatal("toInt64 must reject a float")
	}
	if _, ok := toInt64(nil); ok {
		t.Fatal("toInt64 must reject nil")
	}
	if _, ok := toFloat64(nil); ok {
		t.Fatal("toFloat64 must reject nil")
	}
}

func TestProjectNullsAndValidityUnchanged(t *testing.T) {
	buf, err := projectValues("nullable", []any{1.0, nil, 3.0, nil, 5.0, 6.0, 7.0, 8.0, 9.0})
	if err != nil {
		t.Fatalf("project failed: %v", err)
	}
	wantNulls := []bool{false, true, false, true, false, false, false, false, false}
	for i, want := range wantNulls {
		if buf.Nulls[i] != want {
			t.Fatalf("nulls[%d]: expected %v, got %v", i, want, buf.Nulls[i])
		}
	}
	// bit set means valid; indices 1 and 3 are null; padding bits stay clear.
	want := []byte{0b11110101, 0b00000001}
	if len(buf.Validity) != len(want) {
		t.Fatalf("expected %d validity bytes, got %d", len(want), len(buf.Validity))
	}
	for i := range want {
		if buf.Validity[i] != want[i] {
			t.Fatalf("validity[%d]: expected %08b, got %08b", i, want[i], buf.Validity[i])
		}
	}
	if got := buf.Values.([]float64); got[1] != 0 || got[3] != 0 {
		t.Fatalf("null positions must hold the zero value, got %v", got)
	}
}

func TestProjectBoolStringAndMixedColumns(t *testing.T) {
	bools, err := projectValues("flags", []any{true, nil, false})
	if err != nil || bools.Type != DataTypeBool {
		t.Fatalf("expected a bool column, got %q err=%v", bools.Type, err)
	}
	strs, err := projectValues("labels", []any{"ab", nil, "c"})
	if err != nil || strs.Type != DataTypeString {
		t.Fatalf("expected a string column, got %q err=%v", strs.Type, err)
	}
	if string(strs.StringData) != "abc" {
		t.Fatalf("unexpected string data %q", strs.StringData)
	}
	mixed, err := projectValues("mixed", []any{1, "a", true})
	if err != nil || mixed.Type != DataTypeAny {
		t.Fatalf("expected an any column, got %q err=%v", mixed.Type, err)
	}
}

func TestProjectEmptyColumn(t *testing.T) {
	buf, err := projectValues("empty", nil)
	if err != nil {
		t.Fatalf("project failed: %v", err)
	}
	if buf.Len != 0 {
		t.Fatalf("expected zero length, got %d", buf.Len)
	}
	if buf.Validity != nil {
		t.Fatalf("expected no validity bitmap for an empty column, got %v", buf.Validity)
	}
}
