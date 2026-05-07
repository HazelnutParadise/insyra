package finance

import (
	"testing"

	"github.com/TimLai666/go-decimal/decimal"
)

func TestIPMT_Period1_EndOfPeriod(t *testing.T) {
	// First-period interest at end-of-period timing equals -PV·rate
	// exactly (no balance has been moved yet).
	rate := mustDec("0.005")
	pv := mustDec("100000")
	got, err := IPMT(rate, 1, 360, pv, Zero, PaymentEnd, Options{Scale: 6})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("-500")
	tol := mustDec("0.000001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("IPMT[1] got=%s want≈%s", got.String(), want.String())
	}
}

func TestIPMT_Period1_PaymentBegin_IsZero(t *testing.T) {
	// With begin-of-period payments the first installment carries no
	// interest (no time has elapsed).
	rate := mustDec("0.005")
	pv := mustDec("100000")
	got, err := IPMT(rate, 1, 360, pv, Zero, PaymentBegin, Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(t, got, Zero, mustDec("0.0000001")) {
		t.Fatalf("IPMT[1] begin got=%s want=0", got.String())
	}
}

func TestIPMT_PPMT_SumEqualsPMT(t *testing.T) {
	// Identity: IPMT[per] + PPMT[per] == PMT for every period.
	rate := mustDec("0.005")
	pv := mustDec("100000")
	pmt, _ := PMT(rate, 360, pv, Zero, PaymentEnd, Options{Scale: 20})

	work := decimal.Context{Scale: 20, Mode: decimal.RoundingModeHalfUp}
	for _, per := range []int{1, 2, 90, 180, 359, 360} {
		ipmt, err := IPMT(rate, per, 360, pv, Zero, PaymentEnd, Options{Scale: 20})
		if err != nil {
			t.Fatal(err)
		}
		ppmt, err := PPMT(rate, per, 360, pv, Zero, PaymentEnd, Options{Scale: 20})
		if err != nil {
			t.Fatal(err)
		}
		sum := decimal.Add(work, ipmt, ppmt)
		tol := mustDec("0.0000000001")
		if !approxEqual(t, sum, pmt, tol) {
			t.Fatalf("per=%d ipmt+ppmt=%s want pmt=%s", per, sum.String(), pmt.String())
		}
	}
}

func TestCumIPMT_PlusCumPPMT_EqualsTotalPaid(t *testing.T) {
	// CumIPMT + CumPPMT over [1, nper] should equal pmt·nper.
	rate := mustDec("0.005")
	pv := mustDec("100000")
	pmt, _ := PMT(rate, 360, pv, Zero, PaymentEnd, Options{Scale: 20})

	work := decimal.Context{Scale: 20, Mode: decimal.RoundingModeHalfUp}
	cumI, err := CumIPMT(rate, 360, pv, 1, 360, PaymentEnd, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	cumP, err := CumPPMT(rate, 360, pv, 1, 360, PaymentEnd, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	totalPaid := decimal.Mul(work, pmt, decimal.NewFromInt64(work, 360))
	sum := decimal.Add(work, cumI, cumP)
	tol := mustDec("0.0000000001")
	if !approxEqual(t, sum, totalPaid, tol) {
		t.Fatalf("cumI+cumP=%s want %s", sum.String(), totalPaid.String())
	}
}

func TestCumPPMT_EqualsNegPV_WhenFVZero(t *testing.T) {
	// Total principal repaid over the whole loan equals -PV when fv=0.
	rate := mustDec("0.005")
	pv := mustDec("100000")
	cumP, err := CumPPMT(rate, 360, pv, 1, 360, PaymentEnd, Options{Scale: 12})
	if err != nil {
		t.Fatal(err)
	}
	want := decimal.Neg(pv)
	tol := mustDec("0.0000000001")
	if !approxEqual(t, cumP, want, tol) {
		t.Fatalf("cumP=%s want=%s", cumP.String(), want.String())
	}
}

func TestIPMT_ErrorOnBadPeriod(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	if _, err := IPMT(rate, 0, 360, pv, Zero, PaymentEnd); err == nil {
		t.Fatal("expected error per=0")
	}
	if _, err := IPMT(rate, 361, 360, pv, Zero, PaymentEnd); err == nil {
		t.Fatal("expected error per>nper")
	}
}
