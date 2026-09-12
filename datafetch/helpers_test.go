package datafetch

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// Two helpers that decide what a caller sees, neither of which needs the
// network. The rest of the package's uncovered half is the Google Maps
// crawler (#249, whose removal is still being decided) and the live HTTP
// paths.

func TestRateLimitError(t *testing.T) {
	t.Run("with a reset time", func(t *testing.T) {
		reset := time.Date(2026, 9, 12, 10, 30, 0, 0, time.UTC)
		err := &RateLimitError{Limit: 100, Remaining: 0, ResetAt: reset}

		msg := err.Error()
		for _, want := range []string{"100", "rate limited", reset.Format(time.RFC3339)} {
			if !strings.Contains(msg, want) {
				t.Errorf("the message %q does not mention %q", msg, want)
			}
		}
	})

	// Without the header the message must not print a zero time, which would
	// read as "resets at year 1".
	t.Run("without a reset time", func(t *testing.T) {
		err := &RateLimitError{Limit: 60, Remaining: 3}

		msg := err.Error()
		if strings.Contains(msg, "resets at") {
			t.Errorf("the message %q claims a reset time it does not have", msg)
		}
		for _, want := range []string{"60", "3", "rate limited"} {
			if !strings.Contains(msg, want) {
				t.Errorf("the message %q does not mention %q", msg, want)
			}
		}
	})

	// The documented way to detect it: errors.Is against the sentinel, so a
	// caller can back off without type-asserting.
	t.Run("matches the sentinel", func(t *testing.T) {
		var err error = &RateLimitError{Limit: 1}
		if !errors.Is(err, ErrGeocodeRateLimited) {
			t.Error("errors.Is did not match ErrGeocodeRateLimited")
		}
		var rle *RateLimitError
		if !errors.As(err, &rle) || rle.Limit != 1 {
			t.Error("errors.As did not recover the details")
		}
	})
}

// normalizeDateColumns decides by the last word of the column name, so that a
// column called "notadate" is left alone while "trade_date" and "expiryDate"
// are converted.
func TestNormalizeDateColumns(t *testing.T) {
	const stamp = "2026-09-12 10:30:00"

	tests := []struct {
		name      string
		colName   string
		converted bool
	}{
		{name: "exactly date", colName: "date", converted: true},
		{name: "exactly time", colName: "Time", converted: true},
		{name: "snake case", colName: "trade_date", converted: true},
		{name: "camel case", colName: "expiryDate", converted: true},
		{name: "expiry", colName: "option_expiry", converted: true},
		{name: "expire", colName: "contractExpire", converted: true},
		// The whole point of taking the last token: these only look like dates.
		{name: "a word ending in date", colName: "notadate", converted: false},
		{name: "a word ending in time", colName: "downtime", converted: false},
		{name: "unrelated", colName: "close", converted: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := insyra.NewDataTable(insyra.NewDataList(stamp).SetName(tt.colName))

			got := normalizeDateColumns(dt).GetColByNumber(0).Get(0)

			if tt.converted {
				if _, ok := got.(time.Time); !ok {
					t.Errorf("column %q: got %v (%T), want a time.Time", tt.colName, got, got)
				}
			} else if got != stamp {
				t.Errorf("column %q: got %v (%T), want the string unchanged", tt.colName, got, got)
			}
		})
	}
}

// A cell in a date column that is not a date is left as it is rather than
// replaced with a zero time.
func TestNormalizeDateColumns_UnparseableCell(t *testing.T) {
	dt := insyra.NewDataTable(insyra.NewDataList("not a date", 42).SetName("date"))

	col := normalizeDateColumns(dt).GetColByNumber(0)

	if got := col.Get(0); got != "not a date" {
		t.Errorf("an unparseable string became %v (%T)", got, got)
	}
	if got := col.Get(1); got != 42 {
		t.Errorf("a non-string cell became %v (%T)", got, got)
	}
}

func TestSleepBackoff(t *testing.T) {
	// A zero or negative backoff means no waiting at all.
	y := &yahooFinance{cfg: YFinanceConfig{RetryBackoff: 0}}
	start := time.Now()
	y.sleepBackoff(3)
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("a zero backoff waited %v", elapsed)
	}

	// Otherwise the wait grows with the attempt number.
	y = &yahooFinance{cfg: YFinanceConfig{RetryBackoff: 20 * time.Millisecond}}
	start = time.Now()
	y.sleepBackoff(1) // the second attempt waits two units
	if elapsed := time.Since(start); elapsed < 40*time.Millisecond {
		t.Errorf("attempt 1 waited %v, want at least two backoff units", elapsed)
	}
}
