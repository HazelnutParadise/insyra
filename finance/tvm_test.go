package finance

import (
	"strings"
	"testing"

	"github.com/TimLai666/go-decimal/decimal"
)

// approxEqual reports whether |a - b| <= tol when both are parsed at
// 16 digits. Comparing as strings would be too brittle (we want
// rounding equivalence, not byte equivalence).
func approxEqual(t *testing.T, got, want, tol decimal.Decimal) bool {
	t.Helper()
	ctx := decimal.Context{Scale: 20, Mode: decimal.RoundingModeHalfUp}
	diff := decimal.Sub(ctx, got, want)
	if decimal.Cmp(diff, decimal.NewFromInt64(ctx, 0)) < 0 {
		diff = decimal.Neg(diff)
	}
	return decimal.Cmp(diff, tol) <= 0
}

func mustDec(s string) decimal.Decimal { return MustNew(s) }

// 30-year mortgage, 6% annual / 12 = 0.5% monthly, $100,000 loan.
// Truth from exact rational arithmetic (1.005 = 201/200, so PMT is
// rational): -500·201^360 / (201^360 - 200^360). Note Excel/float64
// truncates this to ~-599.5505251260807, which is wrong from the 11th
// significant digit onward — verifiable via big.Rat.
const mortgagePMTRef = "-599.55052515275239459146124368447591503704"

func TestPMT_StandardMortgage(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	got, err := PMT(rate, 360, pv, Zero, PaymentEnd, Options{Scale: 20})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec(mortgagePMTRef)
	tol := mustDec("0.0000000001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("PMT got=%s want=%s", got.String(), want.String())
	}
}

func TestPMT_DefaultScale(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	got, err := PMT(rate, 360, pv, Zero, PaymentEnd)
	if err != nil {
		t.Fatal(err)
	}
	// Default scale=10. Exact 11th digit of true PMT is …527524 —
	// rounds to 5275 at 4 decimal places past the 7th, i.e. .5505251528.
	want := "-599.5505251528"
	if got.String() != want {
		t.Fatalf("PMT default scale got=%s want=%s", got.String(), want)
	}
}

func TestPMT_ZeroRate(t *testing.T) {
	// If rate=0, PMT just splits PV evenly: PMT = -PV / nper.
	pv := mustDec("12000")
	got, err := PMT(Zero, 12, pv, Zero, PaymentEnd)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "-1000.0000000000" {
		t.Fatalf("PMT zero-rate got=%s want=-1000.0000000000", got.String())
	}
}

func TestPMT_PaymentBegin_LowerThanEnd(t *testing.T) {
	// Annuity-due payments are smaller (money invested longer), so
	// |PMT_begin| < |PMT_end|.
	rate := mustDec("0.005")
	pv := mustDec("100000")
	end, _ := PMT(rate, 360, pv, Zero, PaymentEnd)
	begin, _ := PMT(rate, 360, pv, Zero, PaymentBegin)
	// Both negative; |begin| < |end|.
	if decimal.Cmp(begin, end) <= 0 {
		t.Fatalf("expected begin > end (less negative), got begin=%s end=%s",
			begin.String(), end.String())
	}
}

func TestPV_RoundTripFromPMT(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	pmt, err := PMT(rate, 360, pv, Zero, PaymentEnd, Options{Scale: 20})
	if err != nil {
		t.Fatal(err)
	}
	got, err := PV(rate, 360, pmt, Zero, PaymentEnd, Options{Scale: 12})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.000001")
	if !approxEqual(t, got, pv, tol) {
		t.Fatalf("PV round-trip got=%s want=%s", got.String(), pv.String())
	}
}

func TestFV_RoundTripFromPMT(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	pmt, err := PMT(rate, 360, pv, Zero, PaymentEnd, Options{Scale: 20})
	if err != nil {
		t.Fatal(err)
	}
	got, err := FV(rate, 360, pmt, pv, PaymentEnd, Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	// After paying the loan off, balance should land at 0.
	tol := mustDec("0.0001")
	if !approxEqual(t, got, Zero, tol) {
		t.Fatalf("FV round-trip got=%s want≈0", got.String())
	}
}

func TestFV_SavingsAccount(t *testing.T) {
	// Save $100/month for 12 months at 0.5%/month, opening balance 0.
	// Truth: 20000·(201^12 - 200^12)/200^12 = 1233.55623728999137579415…
	// (Excel/float64 gives 1233.55609479, wrong from the 9th digit.)
	rate := mustDec("0.005")
	pmt := mustDec("-100")
	got, err := FV(rate, 12, pmt, Zero, PaymentEnd, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("1233.55623728999138")
	tol := mustDec("0.00000000001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("FV got=%s want≈%s", got.String(), want.String())
	}
}

func TestNPER_RoundTrip(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	pmt, _ := PMT(rate, 360, pv, Zero, PaymentEnd, Options{Scale: 20})
	got, err := NPER(rate, pmt, pv, Zero, PaymentEnd, Options{Scale: 6})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("360")
	tol := mustDec("0.000001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("NPER got=%s want=360", got.String())
	}
}

func TestNPER_ZeroRate(t *testing.T) {
	// rate=0 case: nper = -(pv+fv)/pmt = 12000 / 1000 = 12
	got, err := NPER(Zero, mustDec("-1000"), mustDec("12000"), Zero, PaymentEnd)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "12.0000000000" {
		t.Fatalf("NPER zero-rate got=%s want=12.0000000000", got.String())
	}
}

func TestRATE_RoundTrip(t *testing.T) {
	rate := mustDec("0.005")
	pv := mustDec("100000")
	pmt, _ := PMT(rate, 360, pv, Zero, PaymentEnd, Options{Scale: 20})
	got, err := RATE(360, pmt, pv, Zero, PaymentEnd, Zero, Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.00000001")
	if !approxEqual(t, got, rate, tol) {
		t.Fatalf("RATE got=%s want=%s", got.String(), rate.String())
	}
}

func TestPMT_Errors(t *testing.T) {
	if _, err := PMT(mustDec("0.05"), 0, mustDec("100"), Zero, PaymentEnd); err == nil {
		t.Fatal("expected error for nper=0")
	}
	if _, err := PMT(mustDec("0.05"), -1, mustDec("100"), Zero, PaymentEnd); err == nil {
		t.Fatal("expected error for nper<0")
	}
	if _, err := PMT(mustDec("0.05"), 12, mustDec("100"), Zero, PaymentTiming(99)); err == nil {
		t.Fatal("expected error for invalid timing")
	}
}

func TestNew_ParseError(t *testing.T) {
	if _, err := New("not-a-number"); err == nil {
		t.Fatal("expected parse error")
	}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("MustNew should have panicked")
		}
	}()
	_ = MustNew("also-bad")
}

func TestPMT_HighPrecision(t *testing.T) {
	// Verify that 50 digits of scale produce a stable answer; the
	// prefix should match the documented mortgagePMTRef constant.
	rate := mustDec("0.005")
	pv := mustDec("100000")
	got, err := PMT(rate, 360, pv, Zero, PaymentEnd, Options{Scale: 50})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.String(), "-599.5505251527523945914612436844") {
		t.Fatalf("high-precision PMT got=%s", got.String())
	}
}
