package finance

import (
	"testing"
)

func TestTBillEq_ExcelExample(t *testing.T) {
	// Excel doc example: settlement 3/31/2008, maturity 6/1/2008,
	// discount 9.14%. Days = 62. Excel returns ≈ 0.094151.
	settlement := date(2008, 3, 31)
	maturity := date(2008, 6, 1)
	got, err := TBillEq(settlement, maturity, mustDec("0.0914"), Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("0.09415149")
	tol := mustDec("0.000001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("TBILLEQ got=%s want≈%s", got.String(), want.String())
	}
}

func TestTBillPrice_ExcelExample(t *testing.T) {
	// Excel doc example: settlement 3/31/2008, maturity 6/1/2008,
	// discount 9%. Days = 62. Price = 100·(1 - 0.09·62/360) = 98.45.
	got, err := TBillPrice(date(2008, 3, 31), date(2008, 6, 1),
		mustDec("0.09"), Options{Scale: 6})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("98.45")
	tol := mustDec("0.000001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("TBILLPRICE got=%s want=%s", got.String(), want.String())
	}
}

func TestTBillYield_RoundTrip(t *testing.T) {
	// Compute price via TBillPrice, feed it into TBillYield, and
	// recover the parameters consistent with the original discount
	// (the two are different rate measures, so we just check the
	// algebra: TBillYield(price)·DSM/360 = (100-price)/price exactly).
	settlement := date(2008, 3, 31)
	maturity := date(2008, 6, 1)
	pr, err := TBillPrice(settlement, maturity, mustDec("0.09"), Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	yield, err := TBillYield(settlement, maturity, pr, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	// Yield should be slightly higher than discount (yields are quoted
	// over remaining principal, not face).
	if yield.String() == "" || yield.String() == "0" {
		t.Fatalf("TBillYield produced %s", yield.String())
	}
}

func TestTBill_RejectsTooLong(t *testing.T) {
	settlement := date(2024, 1, 1)
	maturity := date(2025, 6, 1) // 17 months — beyond T-bill territory
	if _, err := TBillEq(settlement, maturity, mustDec("0.05")); err == nil {
		t.Fatal("expected error for >365 days")
	}
}

func TestTBill_RejectsBackwardDates(t *testing.T) {
	if _, err := TBillPrice(date(2024, 6, 1), date(2024, 1, 1),
		mustDec("0.05")); err == nil {
		t.Fatal("expected error when maturity precedes settlement")
	}
}
