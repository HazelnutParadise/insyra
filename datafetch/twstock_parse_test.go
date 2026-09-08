package datafetch

import (
	"strings"
	"testing"
	"time"
)

func TestParseROCDate(t *testing.T) {
	for _, test := range []struct {
		input string
		want  time.Time
	}{
		{input: "115/09/01", want: time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)},
		{input: "1150903", want: time.Date(2026, time.September, 3, 0, 0, 0, 0, time.UTC)},
		{input: "115年06月01日", want: time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)},
	} {
		got, err := parseROCDate(test.input)
		if err != nil {
			t.Fatalf("parseROCDate(%q) error: %v", test.input, err)
		}
		if !got.Equal(test.want) {
			t.Errorf("parseROCDate(%q) = %v, want %v", test.input, got, test.want)
		}
	}
}

func TestParseROCDateLongFormatRejectsInvalid(t *testing.T) {
	for _, input := range []string{"115年13月01日", "115年06月31日", "115年06月01", "115年06月", "年06月01日", "115年ab月01日"} {
		if got, err := parseROCDate(input); err == nil {
			t.Errorf("parseROCDate(%q) = %v, want error", input, got)
		}
	}
}

func TestExRightsKind(t *testing.T) {
	for input, want := range map[string]string{"息": "dividend", "權": "rights", "權息": "both", " 息 ": "dividend"} {
		if got := exRightsKind(input); got != want {
			t.Errorf("exRightsKind(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseNumber(t *testing.T) {
	tests := []struct {
		input string
		want  float64
		ok    bool
	}{
		{input: "1,234.50", want: 1234.5, ok: true},
		{input: "--", ok: false},
		{input: "X", ok: false},
		{input: "", ok: false},
		{input: "-1,234", want: -1234, ok: true},
	}
	for _, tt := range tests {
		got, ok := parseNumber(tt.input)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("parseNumber(%q) = (%v, %v), want (%v, %v)", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestMapHeadersMissingRequiredColumn(t *testing.T) {
	_, err := mapHeaders(
		[]string{"證券名稱", "三大法人買賣超股數"},
		institutionalHeaderAliases,
		[]string{"證券代號", "證券名稱", "三大法人買賣超股數"},
	)
	if err == nil || !strings.Contains(err.Error(), "證券代號") {
		t.Fatalf("mapHeaders error = %v, want missing 證券代號", err)
	}
}
