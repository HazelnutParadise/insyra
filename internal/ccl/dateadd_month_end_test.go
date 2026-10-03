package ccl

import (
	"testing"
	"time"
)

// CCL-33 (#367): DATEADD by months or years went through time.AddDate, which
// carries a day the target month does not have into the next month, so
// DATEADD('2024-01-31', 1, 'month') was 2024-03-02. Excel's EDATE and pandas'
// DateOffset keep the day where the month has it and stop at the month's last
// day otherwise. Every expected value below is what pandas 2.3.3 gives for
// Timestamp(d) + DateOffset(months=n) or DateOffset(years=n).
func TestDateAddStopsAtTheEndOfTheMonth(t *testing.T) {
	at := func(s string) time.Time {
		v, ok := toTime(s)
		if !ok {
			t.Fatalf("cannot parse %s", s)
		}
		return v
	}
	for _, c := range []struct {
		d    string
		n    float64
		unit string
		want string
	}{
		{"2024-01-31", 1, "month", "2024-02-29"},
		{"2023-01-31", 1, "month", "2023-02-28"},
		{"2024-01-30", 1, "month", "2024-02-29"},
		{"2024-03-31", -1, "month", "2024-02-29"},
		{"2024-01-31", 13, "months", "2025-02-28"},
		{"2024-12-31", 2, "month", "2025-02-28"},
		{"2024-02-29", -12, "month", "2023-02-28"},
		{"2024-08-31", -6, "month", "2024-02-29"},
		{"2024-01-15", 1, "month", "2024-02-15"},
		{"2024-05-31 10:30:15", 1, "month", "2024-06-30 10:30:15"},
		{"2024-02-29", 1, "year", "2025-02-28"},
		{"2024-02-29", 4, "years", "2028-02-29"},
		{"2024-02-29", -1, "year", "2023-02-28"},
		// A fraction still truncates, as it always has.
		{"2024-01-31", 1.9, "month", "2024-02-29"},
		// Days are not months: 31 days after January 31 is March 2.
		{"2024-01-31", 31, "day", "2024-03-02"},
	} {
		got, err := callFn(t, "DATEADD", c.d, c.n, c.unit)
		if err != nil {
			t.Errorf("DATEADD(%s, %v, %s): %v", c.d, c.n, c.unit, err)
			continue
		}
		if !got.(time.Time).Equal(at(c.want)) {
			t.Errorf("DATEADD(%s, %v, %s) = %v, want %s", c.d, c.n, c.unit, got, c.want)
		}
	}

	// The time zone and the clock time are kept.
	taipei := time.FixedZone("UTC+8", 8*3600)
	d := time.Date(2024, 1, 31, 23, 45, 0, 0, taipei)
	got, err := callFn(t, "DATEADD", d, 1.0, "month")
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2024, 2, 29, 23, 45, 0, 0, taipei); !got.(time.Time).Equal(want) || got.(time.Time).Location() != taipei {
		t.Errorf("DATEADD(%v, 1, month) = %v, want %v", d, got, want)
	}
}
