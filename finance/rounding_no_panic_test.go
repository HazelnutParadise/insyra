package finance

import (
	"strings"
	"testing"

	"github.com/TimLai666/go-decimal/decimal"
)

// FI-3 of #248: RoundUnnecessary asks the decimal library to panic with
// ErrRoundingNecessary the moment a result has to be rounded. error-philosophy
// says no insyra package panics under the default configuration, and a rounding
// mode the caller chose is not a reason to take their program down — especially
// when the whole point of the mode is to find out that rounding was needed.
func TestRoundUnnecessary_ReportsInsteadOfPanicking(t *testing.T) {
	// 1 / 1.03 is not representable at two decimal places, so this is exactly
	// the case the mode asserts will not happen.
	rate := mustParse(t, "0.03")

	var got decimal.Decimal
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("rounding under RoundUnnecessary panicked: %v", r)
			}
		}()
		got, err = NPV(rate, []decimal.Decimal{mustParse(t, "0"), mustParse(t, "1")},
			Options{Scale: 2, Mode: RoundUnnecessary})
	}()

	if err == nil {
		t.Fatalf("a result that had to be rounded reported no error (got %v)", got)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "round") {
		t.Errorf("the error %q does not say rounding was needed", err)
	}
}

// A mode that can round still works, and the error only appears when rounding
// was actually required.
func TestRoundUnnecessary_ExactResultIsFine(t *testing.T) {
	got, err := NPV(mustParse(t, "0"), []decimal.Decimal{mustParse(t, "1"), mustParse(t, "2")},
		Options{Scale: 2, Mode: RoundUnnecessary})
	if err != nil {
		t.Fatalf("an exact result reported %v", err)
	}
	if got.String() != "3.00" && got.String() != "3" {
		t.Errorf("got %v, want 3", got)
	}
}

func mustParse(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.ParseExact(s)
	if err != nil {
		t.Fatalf("parsing %q: %v", s, err)
	}
	return d
}

// The same for a schedule, which rounds every cell rather than one result.
func TestRoundUnnecessary_ScheduleReportsInsteadOfPanicking(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("building a schedule under RoundUnnecessary panicked: %v", r)
		}
	}()

	_, err := AmortizationSchedule(mustParse(t, "0.03"), 3,
		mustParse(t, "1000"), mustParse(t, "0"), PaymentEnd,
		Options{Scale: 2, Mode: RoundUnnecessary})
	if err == nil {
		t.Fatal("a schedule that had to be rounded reported no error")
	}
}

// Other rounding modes are untouched.
func TestOtherRoundingModesStillRound(t *testing.T) {
	got, err := NPV(mustParse(t, "0.03"), []decimal.Decimal{mustParse(t, "0"), mustParse(t, "1")},
		Options{Scale: 2, Mode: RoundHalfUp})
	if err != nil {
		t.Fatalf("NPV: %v", err)
	}
	if got.String() != "0.97" {
		t.Errorf("got %v, want 0.97", got)
	}
}
