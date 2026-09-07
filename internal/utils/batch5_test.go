package utils

import (
	"testing"
	"time"
)

// IN-6: the most common wire formats for a timestamp without a zone were not
// recognised, so CCL's date functions and datafetch silently saw a string.
func TestTryParseTimeCommonFormats(t *testing.T) {
	want := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	cases := map[string]time.Time{
		"2024-01-02 03:04:05":       want,
		"2024-01-02T03:04:05":       want,
		"2024-01-02 03:04":          time.Date(2024, 1, 2, 3, 4, 0, 0, time.UTC),
		"2024/01/02":                time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		"2024/01/02 03:04:05":       want,
		"2024-01-02":                time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		"2024-01-02T03:04:05Z":      want,
		"2024-01-02T03:04:05+00:00": want,
	}
	for in, expected := range cases {
		got, ok := TryParseTime(in)
		if !ok {
			t.Errorf("TryParseTime(%q) reported failure", in)
			continue
		}
		if !got.Equal(expected) {
			t.Errorf("TryParseTime(%q) = %v, want %v", in, got, expected)
		}
	}
}

// Things that are not dates must stay unparsed, or CCL would start treating
// ordinary strings as timestamps.
func TestTryParseTimeRejectsNonDates(t *testing.T) {
	for _, in := range []string{"", "hello", "12345", "3.14", "2024", "01-02", "not a date"} {
		if got, ok := TryParseTime(in); ok {
			t.Errorf("TryParseTime(%q) = %v, want no match", in, got)
		}
	}
}
