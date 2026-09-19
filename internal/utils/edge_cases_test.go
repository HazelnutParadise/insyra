package utils

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// IN-10, IN-12 and IN-13 of #337. Each is a value that comes back wrong
// with no sign that anything went awry.

// A 16-digit timestamp is microseconds. Reading it as seconds put a 2023 date
// in the year 53872.
func TestConvertToDateString_Microseconds(t *testing.T) {
	const layout = "2006-01-02 15:04:05"

	tests := []struct {
		name string
		ts   int64
		want string
	}{
		{name: "last millisecond", ts: 99999999999999, want: "5138-11-16 09:46:39"},
		{name: "first microsecond", ts: 1000000000000000, want: "2001-09-09 01:46:40"},
		{name: "a real microsecond stamp", ts: 1700000000000000, want: "2023-11-14 22:13:20"},
		{name: "last microsecond", ts: 999999999999999999, want: "33658-09-27 01:46:39"},
		{name: "first nanosecond", ts: 1000000000000000000, want: "2001-09-09 01:46:40"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToDateString(tt.ts, layout); got != tt.want {
				t.Errorf("ConvertToDateString(%d) = %q, want %q", tt.ts, got, tt.want)
			}
		})
	}
}

// Every index CalcColIndex can produce must read back through ParseColIndex.
// The overflow guard rejected the last step, so the largest indices were
// one-way.
func TestColIndexRoundTrip(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, n := range []int{0, 25, 26, 701, 702, 1000000, maxInt - 1, maxInt} {
		name, ok := CalcColIndex(n)
		if !ok {
			t.Errorf("CalcColIndex(%d) refused", n)
			continue
		}
		back, ok := ParseColIndex(name)
		if !ok {
			t.Errorf("CalcColIndex(%d) = %q, which ParseColIndex refuses", n, name)
			continue
		}
		if back != n {
			t.Errorf("round trip of %d: got %q, which reads back as %d", n, name, back)
		}
	}
}

// Anything ParseColIndex cannot represent is still refused.
func TestParseColIndex_StillRejectsOverflow(t *testing.T) {
	for _, s := range []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", "ZZZZZZZZZZZZZZZZZZZZ"} {
		if got, ok := ParseColIndex(s); ok {
			t.Errorf("ParseColIndex(%q) = %d, want a refusal", s, got)
		}
	}
}

// A value that is not a whole number must not print as one.
func TestFormatValue_NearlyWhole(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{in: 9999.99999, want: "9999.99999"},
		{in: 0.99999999, want: "0.99999999"},
		{in: 1.00000001, want: "1.00000001"},
		{in: 42.0, want: "42"},
		{in: 3.5, want: "3.5"},
	}
	for _, tt := range tests {
		if got := FormatValue(tt.in); got != tt.want {
			t.Errorf("FormatValue(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// floatTextRule is the rule 0.4's FloatText writes a float by: a plain decimal
// from 1e-6 up to 1e21, the exponent form outside. This line has no FloatText,
// so FormatValue's fallback uses strconv's plain decimal directly; the test
// below shows the two agree everywhere that fallback can be reached.
func floatTextRule(f float64) string {
	if abs := math.Abs(f); abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		return strconv.FormatFloat(f, 'e', -1, 64)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// The fallback is reached only for 0.0001 <= |v| < 10000 that is not whole and
// rounds to a whole number at four places, i.e. within 0.00005 of an integer
// from 1 to 10000. Walk every such integer from both sides.
func TestFormatValue_NearlyWholeFallbackMatchesFloatText(t *testing.T) {
	offsets := []float64{0.00005, 0.00004999, 0.00001, 0.000001, 1e-9, 1e-12}
	reached := 0
	for n := 1; n <= 10000; n++ {
		for _, sign := range []float64{1, -1} {
			for _, off := range offsets {
				for _, v := range []float64{sign * (float64(n) - off), sign * (float64(n) + off)} {
					if v == math.Trunc(v) || math.Abs(v) < 0.0001 || math.Abs(v) >= 10000 {
						continue
					}
					s := strconv.FormatFloat(v, 'f', 4, 64)
					s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
					if strings.Contains(s, ".") {
						continue // not the fallback
					}
					reached++
					if got, want := FormatValue(v), floatTextRule(v); got != want {
						t.Fatalf("FormatValue(%v) = %q, FloatText's rule gives %q", v, got, want)
					}
				}
			}
		}
	}
	if reached == 0 {
		t.Fatal("no value reached the fallback; the sampling is wrong")
	}
}
