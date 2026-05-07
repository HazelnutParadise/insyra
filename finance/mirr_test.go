package finance

import (
	"testing"

	"github.com/TimLai666/go-decimal/decimal"
)

func TestMIRR_TextbookExample(t *testing.T) {
	// Classic Excel MIRR example: -120, 39, 30, 21, 37, 46 with
	// financeRate=10% and reinvestRate=12%.
	// Excel returns ≈ 0.126094 (12.6094%).
	cf := []decimal.Decimal{
		mustDec("-120000"),
		mustDec("39000"),
		mustDec("30000"),
		mustDec("21000"),
		mustDec("37000"),
		mustDec("46000"),
	}
	got, err := MIRR(cf, mustDec("0.10"), mustDec("0.12"), Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("0.12609413")
	tol := mustDec("0.0000001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("MIRR got=%s want≈%s", got.String(), want.String())
	}
}

func TestMIRR_EqualToIRR_WhenRatesMatch(t *testing.T) {
	// When financeRate == reinvestRate == r and r equals the IRR of
	// the cashflow stream, MIRR collapses back to that single rate.
	cf := []decimal.Decimal{
		mustDec("-1000"),
		mustDec("300"),
		mustDec("400"),
		mustDec("500"),
		mustDec("600"),
	}
	irr, err := IRR(cf, Zero, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	mirr, err := MIRR(cf, irr, irr, Options{Scale: 10})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.000000001")
	if !approxEqual(t, mirr, irr, tol) {
		t.Fatalf("MIRR(IRR, IRR) got=%s want=%s", mirr.String(), irr.String())
	}
}

func TestMIRR_RejectsNoNegativeFlow(t *testing.T) {
	cf := []decimal.Decimal{mustDec("100"), mustDec("200"), mustDec("300")}
	if _, err := MIRR(cf, mustDec("0.1"), mustDec("0.1")); err == nil {
		t.Fatal("expected error when no negative cashflows")
	}
}

func TestMIRR_RejectsNoPositiveFlow(t *testing.T) {
	cf := []decimal.Decimal{mustDec("-100"), mustDec("-200")}
	if _, err := MIRR(cf, mustDec("0.1"), mustDec("0.1")); err == nil {
		t.Fatal("expected error when no positive cashflows")
	}
}
