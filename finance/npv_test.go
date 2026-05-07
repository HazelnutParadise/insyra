package finance

import (
	"math/big"
	"testing"

	"github.com/TimLai666/go-decimal/decimal"
)

func TestNPV_AgainstExactRational(t *testing.T) {
	// NPV at rate=0.1 of [-1000, 300, 400, 500, 600] starting at t=0.
	// Compute the truth as a rational: Σ cf[i] · (10/11)^i for i=0..4
	// = (-1000·11^4 + 300·10·11^3 + 400·100·11^2 + 500·1000·11 + 600·10000) / 11^4
	cf := []*big.Int{
		big.NewInt(-1000),
		big.NewInt(300),
		big.NewInt(400),
		big.NewInt(500),
		big.NewInt(600),
	}
	num := new(big.Int)
	denom := new(big.Int).SetInt64(1)
	pow11 := []int64{1, 11, 121, 1331, 14641}
	pow10 := []int64{1, 10, 100, 1000, 10000}
	for i, c := range cf {
		// Term: c · 10^i / 11^i, scaled to a common denominator 11^4.
		// scaled = c · 10^i · 11^(4-i)
		coef := new(big.Int).Mul(c, big.NewInt(pow10[i]))
		coef.Mul(coef, big.NewInt(pow11[4-i]))
		num.Add(num, coef)
	}
	denom.SetInt64(pow11[4])
	want := new(big.Rat).SetFrac(num, denom)
	wantStr := want.FloatString(15)
	wantDec := mustDec(wantStr)

	cashflows := []decimal.Decimal{
		mustDec("-1000"), mustDec("300"), mustDec("400"),
		mustDec("500"), mustDec("600"),
	}
	got, err := NPV(mustDec("0.1"), cashflows, Options{Scale: 15})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.000000000001")
	if !approxEqual(t, got, wantDec, tol) {
		t.Fatalf("NPV got=%s want=%s", got.String(), wantDec.String())
	}
}

func TestIRR_RootCheck(t *testing.T) {
	// Verify NPV(IRR(cf), cf) ≈ 0 — the defining property of IRR. We
	// don't pin a specific rate; instead we make sure the rate IRR
	// returns drives NPV to zero.
	cashflows := []decimal.Decimal{
		mustDec("-1000"), mustDec("300"), mustDec("400"),
		mustDec("500"), mustDec("600"),
	}
	irr, err := IRR(cashflows, Zero, Options{Scale: 12})
	if err != nil {
		t.Fatal(err)
	}
	npv, err := NPV(irr, cashflows, Options{Scale: 12})
	if err != nil {
		t.Fatal(err)
	}
	// IRR converges in rate-space to ~Scale+8 digits; because the
	// cashflow magnitudes are O(1000) the NPV at the converged rate
	// is allowed to be O(rate-tol · cashflow), so this tolerance is
	// 4 digits looser than the rate tolerance.
	tol := mustDec("0.000001")
	if !approxEqual(t, npv, Zero, tol) {
		t.Fatalf("NPV(IRR=%s)=%s, expected ≈ 0", irr.String(), npv.String())
	}
}

func TestIRR_SimpleTwoFlow(t *testing.T) {
	// Invest 100 at t=0, receive 110 at t=1 — earns 10% over one
	// period. NPV(r) = -100 + 110/(1+r) = 0  ⇒  r = 0.1.
	cashflows := []decimal.Decimal{mustDec("-100"), mustDec("110")}
	irr, err := IRR(cashflows, Zero, Options{Scale: 12})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("0.1")
	tol := mustDec("0.0000000001")
	if !approxEqual(t, irr, want, tol) {
		t.Fatalf("IRR got=%s want=%s", irr.String(), want.String())
	}
}

func TestIRR_NeedsTwoCashflows(t *testing.T) {
	if _, err := IRR([]decimal.Decimal{mustDec("100")}, Zero); err == nil {
		t.Fatal("expected error for single cashflow")
	}
}

func TestNPV_ExcelVariantOffsetByOnePeriod(t *testing.T) {
	// NPVExcel discounts the first cashflow by one period vs NPV. So
	// the relationship NPV(rate, [0, ...cf]) == NPVExcel(rate, cf)
	// must hold exactly.
	cashflows := []decimal.Decimal{
		mustDec("300"), mustDec("400"), mustDec("500"),
	}
	rate := mustDec("0.07")
	excelNPV, err := NPVExcel(rate, cashflows, Options{Scale: 15})
	if err != nil {
		t.Fatal(err)
	}
	withZero := append([]decimal.Decimal{Zero}, cashflows...)
	plainNPV, err := NPV(rate, withZero, Options{Scale: 15})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.0000000000001")
	if !approxEqual(t, excelNPV, plainNPV, tol) {
		t.Fatalf("NPVExcel=%s != NPV(0+cf)=%s", excelNPV.String(), plainNPV.String())
	}
}
