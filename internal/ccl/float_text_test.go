package ccl

import "testing"

func TestFloatsBecomePlainDecimalTextInCCLStrings(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{0.0000001, "0.0000001"},
		{1e6, "1000000"},
		{12345.678, "12345.678"},
		{1.5, "1.5"},
		{1e21, "1e+21"},
		{float32(0.1), "0.1"},
		{-2.5e-7, "-0.00000025"},
		{0.0, "0"},
		{3, "3"},
		{"s", "s"},
	}
	for _, c := range cases {
		if got := textOf(c.in); got != c.want {
			t.Errorf("textOf(%#v) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := toString(1e6); got != "1000000" {
		t.Errorf("toString(1e6) = %q", got)
	}
}

func TestAmpersandAndConcatWritePlainDecimals(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"a": {0.0000001, 1e6, 12345.678}})
	for expr, want := range map[string][]any{
		"'x' & A":        {"x0.0000001", "x1000000", "x12345.678"},
		"CONCAT('x', A)": {"x0.0000001", "x1000000", "x12345.678"},
		"LEN(A)":         {float64(9), float64(7), float64(9)},
	} {
		got, err := evalCol(t, ctx, expr)
		if err != nil {
			t.Fatalf("%s: %v", expr, err)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s row %d = %#v (%T), want %#v", expr, i, got[i], got[i], want[i])
			}
		}
	}
}
