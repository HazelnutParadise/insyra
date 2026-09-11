package utils

import (
	"math"
	"testing"
)

// The rule is the one encoding/json applies: a plain decimal for magnitudes
// from 1e-6 up to (not including) 1e21, exponent form outside. Go's %v switches
// at one million instead, which turned an ordinary revenue into "1.5e+06".
func TestFloatText(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{math.Copysign(0, -1), "-0"},
		{1, "1"},
		{12345.678, "12345.678"},
		{999999, "999999"},
		{1000000, "1000000"},   // %v: 1e+06
		{1500000, "1500000"},   // %v: 1.5e+06
		{-1500000, "-1500000"}, // sign kept
		{1234567.5, "1234567.5"},
		{1e20, "100000000000000000000"},
		{1e21, "1e+21"},       // at the upper edge: exponent, as before
		{1.5e300, "1.5e+300"}, // stays short
		{0.0001, "0.0001"},
		{0.00001, "0.00001"},  // %v: 1e-05
		{1e-6, "0.000001"},    // lower edge is inclusive
		{9.99e-7, "9.99e-07"}, // just below: exponent, as before
		{1e-7, "1e-07"},       // unchanged from %v
		{math.NaN(), "NaN"},
		{math.Inf(1), "+Inf"},
		{math.Inf(-1), "-Inf"},
	} {
		if got := FloatText(tc.in, 64); got != tc.want {
			t.Errorf("FloatText(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// A float32 is written with float32 precision, so 0.1 is "0.1" rather than the
// widened "0.10000000149011612".
func TestFloatTextFloat32(t *testing.T) {
	if got := ValueText(float32(0.1)); got != "0.1" {
		t.Errorf("ValueText(float32(0.1)) = %q, want %q", got, "0.1")
	}
	if got := ValueText(float32(2000000)); got != "2000000" {
		t.Errorf("ValueText(float32(2e6)) = %q, want %q", got, "2000000")
	}
}

// ValueText only changes how floats are written. Everything else keeps the
// text fmt.Sprint gave it, nil included: OneHotEncode and Pivot name a nil
// category "<nil>" by documented design.
func TestValueTextLeavesOtherTypesAlone(t *testing.T) {
	for _, tc := range []struct {
		in   any
		want string
	}{
		{"abc", "abc"},
		{42, "42"},
		{int64(1500000), "1500000"},
		{true, "true"},
		{nil, "<nil>"},
		{1500000.0, "1500000"},
	} {
		if got := ValueText(tc.in); got != tc.want {
			t.Errorf("ValueText(%#v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
