package insyra

import (
	"math"
	"testing"
)

// A column CCL computes holds one kind of number, so nobody has to notice
// whether a 0 was written as an integer. When its numbers include a float,
// every integer a float64 holds exactly becomes a float64; an integer past
// 2^53 keeps its digits. Text, booleans, nil and other values are not numbers
// and are left alone, and a column of integers is left as it is.
func TestCCLColumnHoldsOneKindOfNumber(t *testing.T) {
	const big = int64(1<<53 + 1)
	for _, c := range []struct {
		name string
		data []any
		expr string
		want []any
	}{
		{
			name: "a literal fallback in a float column",
			data: []any{"19.5", "abc", "30"},
			expr: "COALESCE(TONUM(A), 0)",
			want: []any{19.5, 0.0, 30.0},
		},
		{
			name: "integers next to fractions",
			data: []any{int64(2), int64(-1), int64(4)},
			expr: "IF(A > 0, A * 1.5, 0)",
			want: []any{3.0, 0.0, 6.0},
		},
		{
			name: "integers stay integers",
			data: []any{int64(1), nil, int64(3)},
			expr: "A + 1",
			want: []any{int64(2), int64(1), int64(4)},
		},
		{
			name: "text and booleans are left alone",
			data: []any{int64(0), int64(1), int64(2)},
			expr: "IF(A == 0, 'none', IF(A == 1, true, IF(A == 2, 3, 2.5)))",
			want: []any{"none", true, int64(3)},
		},
		{
			name: "numbers among text are unified",
			data: []any{int64(0), int64(1), int64(2)},
			expr: "IF(A == 0, 'none', IF(A == 1, 2.5, 3))",
			want: []any{"none", 2.5, 3.0},
		},
		{
			name: "nil stays nil",
			data: []any{int64(0), int64(1), int64(2)},
			expr: "IF(A == 0, nil, IF(A == 1, 2.5, 3))",
			want: []any{nil, 2.5, 3.0},
		},
		{
			name: "an integer a float64 cannot hold keeps its digits",
			data: []any{big, int64(2), int64(3)},
			expr: "IF(# == 2, 0.5, A)",
			want: []any{big, 2.0, 0.5},
		},
		{
			name: "NaN is a float",
			data: []any{math.NaN(), 2.0, 3.0},
			expr: "IF(# == 0, A, 7)",
			want: []any{"NaN", 7.0, 7.0},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			check := func(how string, got []any) {
				t.Helper()
				for i, w := range c.want {
					if w == "NaN" {
						if f, ok := got[i].(float64); !ok || !math.IsNaN(f) {
							t.Errorf("%s row %d = %#v (%T), want float64 NaN", how, i, got[i], got[i])
						}
						continue
					}
					if got[i] != w {
						t.Errorf("%s row %d = %#v (%T), want %#v (%T)", how, i, got[i], got[i], w, w)
					}
				}
			}

			dt := NewDataTable(NewDataList(c.data...).SetName("a"))
			dt.AddColUsingCCL("r", c.expr)
			if err := dt.Err(); err != nil {
				t.Fatal(err)
			}
			check("AddColUsingCCL", dt.GetColByName("r").Data())

			dt = NewDataTable(NewDataList(c.data...).SetName("a"))
			dt.ExecuteCCL("NEW('r') = " + c.expr + "; ['a'] = " + c.expr)
			if err := dt.Err(); err != nil {
				t.Fatal(err)
			}
			check("ExecuteCCL NEW", dt.GetColByName("r").Data())
			check("ExecuteCCL assignment", dt.GetColByName("a").Data())

			dt = NewDataTable(NewDataList(c.data...).SetName("a"))
			dt.EditColByNameUsingCCL("a", c.expr)
			if err := dt.Err(); err != nil {
				t.Fatal(err)
			}
			check("EditColByNameUsingCCL", dt.GetColByName("a").Data())

			dt = NewDataTable(NewDataList(c.data...).SetName("a"))
			dt.EditColByIndexUsingCCL("A", c.expr)
			if err := dt.Err(); err != nil {
				t.Fatal(err)
			}
			check("EditColByIndexUsingCCL", dt.GetColByName("a").Data())
		})
	}
}

// A column of integers copied as it is keeps its Go types; nothing is widened
// when no float asks for it.
func TestCCLColumnOfIntegersKeepsItsTypes(t *testing.T) {
	dt := NewDataTable(NewDataList(int16(1), int16(2)).SetName("a"))
	dt.AddColUsingCCL("b", "A")
	if err := dt.Err(); err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("b").Data(); got[0] != int16(1) || got[1] != int16(2) {
		t.Errorf("A on an int16 column = %#v, want the int16 cells", got)
	}
}
