package finance

import (
	"errors"
	"time"

	"github.com/TimLai666/go-decimal/decimal"
)

// DayCountBasis selects how day counts and year lengths are computed
// in bond and ACCRINT-style functions. Values match Excel's "basis"
// parameter exactly.
type DayCountBasis uint8

const (
	// Basis30_360US is the US (NASD) 30/360 convention. Excel basis = 0.
	Basis30_360US DayCountBasis = 0
	// BasisActualActual uses calendar days for both numerator and a
	// year length derived from the years actually spanned by the
	// period (ISDA actual/actual). Excel basis = 1.
	BasisActualActual DayCountBasis = 1
	// BasisActual360 uses calendar days over a 360-day year — common
	// for money-market instruments. Excel basis = 2.
	BasisActual360 DayCountBasis = 2
	// BasisActual365 uses calendar days over a fixed 365-day year.
	// Excel basis = 3.
	BasisActual365 DayCountBasis = 3
	// Basis30_360EU is the European 30/360 convention. Excel basis = 4.
	Basis30_360EU DayCountBasis = 4
)

// validateBasis checks that b is one of the five recognized day-count
// conventions.
func validateBasis(b DayCountBasis) error {
	if b > Basis30_360EU {
		return errors.New("basis must be one of 0..4")
	}
	return nil
}

// toDateUTC strips the time-of-day from t and returns the resulting
// midnight in UTC. Day-count math always operates on whole days, so
// pinning the time-zone here avoids DST-induced off-by-one errors.
func toDateUTC(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// calendarDays returns the signed difference d2 - d1 in whole days.
func calendarDays(d1, d2 time.Time) int {
	return int(toDateUTC(d2).Sub(toDateUTC(d1)) / (24 * time.Hour))
}

// isLeapYear reports whether y is a Gregorian leap year.
func isLeapYear(y int) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}

// daysInMonth returns the number of days in month m of year y.
func daysInMonth(y int, m time.Month) int {
	// Trick: day-0 of month (m+1) is the last day of month m, normalized
	// by time.Date.
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// isLastDayOfFeb reports whether t falls on Feb 28 (non-leap) or Feb 29.
func isLastDayOfFeb(t time.Time) bool {
	y, m, d := t.Date()
	if m != time.February {
		return false
	}
	return d == daysInMonth(y, time.February)
}

// days30360US returns the day count between d1 and d2 under the US
// (NASD) 30/360 rule. The four conditional substitutions are applied
// in order, exactly as Excel's PRICE/COUPDAYS family does.
func days30360US(d1, d2 time.Time) int {
	y1, m1, day1 := d1.Date()
	y2, m2, day2 := d2.Date()

	last1 := isLastDayOfFeb(d1)
	last2 := isLastDayOfFeb(d2)

	if last1 && last2 {
		day2 = 30
	}
	if last1 {
		day1 = 30
	}
	if day1 == 31 {
		day1 = 30
	}
	if day2 == 31 && day1 == 30 {
		day2 = 30
	}
	return 360*(y2-y1) + 30*(int(m2)-int(m1)) + (day2 - day1)
}

// days30360EU returns the day count between d1 and d2 under the
// European 30/360 rule.
func days30360EU(d1, d2 time.Time) int {
	y1, m1, day1 := d1.Date()
	y2, m2, day2 := d2.Date()
	if day1 == 31 {
		day1 = 30
	}
	if day2 == 31 {
		day2 = 30
	}
	return 360*(y2-y1) + 30*(int(m2)-int(m1)) + (day2 - day1)
}

// dayDiff returns the day count between d1 and d2 under the given
// basis. The result is signed when d2 < d1.
func dayDiff(d1, d2 time.Time, basis DayCountBasis) int {
	switch basis {
	case Basis30_360US:
		return days30360US(d1, d2)
	case Basis30_360EU:
		return days30360EU(d1, d2)
	default:
		return calendarDays(d1, d2)
	}
}

// yearFraction returns (d2 - d1) expressed in years according to the
// given basis. For Actual/Actual it uses the ISDA convention: each
// year-segment of the period contributes daysInSegment / 365_or_366.
func yearFraction(ctx decimal.Context, d1, d2 time.Time, basis DayCountBasis) (decimal.Decimal, error) {
	if err := validateBasis(basis); err != nil {
		return decimal.Decimal{}, err
	}
	switch basis {
	case Basis30_360US, Basis30_360EU:
		days := dayDiff(d1, d2, basis)
		return decimal.Div(ctx, decimal.NewFromInt64(ctx, int64(days)),
			decimal.NewFromInt64(ctx, 360))
	case BasisActual360:
		return decimal.Div(ctx, decimal.NewFromInt64(ctx, int64(calendarDays(d1, d2))),
			decimal.NewFromInt64(ctx, 360))
	case BasisActual365:
		return decimal.Div(ctx, decimal.NewFromInt64(ctx, int64(calendarDays(d1, d2))),
			decimal.NewFromInt64(ctx, 365))
	case BasisActualActual:
		return actActYearFraction(ctx, d1, d2)
	}
	return decimal.Decimal{}, errors.New("unsupported basis")
}

// actActYearFraction implements ISDA actual/actual: split the period
// at calendar-year boundaries and for each year y inside the period
// add (days in [start, end] ∩ year y) / (days in year y).
func actActYearFraction(ctx decimal.Context, d1, d2 time.Time) (decimal.Decimal, error) {
	sign := 1
	if d1.After(d2) {
		d1, d2 = d2, d1
		sign = -1
	}
	d1 = toDateUTC(d1)
	d2 = toDateUTC(d2)

	yf := decimal.NewFromInt64(ctx, 0)
	for y := d1.Year(); y <= d2.Year(); y++ {
		segStart := time.Date(y, time.January, 1, 0, 0, 0, 0, time.UTC)
		if y == d1.Year() {
			segStart = d1
		}
		segEnd := time.Date(y+1, time.January, 1, 0, 0, 0, 0, time.UTC)
		if y == d2.Year() {
			segEnd = d2
		}
		days := calendarDays(segStart, segEnd)
		if days <= 0 {
			continue
		}
		yearLen := 365
		if isLeapYear(y) {
			yearLen = 366
		}
		contrib, err := decimal.Div(ctx,
			decimal.NewFromInt64(ctx, int64(days)),
			decimal.NewFromInt64(ctx, int64(yearLen)))
		if err != nil {
			return decimal.Decimal{}, err
		}
		yf = decimal.Add(ctx, yf, contrib)
	}
	if sign < 0 {
		yf = decimal.Neg(yf)
	}
	return yf, nil
}

// addMonths shifts t forward by m calendar months. If the resulting
// month has fewer days than t.Day(), the result is clamped to the
// last day of that month — matching the behavior bond coupon-date
// generators rely on (e.g., Aug 31 + 1 month = Sep 30).
func addMonths(t time.Time, m int) time.Time {
	y, mo, d := t.Date()
	t2 := time.Date(y, mo+time.Month(m), 1, 0, 0, 0, 0, time.UTC)
	ny, nm, _ := t2.Date()
	last := daysInMonth(ny, nm)
	if d > last {
		d = last
	}
	return time.Date(ny, nm, d, 0, 0, 0, 0, time.UTC)
}
