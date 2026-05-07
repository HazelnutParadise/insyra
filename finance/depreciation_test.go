package finance

import (
	"testing"

	"github.com/TimLai666/go-decimal/decimal"
)

func TestSLN_Basic(t *testing.T) {
	// Asset: cost 30000, salvage 7500, life 10 → 2250 per year.
	got, err := SLN(mustDec("30000"), mustDec("7500"), 10, Options{Scale: 4})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("2250")
	tol := mustDec("0.0001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("SLN got=%s want=%s", got.String(), want.String())
	}
}

func TestSYD_Basic(t *testing.T) {
	// cost 30000, salvage 7500, life 10, period 1
	// SYD = 22500 · 10 / 55 = 4090.9090909…
	got, err := SYD(mustDec("30000"), mustDec("7500"), 10, 1, Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("4090.90909091")
	tol := mustDec("0.0000001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("SYD[1] got=%s want=%s", got.String(), want.String())
	}
}

func TestSYD_TotalEqualsDepreciable(t *testing.T) {
	// Σ SYD[per] for per=1..life equals cost - salvage exactly.
	cost := mustDec("30000")
	salv := mustDec("7500")
	life := 10
	work := decimal.Context{Scale: 14, Mode: decimal.RoundingModeHalfUp}

	total := decimal.NewFromInt64(work, 0)
	for per := 1; per <= life; per++ {
		s, err := SYD(cost, salv, life, per, Options{Scale: 14})
		if err != nil {
			t.Fatal(err)
		}
		total = decimal.Add(work, total, s)
	}
	want := decimal.Sub(work, cost, salv) // 22500
	tol := mustDec("0.0000000001")
	if !approxEqual(t, total, want, tol) {
		t.Fatalf("ΣSYD got=%s want=%s", total.String(), want.String())
	}
}

func TestDDB_FirstAndSecondPeriod(t *testing.T) {
	// cost 2400, salvage 300, life 10, factor 2 → rate 0.2.
	// Period 1: 2400·0.2 = 480
	// Period 2: 1920·0.2 = 384
	got1, err := DDB(mustDec("2400"), mustDec("300"), 10, 1, mustDec("2"), Options{Scale: 6})
	if err != nil {
		t.Fatal(err)
	}
	if !approxEqual(t, got1, mustDec("480"), mustDec("0.000001")) {
		t.Fatalf("DDB[1] got=%s want=480", got1.String())
	}
	got2, _ := DDB(mustDec("2400"), mustDec("300"), 10, 2, mustDec("2"), Options{Scale: 6})
	if !approxEqual(t, got2, mustDec("384"), mustDec("0.000001")) {
		t.Fatalf("DDB[2] got=%s want=384", got2.String())
	}
}

func TestDDB_CapsAtSalvage(t *testing.T) {
	// Past the point where book value would dip below salvage, the
	// remaining periods should depreciate to exactly salvage and then
	// produce zero further depreciation.
	got, err := DDB(mustDec("100"), mustDec("90"), 5, 5, mustDec("2"), Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	// Book value reaches 90 quickly; later periods produce 0 deprec.
	if decimal.Cmp(got, mustDec("0.00000001")) > 0 {
		t.Fatalf("DDB with salvage cap got=%s want≈0", got.String())
	}
}

func TestVDB_NoSwitch_MatchesIterativeDDB(t *testing.T) {
	// VDB with noSwitch=true must equal Σ DDB[k] over k=1..life when
	// the [0, life] range covers the whole life.
	cost := mustDec("2400")
	salv := mustDec("300")
	life := 10
	factor := mustDec("2")

	work := decimal.Context{Scale: 14, Mode: decimal.RoundingModeHalfUp}
	totalDDB := decimal.NewFromInt64(work, 0)
	for per := 1; per <= life; per++ {
		d, err := DDB(cost, salv, life, per, factor, Options{Scale: 14})
		if err != nil {
			t.Fatal(err)
		}
		totalDDB = decimal.Add(work, totalDDB, d)
	}
	totalVDB, err := VDB(cost, salv, life,
		decimal.NewFromInt64(work, 0),
		decimal.NewFromInt64(work, int64(life)),
		factor, true, Options{Scale: 14})
	if err != nil {
		t.Fatal(err)
	}
	tol := mustDec("0.0000000001")
	if !approxEqual(t, totalVDB, totalDDB, tol) {
		t.Fatalf("VDB(noSwitch) cum=%s != ΣDDB=%s", totalVDB.String(), totalDDB.String())
	}
}

func TestVDB_WithSwitch_ReachesFullDepreciation(t *testing.T) {
	// Default factor=2 with switch. Over the full asset life the
	// total depreciation should equal cost - salvage exactly (the
	// switch to straight-line guarantees we use up the whole
	// depreciable base).
	cost := mustDec("10000")
	salv := mustDec("1000")
	life := 5
	factor := mustDec("2")

	work := decimal.Context{Scale: 14, Mode: decimal.RoundingModeHalfUp}
	total, err := VDB(cost, salv, life,
		decimal.NewFromInt64(work, 0),
		decimal.NewFromInt64(work, int64(life)),
		factor, false, Options{Scale: 12})
	if err != nil {
		t.Fatal(err)
	}
	want := decimal.Sub(work, cost, salv) // 9000
	tol := mustDec("0.0000000001")
	if !approxEqual(t, total, want, tol) {
		t.Fatalf("VDB(switch, full) got=%s want=%s", total.String(), want.String())
	}
}

func TestVDB_PartialPeriod(t *testing.T) {
	// VDB(start=0, end=0.5) should give roughly half of period 1's
	// declining-balance depreciation, since fractional periods are
	// pro-rated linearly within a period.
	cost := mustDec("10000")
	salv := mustDec("0")
	life := 5
	factor := mustDec("2") // rate = 0.4 → period 1 depr = 4000
	got, err := VDB(cost, salv, life,
		mustDec("0"), mustDec("0.5"),
		factor, true, Options{Scale: 8})
	if err != nil {
		t.Fatal(err)
	}
	want := mustDec("2000")
	tol := mustDec("0.0001")
	if !approxEqual(t, got, want, tol) {
		t.Fatalf("VDB partial got=%s want=2000", got.String())
	}
}

func TestSLN_SYD_DDB_Errors(t *testing.T) {
	if _, err := SLN(mustDec("100"), mustDec("10"), 0); err == nil {
		t.Fatal("expected error for life=0")
	}
	if _, err := SYD(mustDec("100"), mustDec("10"), 5, 6); err == nil {
		t.Fatal("expected error for per>life")
	}
	if _, err := DDB(mustDec("100"), mustDec("10"), 5, 0, mustDec("2")); err == nil {
		t.Fatal("expected error for per=0")
	}
}
