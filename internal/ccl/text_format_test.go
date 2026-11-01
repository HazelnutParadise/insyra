package ccl

import (
	"math"
	"strings"
	"testing"
	"time"
)

// TEXT and FORMAT_DATE read a format the Go way or the Excel way.
func TestTextNumberFormats(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {1.5}, "B": {1234567.891}, "C": {-0.5}, "D": {int64(5)}, "E": {"1.5"}})
	for _, c := range []struct{ expr, want string }{
		{"TEXT(A, '%.2f')", "1.50"},
		{"TEXT(A, '0.00')", "1.50"},
		{"TEXT(A, '0')", "2"},
		{"TEXT(2.5, '0')", "3"},
		{"TEXT(1.005, '0.00')", "1.01"},
		{"TEXT(0.125, '0.00')", "0.13"},
		{"TEXT(A, '0.0%')", "150.0%"},
		{"TEXT(0.07, '0%')", "7%"},
		{"TEXT(B, '#,##0.00')", "1,234,567.89"},
		{"TEXT(B, '#,##0')", "1,234,568"},
		{"TEXT(999.5, '#,##0')", "1,000"},
		{"TEXT(A, '$0.00')", "$1.50"},
		{"TEXT(C, '$0.00')", "-$0.50"},
		{"TEXT(-0.001, '0.00')", "0.00"},
		{"TEXT(A, '#.##')", "1.5"},
		{"TEXT(1, '#.00')", "1.00"},
		{"TEXT(0.5, '#.00')", ".50"},
		{"TEXT(5, '000')", "005"},
		{"TEXT(A, '0.00 \"kg\"')", "1.50 kg"},
		{"TEXT(D, '0.00')", "5.00"},
		{"TEXT(D, '%.2f')", "5.00"},
		{"TEXT(E, '0.00')", "1.50"},
		{"TEXT(1e21, '0')", "1000000000000000000000"},
		{"TEXT(A, '0.00 \"v2\"')", "1.50 v2"},
		{"TEXT(9007199254740993, '0')", "9007199254740993"},
		{"TEXT(9007199254740993, '#,##0')", "9,007,199,254,740,993"},
		{"TEXT('9007199254740993', '0')", "9007199254740993"},
		{"TEXT(-5, '0.0')", "-5.0"},
		{"TEXT(A, '%[1]f')", "1.500000"},
		{"TEXT(A, '%.f')", "2"},
		{"TEXT(7, '%[1]d')", "7"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil || got != c.want {
			t.Errorf("%s = %#v, %v; want %q", c.expr, got, err, c.want)
		}
	}
}

func TestTextDateFormats(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {time.Date(2024, 3, 9, 14, 5, 7, 123_000_000, time.UTC)},
		"B": {"2024-03-09T02:05:07Z"},
	})
	for _, c := range []struct{ expr, want string }{
		{"TEXT(A, 'yyyy-mm-dd')", "2024-03-09"},
		{"FORMAT_DATE(A, 'yyyy-mm-dd')", "2024-03-09"},
		{"FORMAT_DATE(A, '2006-01-02')", "2024-03-09"},
		{"TEXT(A, '2006-01-02')", "2024-03-09"},
		{"TEXT(A, 'hh:mm')", "14:05"},
		{"FORMAT_DATE(A, 'hh:mm')", "14:05"},
		{"FORMAT_DATE(A, '15:04')", "14:05"},
		{"TEXT(A, '15:04')", "14:05"},
		{"TEXT(A, 'YYYY/MM/DD')", "2024/03/09"},
		{"TEXT(A, 'yy-m-d')", "24-3-9"},
		{"TEXT(A, 'yyyy年mm月dd日')", "2024年03月09日"},
		{"TEXT(A, 'mmm mmmm')", "Mar March"},
		{"TEXT(A, 'ddd dddd')", "Sat Saturday"},
		{"TEXT(A, 'hh:mm:ss')", "14:05:07"},
		{"TEXT(A, 'mm:ss')", "05:07"},
		{"TEXT(A, 'ss.000')", "07.123"},
		{"TEXT(A, 'h:mm AM/PM')", "2:05 PM"},
		{"TEXT(B, 'h:mm AM/PM')", "2:05 AM"},
		{"TEXT(B, 'hh:mm')", "02:05"},
		{"TEXT(A, 'h:mm a/p')", "2:05 p"},
		{"TEXT(A, 'yyyy \"Q\" m')", "2024 Q 3"},
		{"FORMAT_DATE(A, 'yyyy-mm-dd \"Q1\"')", "2024-03-09 Q1"},
		{"FORMAT_DATE(A, 'Monday, pm')", "Saturday, pm"},
		{"FORMAT_DATE(A, 'Monday, PM')", "Saturday, PM"},
		{"FORMAT_DATE(A, 'Mon')", "Sat"},
		{"FORMAT_DATE(A, 'Monday')", "Saturday"},
		{"FORMAT_DATE(A, 'Jan')", "Mar"},
		{"FORMAT_DATE(A, 'January')", "March"},
		{"FORMAT_DATE(A, 'Mon, 02 Jan 2006')", "Sat, 09 Mar 2024"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil || got != c.want {
			t.Errorf("%s = %#v, %v; want %q", c.expr, got, err, c.want)
		}
	}
}

// A format that cannot be read is an error naming it, never its own text.
func TestFormatsThatCannotBeRead(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{
		"A": {time.Date(2024, 3, 9, 14, 5, 7, 0, time.UTC)},
		"B": {math.NaN()},
	})
	for _, c := range []struct{ expr, want string }{
		{"FORMAT_DATE(A, 'yyyy-qq')", "qq"},
		{"FORMAT_DATE(A, 'mon')", "date code"},
		{"FORMAT_DATE(A, 'date')", "date code"},
		{"FORMAT_DATE(A, '-')", "no date code"},
		{"FORMAT_DATE(A, 'yyyy\"')", "quote"},
		{"TEXT(1.5, 'yyyy')", "not a date"},
		{"TEXT(1.5, '2006')", "not a date"},
		{"TEXT(A, '0.00')", "not a number"},
		{"TEXT(true, '0.00')", "not a number"},
		{"TEXT('abc', '0.00')", "not a number"},
		{"TEXT(1, '%')", "neither"},
		{"TEXT(1, '$')", "neither"},
		{"TEXT(1, '0;0')", "section"},
		{"TEXT(1, '#,##0,')", "scaling"},
		{"TEXT(1, '0.0.0')", "second point"},
		{"TEXT(B, '0.00')", "NaN"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err == nil {
			t.Errorf("%s = %#v, want an error", c.expr, got)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: the error %q does not mention %q", c.expr, err, c.want)
		}
	}
}

// One empty cell does not fail a column under an Excel format.
func TestTextLeavesAnEmptyCellEmpty(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {nil}})
	for _, expr := range []string{"TEXT(A, '0.00')", "TEXT(A, 'yyyy-mm-dd')"} {
		if got, err := evalOne(t, ctx, expr); err != nil || got != nil {
			t.Errorf("%s = %#v, %v; want nil and no error", expr, got, err)
		}
	}
}

// A % inside quoted text is that text, not a Go verb and not a percent format.
func TestQuotedPercentIsLiteralText(t *testing.T) {
	ctx := mapCtx(t, map[string][]any{"A": {1.0}})
	for _, c := range []struct{ expr, want string }{
		{`TEXT(5, '0 "%off"')`, "5 %off"},
		{`TEXT(1.5, '0.00 "% savings"')`, "1.50 % savings"},
		{`TEXT(1.5, '"%.2f"')`, `"1.50"`},
		{`TEXT(1.5, '%.2f')`, "1.50"},
		{`TEXT(5, '%d items')`, "5 items"},
		{`TEXT(9007199254740993, '0')`, "9007199254740993"},
	} {
		got, err := evalOne(t, ctx, c.expr)
		if err != nil {
			t.Errorf("%s: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s = %#v, want %q", c.expr, got, c.want)
		}
	}
}

func TestFormatKindPrintsItsName(t *testing.T) {
	if got := kindDate.String(); got != "kindDate" {
		t.Errorf("kindDate prints %q", got)
	}
	if got := formatKind(99).String(); got != "formatKind(99)" {
		t.Errorf("an unknown kind prints %q", got)
	}
}

func TestFormatEdges(t *testing.T) {
	stamp := time.Date(2024, 3, 9, 14, 5, 6, 123_000_000, time.UTC)
	ancient := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		value  time.Time
		format string
		want   string
	}{
		{stamp, ".000", ".123"},
		{stamp, ",000000", ",123000"},
		{stamp, "ss.000", "06.123"},
		{ancient, "yyyy-mm-dd", "0001-01-01"},
	} {
		got, err := formatDateValue(c.value, c.format)
		if err != nil {
			t.Errorf("%q: %v", c.format, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %q, want %q", c.format, got, c.want)
		}
	}
	// Literal text costs no more than its length.
	long := "yyyy" + strings.Repeat("-", 80000)
	allocs := testing.AllocsPerRun(1, func() {
		if _, err := formatDateValue(stamp, long); err != nil {
			t.Fatal(err)
		}
	})
	if allocs > 200 {
		t.Errorf("a long literal took %v allocations", allocs)
	}
}
