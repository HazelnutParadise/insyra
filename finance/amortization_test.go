package finance

import (
	"testing"

	"github.com/TimLai666/go-decimal/decimal"
)

func TestAmortization_RowCountAndPayment(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	pmt, _ := PMT(rate, 12, pv, Zero, PaymentEnd, Options{Scale: 14})

	rows, err := AmortizationSchedule(rate, 12, pv, Zero, PaymentEnd, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 12 {
		t.Fatalf("len(rows)=%d, want 12", len(rows))
	}

	// Every Payment column entry equals PMT.
	for _, r := range rows {
		if !approxEqual(t, r.Payment, pmt, mustDec("0.0000000001")) {
			t.Fatalf("period %d Payment=%s want=%s",
				r.Period, r.Payment.String(), pmt.String())
		}
	}
}

func TestAmortization_FinalBalanceIsZero(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	rows, err := AmortizationSchedule(rate, 360, pv, Zero, PaymentEnd, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	last := rows[len(rows)-1]
	if !approxEqual(t, last.Balance, Zero, mustDec("0.000000001")) {
		t.Fatalf("final balance=%s want≈0", last.Balance.String())
	}
}

func TestAmortization_SumIPMT_MatchesCumIPMT(t *testing.T) {
	// Iterating IPMT/PPMT through the schedule must add up to the
	// closed-form CumIPMT/CumPPMT — cross-checks both implementations.
	rate := mustDec("0.005")
	pv := mustDec("100000")
	nper := 12

	rows, err := AmortizationSchedule(rate, nper, pv, Zero, PaymentEnd, Options{Scale: 16})
	if err != nil {
		t.Fatal(err)
	}
	work := decimal.Context{Scale: 16, Mode: decimal.RoundingModeHalfUp}

	sumI := decimal.NewFromInt64(work, 0)
	sumP := decimal.NewFromInt64(work, 0)
	for _, r := range rows {
		sumI = decimal.Add(work, sumI, r.Interest)
		sumP = decimal.Add(work, sumP, r.Principal)
	}
	cumI, _ := CumIPMT(rate, nper, pv, 1, nper, PaymentEnd, Options{Scale: 14})
	cumP, _ := CumPPMT(rate, nper, pv, 1, nper, PaymentEnd, Options{Scale: 14})
	tol := mustDec("0.000000001")
	if !approxEqual(t, sumI, cumI, tol) {
		t.Fatalf("ΣInterest=%s, CumIPMT=%s", sumI.String(), cumI.String())
	}
	if !approxEqual(t, sumP, cumP, tol) {
		t.Fatalf("ΣPrincipal=%s, CumPPMT=%s", sumP.String(), cumP.String())
	}
}

func TestAmortization_PaymentBegin_FinalBalanceIsZero(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	rows, err := AmortizationSchedule(rate, 360, pv, Zero, PaymentBegin, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	last := rows[len(rows)-1]
	if !approxEqual(t, last.Balance, Zero, mustDec("0.000000001")) {
		t.Fatalf("final balance (begin timing)=%s want≈0", last.Balance.String())
	}
}

func TestScheduleTable_HasFiveColumns(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("10000")
	tbl, err := ScheduleTable(rate, 6, pv, Zero, PaymentEnd, Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	rows, cols := tbl.Size()
	if rows != 6 {
		t.Fatalf("rows=%d, want 6", rows)
	}
	if cols != 5 {
		t.Fatalf("cols=%d, want 5 (Period, Payment, Interest, Principal, Balance)", cols)
	}
}
