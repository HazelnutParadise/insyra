package finance

import (
	"math/big"
	"testing"
)

func TestEffectiveRate_AgainstExactRational(t *testing.T) {
	// EFFECT(0.06, 12) = (1 + 0.06/12)^12 - 1 = (1.005)^12 - 1
	//                  = (201^12 - 200^12) / 200^12
	a := new(big.Int).Exp(big.NewInt(201), big.NewInt(12), nil)
	b := new(big.Int).Exp(big.NewInt(200), big.NewInt(12), nil)
	num := new(big.Int).Sub(a, b)
	want := new(big.Rat).SetFrac(num, b)
	wantDec := mustDec(want.FloatString(20))

	got, err := EffectiveRate(mustDec("0.06"), 12, Options{Scale: 18})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.0000000000000001")
	if !approxEqual(t, got, wantDec, tol) {
		t.Fatalf("Effective got=%s want=%s", got.String(), wantDec.String())
	}
}

func TestEffectiveAndNominal_RoundTrip(t *testing.T) {
	// NominalRate ∘ EffectiveRate should be the identity.
	for _, m := range []int{1, 4, 12, 365} {
		nominal := mustDec("0.06")
		eff, err := EffectiveRate(nominal, m, Options{Scale: 30})
		if err != nil {
			t.Fatalf("EffectiveRate m=%d: %v", m, err)
		}
		round, err := NominalRate(eff, m, Options{Scale: 20})
		if err != nil {
			t.Fatalf("NominalRate m=%d: %v", m, err)
		}
		tol := mustDec("0.000000000001")
		if !approxEqual(t, round, nominal, tol) {
			t.Fatalf("m=%d round-trip got=%s want=%s",
				m, round.String(), nominal.String())
		}
	}
}

func TestContinuousRoundTrip(t *testing.T) {
	// AnnualFromContinuous ∘ ContinuousFromAnnual is the identity.
	annual := mustDec("0.075")
	cont, err := ContinuousFromAnnual(annual, Options{Scale: 30})
	if err != nil {
		t.Fatal(err)
	}
	back, err := AnnualFromContinuous(cont, Options{Scale: 20})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.000000000001")
	if !approxEqual(t, back, annual, tol) {
		t.Fatalf("continuous round-trip got=%s want=%s",
			back.String(), annual.String())
	}
}

func TestEffectiveRate_MonthlyCompoundingExceedsNominal(t *testing.T) {
	// 6% nominal compounded monthly should exceed 6% by a small but
	// strictly positive amount (~6.168%). Sanity check that the sign
	// of the spread is right.
	eff, err := EffectiveRate(mustDec("0.06"), 12, Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(t, eff, mustDec("0.06167781"), mustDec("0.00000001")) {
		t.Fatalf("effective@6%%/12 got=%s, want≈0.06167781", eff.String())
	}
}

func TestEffectiveRate_BadPeriods(t *testing.T) {
	if _, err := EffectiveRate(mustDec("0.05"), 0); err == nil {
		t.Fatal("expected error for periodsPerYear=0")
	}
}
