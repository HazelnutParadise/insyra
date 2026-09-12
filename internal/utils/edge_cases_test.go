package utils

import (
	"testing"
	"time"
)

// IN-10, IN-12, IN-13 and IN-14 of #337. Each is a value that comes back wrong
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

// ConvertDateFormat replaced one pattern at a time over the whole string, so a
// literal letter — or a token longer than the map's keys — came out mangled.
func TestConvertDateFormat_DoesNotMangleLiterals(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "iso date", pattern: "YYYY-MM-DD", want: "2006-01-02"},
		{name: "with time", pattern: "YYYY-MM-DD HH:mm:ss", want: "2006-01-02 15:04:05"},
		{name: "slashes", pattern: "DD/MM/YYYY", want: "02/01/2006"},
		// "MMM" is a three-letter month; it used to become "011".
		{name: "three-letter month", pattern: "MMM DD", want: "Jan 02"},
		// Compact patterns keep working: each run is a token of its own.
		{name: "no separators", pattern: "YYYYMMDD", want: "20060102"},
		// Brackets are the way to keep letters out of the token scan. "Mon"
		// unescaped is a month token followed by two literal letters, which is
		// what any unescaped letter-token format does; "1on" was the old,
		// different breakage where the replacement ran over the whole string.
		{name: "escaped word", pattern: "[Mon] DD", want: "Mon 02"},
		{name: "escaped text", pattern: "[Date]: YYYY", want: "Date: 2006"},
		{name: "am pm", pattern: "hh:mm A", want: "15:04 PM"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ConvertDateFormat(tt.pattern)
			if got != tt.want {
				t.Errorf("ConvertDateFormat(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
			// The result has to be a layout Go can actually use.
			if _, err := time.Parse(got, time.Now().Format(got)); err != nil {
				t.Errorf("ConvertDateFormat(%q) = %q, which is not a usable layout: %v", tt.pattern, got, err)
			}
		})
	}
}
