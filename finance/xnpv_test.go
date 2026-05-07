package finance

import (
	"testing"
	"time"

	"github.com/TimLai666/go-decimal/decimal"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestXNPV_BasicMatchesExcelDoc(t *testing.T) {
	// Excel doc example:
	// values: -10000, 2750, 4250, 3250, 2750
	// dates:  1/1/2008, 3/1/2008, 10/30/2008, 2/15/2009, 4/1/2009
	// XNPV(0.09, ...) ≈ 2086.6476 (Excel's float-based answer).
	values := []decimal.Decimal{
		mustDec("-10000"), mustDec("2750"), mustDec("4250"),
		mustDec("3250"), mustDec("2750"),
	}
	dates := []time.Time{
		date(2008, 1, 1),
		date(2008, 3, 1),
		date(2008, 10, 30),
		date(2009, 2, 15),
		date(2009, 4, 1),
	}
	got, err := XNPV(mustDec("0.09"), values, dates, Options{Scale: 6})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("2086.6476")
	tol := mustDec("0.001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("XNPV got=%s want≈%s", got.String(), want.String())
	}
}

func TestXIRR_RootCheck(t *testing.T) {
	// XNPV(XIRR(cf, dates), cf, dates) ≈ 0 — the defining property.
	values := []decimal.Decimal{
		mustDec("-10000"), mustDec("2750"), mustDec("4250"),
		mustDec("3250"), mustDec("2750"),
	}
	dates := []time.Time{
		date(2008, 1, 1),
		date(2008, 3, 1),
		date(2008, 10, 30),
		date(2009, 2, 15),
		date(2009, 4, 1),
	}
	rate, err := XIRR(values, dates, Zero, Options{Scale: 12})
	if err != nil {
		t.Fatal(err)
	}
	npv, err := XNPV(rate, values, dates, Options{Scale: 10})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.000001")
	if !approxEqual(t, npv, Zero, tol) {
		t.Fatalf("XNPV(XIRR=%s)=%s, want ≈ 0", rate.String(), npv.String())
	}
}

func TestXIRR_AnnualEquivalence(t *testing.T) {
	// Two cashflows exactly 365 days apart should give the same XIRR
	// as the equivalent IRR with 1 period.
	values := []decimal.Decimal{mustDec("-100"), mustDec("110")}
	dates := []time.Time{date(2024, 1, 1), date(2025, 1, 1)} // exactly 366 days
	// Wait: 2024 is a leap year, so 1/1/2024 to 1/1/2025 = 366 days.
	// Use 2023→2024 instead for 365 days.
	dates = []time.Time{date(2023, 1, 1), date(2024, 1, 1)}
	xirr, err := XIRR(values, dates, Zero, Options{Scale: 10})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("0.1")
	tol := mustDec("0.0000001")
	if !approxEqual(t, xirr, want, tol) {
		t.Fatalf("XIRR (1-yr) got=%s want=0.1", xirr.String())
	}
}

func TestXNPV_LengthMismatch(t *testing.T) {
	if _, err := XNPV(mustDec("0.05"),
		[]decimal.Decimal{mustDec("100"), mustDec("200")},
		[]time.Time{date(2024, 1, 1)}); err == nil {
		t.Fatal("expected error on length mismatch")
	}
}

func TestDayCount_30360US_Excel(t *testing.T) {
	// 1/15 → 5/15 of the same year is 4·30 = 120 with no edge-case
	// substitutions triggered.
	got := days30360US(date(2024, 1, 15), date(2024, 5, 15))
	if got != 120 {
		t.Fatalf("30/360 US 1/15→5/15 got=%d want=120", got)
	}
	// 1/31 → 5/31 — both endpoints are 31. After substitution both
	// become 30, so the result is 4·30 = 120.
	got = days30360US(date(2024, 1, 31), date(2024, 5, 31))
	if got != 120 {
		t.Fatalf("30/360 US 1/31→5/31 got=%d want=120", got)
	}
	// 2/28 (non-leap) → 8/31 — D1 is last-of-Feb so it becomes 30;
	// D2 = 31 with D1 now 30 also becomes 30. Result = 30·6 + 0 = 180.
	got = days30360US(date(2023, 2, 28), date(2023, 8, 31))
	if got != 180 {
		t.Fatalf("30/360 US 2/28→8/31 got=%d want=180", got)
	}
}

func TestActActYearFraction_OneCalendarYear(t *testing.T) {
	ctx := decimal.Context{Scale: 20, Mode: decimal.RoundingModeHalfUp}
	// Exactly Jan 1 -> Jan 1 next year, non-leap, gives 1.0.
	yf, err := actActYearFraction(ctx, date(2023, 1, 1), date(2024, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(t, yf, mustDec("1"), mustDec("0.0000000001")) {
		t.Fatalf("yf got=%s want=1", yf.String())
	}
	// Same range across a leap year contributes 366/366 = 1.0.
	yf, err = actActYearFraction(ctx, date(2024, 1, 1), date(2025, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(t, yf, mustDec("1"), mustDec("0.0000000001")) {
		t.Fatalf("leap yf got=%s want=1", yf.String())
	}
}
