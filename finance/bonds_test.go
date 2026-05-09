package finance

import (
	"testing"

	"github.com/TimLai666/go-decimal/decimal"
)

// Reference values below are Excel's documented bond examples. Excel
// uses float64 internally, so a tolerance of ~1e-4 is appropriate for
// price comparisons (price-per-100 already covers the float-error
// regime to 4 decimal places).

func TestPrice_ExcelDocExample(t *testing.T) {
	// Excel docs: PRICE(2/15/2008, 11/15/2017, 5.75%, 6.5%, 100, 2, 0)
	// → 94.63374961...
	got, err := Price(date(2008, 2, 15), date(2017, 11, 15),
		mustDec("0.0575"), mustDec("0.065"), mustDec("100"),
		2, Basis30_360US, Options{Scale: 6})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("94.633750")
	tol := mustDec("0.001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("PRICE got=%s want≈%s", got.String(), want.String())
	}
}

func TestYield_RoundTripFromPrice(t *testing.T) {
	settlement := date(2008, 2, 15)
	maturity := date(2017, 11, 15)
	rate := mustDec("0.0575")
	expectedYld := mustDec("0.065")

	pr, err := Price(settlement, maturity, rate, expectedYld,
		mustDec("100"), 2, Basis30_360US, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Yield(settlement, maturity, rate, pr, mustDec("100"),
		2, Basis30_360US, Zero, Options{Scale: 10})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.0000000001")
	if !approxEqual(t, got, expectedYld, tol) {
		t.Fatalf("YIELD got=%s want=%s", got.String(), expectedYld.String())
	}
}

func TestDuration_ExcelDocExample(t *testing.T) {
	// Excel: DURATION(1/1/2008, 1/1/2016, 8%, 9%, 2, 1) → 5.993775...
	got, err := Duration(date(2008, 1, 1), date(2016, 1, 1),
		mustDec("0.08"), mustDec("0.09"),
		2, BasisActualActual, Options{Scale: 6})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("5.993775")
	tol := mustDec("0.001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("DURATION got=%s want≈%s", got.String(), want.String())
	}
}

func TestMDuration_DerivedFromDuration(t *testing.T) {
	// MD = D / (1 + yld/freq) — verify by computing both and checking
	// the algebraic relationship.
	settlement := date(2008, 1, 1)
	maturity := date(2016, 1, 1)
	rate := mustDec("0.08")
	yld := mustDec("0.09")

	d, err := Duration(settlement, maturity, rate, yld, 2, BasisActualActual,
		Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	md, err := MDuration(settlement, maturity, rate, yld, 2, BasisActualActual,
		Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}

	work := decimal.Context{Scale: 14, Mode: decimal.RoundingModeHalfUp}
	one := decimal.NewFromInt64(work, 1)
	yldOverFreq, _ := decimal.Div(work, yld, decimal.NewFromInt64(work, 2))
	denom := decimal.Add(work, one, yldOverFreq)
	expected, _ := decimal.Div(work, d, denom)

	tol := mustDec("0.0000000001")
	if !approxEqual(t, md, expected, tol) {
		t.Fatalf("MDURATION=%s, D/(1+y/f)=%s", md.String(), expected.String())
	}
}

func TestAccrInt_ExcelDocExample(t *testing.T) {
	// Excel: ACCRINT(3/1/2008, 8/31/2008, 5/1/2008, 10%, 1000, 2, 0, TRUE)
	// 30/360: days(3/1 → 5/1) = 60. yf = 60/360 = 1/6.
	// ACCRINT = 1000 · 0.10 · 1/6 = 16.66667
	got, err := AccrInt(date(2008, 3, 1), date(2008, 8, 31), date(2008, 5, 1),
		mustDec("0.10"), mustDec("1000"),
		2, Basis30_360US, true, Options{Scale: 6})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("16.666667")
	tol := mustDec("0.000001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("ACCRINT got=%s want≈%s", got.String(), want.String())
	}
}

func TestAccrInt_CalcMethodFalseUsesFirstInterest(t *testing.T) {
	// With calcMethod=false, accrual starts at firstInterest. In this
	// case settlement happens *before* firstInterest, so accrual is
	// effectively negative (settlement is earlier than the start).
	// Not a real-world case, but verifies the start date selection.
	gotTrue, _ := AccrInt(date(2008, 3, 1), date(2008, 8, 31), date(2008, 5, 1),
		mustDec("0.10"), mustDec("1000"),
		2, Basis30_360US, true, Options{Scale: 8})
	gotFalse, _ := AccrInt(date(2008, 3, 1), date(2008, 8, 31), date(2008, 5, 1),
		mustDec("0.10"), mustDec("1000"),
		2, Basis30_360US, false, Options{Scale: 8})
	// gotTrue uses 3/1 as start (60 days to 5/1), gotFalse uses 8/31
	// as start (negative day count to 5/1). They must differ.
	if decimal.Cmp(gotTrue, gotFalse) == 0 {
		t.Fatalf("calcMethod true/false produced identical results: %s",
			gotTrue.String())
	}
}

func TestPrice_RejectsBadInputs(t *testing.T) {
	_, err := Price(date(2024, 1, 1), date(2023, 1, 1),
		mustDec("0.05"), mustDec("0.05"), mustDec("100"),
		2, Basis30_360US)
	if err == nil {
		t.Fatal("expected error for maturity before settlement")
	}
	_, err = Price(date(2023, 1, 1), date(2024, 1, 1),
		mustDec("0.05"), mustDec("0.05"), mustDec("100"),
		3, Basis30_360US) // freq=3 invalid
	if err == nil {
		t.Fatal("expected error for invalid frequency")
	}
}

func TestCoupHelpers(t *testing.T) {
	// Settlement 2/15/2008, maturity 11/15/2017, semi-annual.
	// Last coupon before settlement: 11/15/2007. Next: 5/15/2008.
	prev, next := coupPrevNext(date(2008, 2, 15), date(2017, 11, 15), 2)
	if prev.Year() != 2007 || prev.Month() != 11 || prev.Day() != 15 {
		t.Fatalf("prev coupon got=%s", prev.Format("2006-01-02"))
	}
	if next.Year() != 2008 || next.Month() != 5 || next.Day() != 15 {
		t.Fatalf("next coupon got=%s", next.Format("2006-01-02"))
	}
	// Number of remaining coupons: 5/15/08 → 11/15/17 inclusive = 20.
	if got := coupNum(date(2008, 2, 15), date(2017, 11, 15), 2); got != 20 {
		t.Fatalf("coupNum got=%d want=20", got)
	}
}
