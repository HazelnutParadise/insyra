package utils

import (
	"math"
	"testing"
)

// FormatValue decided whether a float was a whole number with
// v == float64(int(v)). int(v) is undefined in Go for a value outside int's
// range: amd64 gives MinInt64 and arm64 saturates to MaxInt64. For 2^63 that
// made a Mac print 9223372036854775807 — an exact-looking integer one less
// than the value — while Linux and Windows printed 9.2234e+18. This is
// Show()'s formatter, so the same table read differently depending on where it
// ran. Run this with GOARCH=amd64 as well as natively.
func TestFormatValue_LargeFloatsArePortable(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want string
	}{
		// Inside int64: still printed as an integer.
		{name: "largest exact int64", in: math.MaxInt64 - 1024, want: "9223372036854774784"},
		{name: "smallest int64", in: math.MinInt64, want: "-9223372036854775808"},
		{name: "a million", in: 1000000, want: "1000000"},
		// 2^63 is one past int64's range, so it cannot be printed as an integer.
		{name: "two to the sixty-three", in: 9223372036854775808, want: "9.2234e+18"},
		{name: "two to the sixty-four", in: 18446744073709551616, want: "1.8447e+19"},
		{name: "ten to the three hundred", in: 1e300, want: "1.0000e+300"},
		{name: "largest float64", in: math.MaxFloat64, want: "1.7977e+308"},
		{name: "negative beyond int64", in: -1e300, want: "-1.0000e+300"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatValue(tt.in); got != tt.want {
				t.Errorf("FormatValue(%g) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// The millisecond branch multiplied by time.Millisecond to get nanoseconds, and
// that overflows int64 above 9223372036854 ms (about 2262-04-11) — while the
// branch itself accepts values up to 10^14. Anything past the overflow point
// came back as a different date with no sign that anything was wrong.
func TestConvertToDateString_MillisecondsPastTwentyTwoSixtyTwo(t *testing.T) {
	const layout = "2006-01-02 15:04:05"

	tests := []struct {
		name string
		ts   int64
		want string
	}{
		{name: "first millisecond in the window", ts: 1000000000000, want: "2001-09-09 01:46:40"},
		{name: "just before the overflow", ts: 9223372036854, want: "2262-04-11 23:47:16"},
		{name: "just past the overflow", ts: 9223372036855, want: "2262-04-11 23:47:16"},
		{name: "last millisecond in the window", ts: 99999999999999, want: "5138-11-16 09:46:39"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToDateString(tt.ts, layout); got != tt.want {
				t.Errorf("ConvertToDateString(%d) = %q, want %q", tt.ts, got, tt.want)
			}
		})
	}
}
