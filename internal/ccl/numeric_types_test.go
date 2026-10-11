package ccl

import (
	"fmt"
	"math"
	"testing"
)

// everyNumericType is one value of each Go type CCL must read as a number.
// Before this change CCL knew only int, int32, int64, float32 and float64, so
// every other type was either an error or a silent wrong answer.
var everyNumericType = []any{
	int8(1),
	int16(1),
	int32(1),
	int64(1),
	int(1),
	uint8(1),
	uint16(1),
	uint32(1),
	uint64(1),
	uint(1),
	float32(1),
}

// TestEveryIntegerTypeIsANumber pins that arithmetic, comparison, conditions,
// functions and aggregates all read a cell of any Go numeric type, by giving
// the same value 1 in each type and comparing the answer as printed.
func TestEveryIntegerTypeIsANumber(t *testing.T) {
	for _, v := range everyNumericType {
		for expr, want := range map[string]string{
			"A * 2":           "2",
			"A == 1":          "true",
			"IF(A, 'y', 'n')": "y",
			"SUM(A)":          "2",
			"MAX(A)":          "1",
			"ROUND(A, A)":     "1",
			"B.(A)":           "q",
		} {
			t.Run(fmt.Sprintf("%T %s", v, expr), func(t *testing.T) {
				ctx := mapCtx(t, map[string][]any{
					"A": {v, v},
					"B": {"p", "q"},
				})
				got, err := evalCol(t, ctx, expr)
				if err != nil {
					t.Fatalf("%s: %v", expr, err)
				}
				if s := fmt.Sprint(got[0]); s != want {
					t.Fatalf("row 0 of %s = %s, want %s", expr, s, want)
				}
			})
		}
	}
}

// TestWordAgainstEveryNumericType pins that ordering a word against a cell of
// any numeric type is the error it is against an int or a float64, not a
// silent false.
func TestWordAgainstEveryNumericType(t *testing.T) {
	for _, v := range everyNumericType {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			ctx := mapCtx(t, map[string][]any{"A": {v}})
			if got, err := evalCol(t, ctx, "'hello' > A"); err == nil {
				t.Fatalf("'hello' > A = %v, want an error", got)
			}
		})
	}
}

// TestNarrowIntegerAggregates is the case the proposal was measured on: a
// mixed narrow column whose aggregates answered 0 and nil without an error.
func TestNarrowIntegerAggregates(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {int16(3), int16(4), uint8(5)},
	})
	for expr, want := range map[string]string{
		"SUM(A)": "12",
		"MAX(A)": "5",
		"AVG(A)": "4",
	} {
		t.Run(expr, func(t *testing.T) {
			got, err := evalCol(t, ctx, expr)
			if err != nil {
				t.Fatalf("%s: %v", expr, err)
			}
			if s := fmt.Sprint(got[0]); s != want {
				t.Fatalf("%s = %s, want %s", expr, s, want)
			}
		})
	}
}

// TestUint64PastExactFloats pins that a uint64 above 2^53 is read as the
// nearest float64, the way an int64 that large already is.
func TestUint64PastExactFloats(t *testing.T) {
	want := fmt.Sprint(float64(1 << 53))
	for _, v := range []any{uint64(1<<53 + 1), int64(1<<53 + 1)} {
		t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
			ctx := mapCtx(t, map[string][]any{"A": {v}})
			got, err := evalCol(t, ctx, "A + 0")
			if err != nil {
				t.Fatalf("A + 0: %v", err)
			}
			if s := fmt.Sprint(got[0]); s != want {
				t.Fatalf("A + 0 = %s, want %s", s, want)
			}
		})
	}
}

// TestFloat32NaNIsMissing pins that a float32 NaN is missing to ISNA and IFNA,
// the way a float64 one is.
func TestFloat32NaNIsMissing(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {float32(math.NaN())}})
	for expr, want := range map[string]string{
		"ISNA(A)":    "true",
		"IFNA(A, 0)": "0",
	} {
		t.Run(expr, func(t *testing.T) {
			got, err := evalCol(t, ctx, expr)
			if err != nil {
				t.Fatalf("%s: %v", expr, err)
			}
			if s := fmt.Sprint(got[0]); s != want {
				t.Fatalf("%s = %s, want %s", expr, s, want)
			}
		})
	}
}

// TestRowIndexOfEveryNumericType pins the row-index rule for every numeric
// type. A row index of any type reads the way a float64 index always has on
// this line: a fraction is dropped.
func TestRowIndexOfEveryNumericType(t *testing.T) {
	t.Run("fractional float32 drops the fraction like float64", func(t *testing.T) {
		for _, idx := range []any{float64(1.5), float32(1.5)} {
			ctx := mapCtx(t, map[string][]any{
				"A": {int64(10), int64(20)},
				"B": {idx, idx},
			})
			got, err := evalCol(t, ctx, "A.B")
			if err != nil {
				t.Fatalf("A.B with a %T(1.5) row index: %v", idx, err)
			}
			if got[0] != int64(20) {
				t.Fatalf("A.B with a %T(1.5) row index = %v, want 20", idx, got[0])
			}
		}
	})

	t.Run("whole numeric index names the row", func(t *testing.T) {
		for _, v := range everyNumericType {
			t.Run(fmt.Sprintf("%T", v), func(t *testing.T) {
				ctx := mapCtx(t, map[string][]any{
					"A": {"p", "q", "r"},
					"B": {v, v, v},
				})
				got, err := evalCol(t, ctx, "A.B")
				if err != nil {
					t.Fatalf("A.B with a %T row index: %v", v, err)
				}
				if got[0] != "q" {
					t.Fatalf("A.B with a %T row index = %v, want q", v, got[0])
				}
			})
		}
	})

	// int(f) on NaN or past the int range differs between amd64 and arm64, so
	// a float index that cannot name a row is refused rather than read as
	// whatever row the platform makes of it.
	t.Run("float index that names no row is refused", func(t *testing.T) {
		for _, idx := range []any{float32(math.NaN()), float32(math.Inf(1)), float32(1e10), math.NaN(), math.Inf(-1), 1e10} {
			ctx := mapCtx(t, map[string][]any{
				"A": {int64(10), int64(20)},
				"B": {idx, idx},
			})
			if got, err := evalCol(t, ctx, "A.B"); err == nil {
				t.Fatalf("A.B with a %T(%v) row index = %v, want an error", idx, idx, got)
			}
		}
	})
}
