package utils

import (
	"math"
	"strings"
	"testing"
	"time"
)

// utils_test.go and text_test.go cover FloatText, ValueText, CalcColIndex,
// TryParseTime and ConvertDateFormat. The conversion, truncation and display
// helpers below were at 0%, and they decide what every Show() prints and what
// every numeric read returns for a value it cannot read.

func TestToFloat64_EveryNumericType(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want float64
	}{
		{name: "int", in: int(-3), want: -3},
		{name: "int8", in: int8(-8), want: -8},
		{name: "int16", in: int16(-16), want: -16},
		{name: "int32", in: int32(-32), want: -32},
		{name: "int64", in: int64(-64), want: -64},
		{name: "uint", in: uint(3), want: 3},
		{name: "uint8", in: uint8(8), want: 8},
		{name: "uint16", in: uint16(16), want: 16},
		{name: "uint32", in: uint32(32), want: 32},
		{name: "uint64", in: uint64(64), want: 64},
		{name: "float32", in: float32(1.5), want: 1.5},
		{name: "float64", in: 2.25, want: 2.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToFloat64(tt.in); got != tt.want {
				t.Errorf("ToFloat64(%v) = %v, want %v", tt.in, got, tt.want)
			}
			got, ok := ToFloat64Safe(tt.in)
			if !ok {
				t.Errorf("ToFloat64Safe(%v) reported failure", tt.in)
			}
			if got != tt.want {
				t.Errorf("ToFloat64Safe(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// The pair's whole point: ToFloat64 answers 0 for something it cannot read,
// indistinguishable from a real zero, and ToFloat64Safe is the guarded twin.
// A type list that drifts between the two would show up here.
func TestToFloat64_UnreadableValues(t *testing.T) {
	for _, v := range []any{nil, "3", true, []int{1}, struct{}{}, time.Time{}} {
		if got := ToFloat64(v); got != 0 {
			t.Errorf("ToFloat64(%#v) = %v, want 0", v, got)
		}
		got, ok := ToFloat64Safe(v)
		if ok {
			t.Errorf("ToFloat64Safe(%#v) reported success", v)
		}
		if got != 0 {
			t.Errorf("ToFloat64Safe(%#v) = %v, want 0", v, got)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		maxLength int
		want      string
	}{
		{name: "fits", s: "hello", maxLength: 10, want: "hello"},
		{name: "exactly fits", s: "hello", maxLength: 5, want: "hello"},
		{name: "ellipsis", s: "hello world", maxLength: 8, want: "hello..."},
		{name: "tiny limit cuts by rune", s: "hello", maxLength: 3, want: "hel"},
		{name: "limit of one", s: "hello", maxLength: 1, want: "h"},
		{name: "limit of zero", s: "hello", maxLength: 0, want: ""},
		{name: "empty string", s: "", maxLength: 5, want: ""},
		// Width, not rune count: each of these is two columns wide.
		{name: "wide runes counted by width", s: "日本語テスト", maxLength: 9, want: "日本語..."},
		{name: "wide runes below the ellipsis", s: "日本語テスト", maxLength: 4, want: "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateString(tt.s, tt.maxLength); got != tt.want {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.s, tt.maxLength, got, tt.want)
			}
		})
	}
}

func TestFormatValue_Numbers(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "nil", in: nil, want: "nil"},
		{name: "NaN", in: math.NaN(), want: "NaN"},
		{name: "positive infinity", in: math.Inf(1), want: "+Inf"},
		{name: "negative infinity", in: math.Inf(-1), want: "-Inf"},
		{name: "integral float", in: 42.0, want: "42"},
		{name: "negative integral float", in: -42.0, want: "-42"},
		{name: "fraction", in: 3.5, want: "3.5"},
		{name: "trailing zeros trimmed", in: 3.2500, want: "3.25"},
		{name: "rounded to four places", in: 1.234567, want: "1.2346"},
		// The thresholds that switch to exponent form.
		{name: "just under the small threshold", in: 0.00009, want: "9.0000e-05"},
		{name: "at the small threshold", in: 0.0001, want: "0.0001"},
		{name: "just under the large threshold", in: 9999.5, want: "9999.5"},
		{name: "at the large threshold", in: 10000.5, want: "1.0000e+04"},
		{name: "float32", in: float32(1.5), want: "1.5"},
		{name: "int", in: 7, want: "7"},
		{name: "int64", in: int64(-7), want: "-7"},
		{name: "uint8", in: uint8(255), want: "255"},
		{name: "true", in: true, want: "true"},
		{name: "false", in: false, want: "false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatValue(tt.in); got != tt.want {
				t.Errorf("FormatValue(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatValue_StringsBytesAndTime(t *testing.T) {
	if got, want := FormatValue("hi"), "'hi'"; got != want {
		t.Errorf("FormatValue(\"hi\") = %q, want %q", got, want)
	}
	// A multi-line string shows its first line only, so one cell cannot break
	// the table's layout.
	if got, want := FormatValue("first\nsecond"), "'first...'"; got != want {
		t.Errorf("FormatValue of a multi-line string = %q, want %q", got, want)
	}

	if got, want := FormatValue([]byte{0xde, 0xad}), "dead"; got != want {
		t.Errorf("FormatValue of a short byte slice = %q, want %q", got, want)
	}
	long := make([]byte, 21)
	got := FormatValue(long)
	if !strings.HasSuffix(got, "... (21 bytes)") {
		t.Errorf("FormatValue of a 21-byte slice = %q, want it truncated with a byte count", got)
	}
	if !strings.HasPrefix(got, "00000000000000000000") {
		t.Errorf("FormatValue of a 21-byte slice = %q, want the first ten bytes in hex", got)
	}

	when := time.Date(2026, 9, 12, 13, 45, 6, 0, time.UTC)
	if got, want := FormatValue(when), "2026-09-12 13:45:06"; got != want {
		t.Errorf("FormatValue of a time = %q, want %q", got, want)
	}
}

func TestFormatValue_Containers(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "empty slice", in: []int{}, want: "[]"},
		{name: "three elements", in: []int{1, 2, 3}, want: "[1, 2, 3]"},
		{name: "more than three", in: []int{1, 2, 3, 4, 5}, want: "[1, 2, ... +3]"},
		{name: "array", in: [2]string{"a", "b"}, want: "[a, b]"},
		{name: "empty map", in: map[string]int{}, want: "{}"},
		{name: "map", in: map[string]int{"a": 1, "b": 2}, want: "{...2 keys}"},
		{name: "struct", in: struct{ A int }{A: 1}, want: "<struct { A int }>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatValue(tt.in); got != tt.want {
				t.Errorf("FormatValue(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestConvertToDateString(t *testing.T) {
	const layout = "2006-01-02"

	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "time.Time", in: time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC), want: "2026-09-12"},
		// Below 100000 an integer is read as an Excel serial: day 1 is 1899-12-31.
		{name: "excel serial as int", in: 1, want: "1899-12-31"},
		{name: "excel serial as int32", in: int32(46277), want: "2026-09-12"},
		{name: "excel serial as float", in: 46277.5, want: "2026-09-12"},
		// At or above 100000 an integer is a Unix timestamp.
		{name: "unix seconds", in: int64(1789000000), want: "2026-09-10"},
		{name: "unix milliseconds", in: int64(1789000000000), want: "2026-09-10"},
		{name: "unix nanoseconds", in: int64(1789000000000000000), want: "2026-09-10"},
		// A float outside the Excel-serial window is not a date at all.
		{name: "float too large", in: 1e6, want: "1e+06"},
		{name: "float zero", in: 0.0, want: "0"},
		{name: "string passes through", in: "not a date", want: "not a date"},
		{name: "anything else", in: true, want: "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToDateString(tt.in, layout); got != tt.want {
				t.Errorf("ConvertToDateString(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// The boundaries between the four timestamp readings, which are chosen purely
// by magnitude. Each pair straddles one boundary.
func TestConvertToDateString_TimestampBoundaries(t *testing.T) {
	const layout = "2006-01-02 15:04:05"

	tests := []struct {
		name string
		ts   int64
		want string
	}{
		{name: "last excel serial", ts: 99999, want: "2173-10-13 00:00:00"},
		{name: "first unix second", ts: 100000, want: "1970-01-02 03:46:40"},
		{name: "last unix second", ts: 999999999999, want: "33658-09-27 01:46:39"},
		{name: "first millisecond", ts: 1000000000000, want: "2001-09-09 01:46:40"},
		// Between the millisecond and nanosecond windows it falls back to reading
		// the number as seconds, which is why this is a year and not a date near
		// the one above it.
		{name: "between the windows", ts: 100000000000000, want: "3170843-11-07 09:46:40"},
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

// A column reference long enough to overflow an int is refused rather than
// wrapping into a negative index.
func TestParseColIndex_Overflow(t *testing.T) {
	if got, ok := ParseColIndex(strings.Repeat("A", 40)); ok || got != -1 {
		t.Errorf("a 40-letter reference gave (%d, %v), want (-1, false)", got, ok)
	}
	// The rejection is about the value, not the length: a long-but-valid one works.
	if got, ok := ParseColIndex("AAA"); !ok || got != 26*26+26+1-1 {
		t.Errorf("ParseColIndex(\"AAA\") = (%d, %v)", got, ok)
	}
}
