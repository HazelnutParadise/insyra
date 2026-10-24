package quant

import (
	"math"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// Four stocks over four months, simple returns:
//
//	      m0     m1     m2     m3
//	A   0.10   0.02  -0.01   0.01
//	B   0.05   0.04   0.02   0.03
//	C  -0.05   0.01   0.03  -0.02
//	D  -0.10   0.03   0.00   0.04
func momentumPanel() *insyra.DataTable {
	dates := []any{}
	stocks := []any{}
	rets := []any{}
	data := map[string][]float64{
		"A": {0.10, 0.02, -0.01, 0.01},
		"B": {0.05, 0.04, 0.02, 0.03},
		"C": {-0.05, 0.01, 0.03, -0.02},
		"D": {-0.10, 0.03, 0.00, 0.04},
	}
	for m := 0; m < 4; m++ {
		for _, s := range []string{"A", "B", "C", "D"} {
			dates = append(dates, "m"+string(rune('0'+m)))
			stocks = append(stocks, s)
			rets = append(rets, data[s][m])
		}
	}
	return insyra.NewDataTable(
		insyra.NewDataList(dates...).SetName("date"),
		insyra.NewDataList(stocks...).SetName("stock"),
		insyra.NewDataList(rets...).SetName("ret"),
	)
}

func columnFloats(t *testing.T, dt *insyra.DataTable, name string) []float64 {
	t.Helper()
	col := dt.GetColByName(name)
	if col == nil {
		t.Fatalf("no column %q", name)
	}
	out := []float64{}
	for _, v := range col.Data() {
		f, _ := insyra.ToFloat64Safe(v)
		out = append(out, f)
	}
	return out
}

func expectClose(t *testing.T, name string, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", name, got, want)
	}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-12 {
			t.Fatalf("%s: row %d: got %v, want %v (all %v)", name, i, got[i], want[i], got)
		}
	}
}

// J=1, S=0, K=1: rank on last month's return, hold the next month. Losers are
// C and D then A and C then A and D; winners the other half.
func TestMomentumPortfoliosOneMonthHold(t *testing.T) {
	res, err := MomentumPortfolios(momentumPanel(), insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	// Month 1: winners {A,B} 0.03, losers {C,D} 0.02.
	// Month 2: winners {B,D} 0.01, losers {A,C} 0.01.
	// Month 3: winners {B,C} 0.005, losers {A,D} 0.025.
	expectClose(t, "losers", columnFloats(t, res.Groups, "group_1"), []float64{0.02, 0.01, 0.025})
	expectClose(t, "winners", columnFloats(t, res.Groups, "group_2"), []float64{0.03, 0.01, 0.005})
	expectClose(t, "long-short", floats(t, res.LongShort), []float64{0.01, 0, -0.02})
	dates := res.Groups.GetColByName("date").Data()
	if len(dates) != 3 || dates[0] != "m1" || dates[2] != "m3" {
		t.Errorf("dates %v, want m1..m3", dates)
	}
}

// K=2: each month averages the two cohorts formed one and two months before, so
// the first month with both is m2.
func TestMomentumPortfoliosOverlappingCohorts(t *testing.T) {
	res, err := MomentumPortfolios(momentumPanel(), insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2, MomentumOptions{Hold: 2})
	if err != nil {
		t.Fatal(err)
	}
	// m2: winners (0.005 + 0.01)/2, losers (0.015 + 0.01)/2. m3: winners (0.005 + 0.035)/2, losers (0.025 - 0.005)/2.
	expectClose(t, "losers", columnFloats(t, res.Groups, "group_1"), []float64{0.0125, 0.01})
	expectClose(t, "winners", columnFloats(t, res.Groups, "group_2"), []float64{0.0075, 0.02})
	expectClose(t, "long-short", floats(t, res.LongShort), []float64{-0.005, 0.01})
}

// S=1 ranks on the return of the month before last: the cohort formed in m1
// ranks on m0 and holds m2.
func TestMomentumPortfoliosSkipsTheMostRecentMonth(t *testing.T) {
	res, err := MomentumPortfolios(momentumPanel(), insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2, MomentumOptions{Skip: 1})
	if err != nil {
		t.Fatal(err)
	}
	// m2: winners {A,B} 0.005, losers {C,D} 0.015. m3: winners {D,B} 0.035, losers {C,A} -0.005.
	expectClose(t, "long-short", floats(t, res.LongShort), []float64{-0.01, 0.04})
}

// J=2 compounds the two months: A 1.10*1.02-1, B 1.05*1.04-1, C 0.95*1.01-1, D 0.90*1.03-1.
func TestMomentumPortfoliosCompoundsTheFormationMonths(t *testing.T) {
	res, err := MomentumPortfolios(momentumPanel(), insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	// The first cohort forms at m1 and holds m2: winners {A,B} 0.005, losers {C,D} 0.015.
	expectClose(t, "long-short", floats(t, res.LongShort)[:1], []float64{-0.01})
	if got := len(floats(t, res.LongShort)); got != 2 {
		t.Errorf("%d months, want m2 and m3", got)
	}
}

func TestMomentumPortfoliosDropsAMissingHoldingReturn(t *testing.T) {
	panel := momentumPanel()
	// Remove B's return in m1: winners of the first cohort are {A,B}, now only A (0.02).
	panel.GetColByName("ret").Data() // read to confirm the column exists
	panel.UpdateElement(5, "C", nil)
	res, err := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	expectClose(t, "winners", columnFloats(t, res.Groups, "group_2")[:1], []float64{0.02})
}

func TestMomentumPortfoliosSkipsAMonthWithTooFewStocks(t *testing.T) {
	dt := insyra.NewDataTable(
		insyra.NewDataList("m0", "m0", "m0", "m1", "m1", "m1", "m2", "m2", "m2").SetName("date"),
		insyra.NewDataList("A", "B", "C", "A", "B", "C", "A", "B", "C").SetName("stock"),
		insyra.NewDataList(0.1, 0.0, -0.1, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0).SetName("ret"),
	)
	// Three groups need three stocks; one return of m0 is missing below.
	dt.UpdateElement(2, "C", nil)
	res, err := MomentumPortfolios(dt, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) == 0 {
		t.Error("a cohort with two ranked stocks and three groups was not reported as skipped")
	}
}

func TestMomentumPortfoliosRefusesBadInput(t *testing.T) {
	panel := momentumPanel()
	for name, f := range map[string]func() error{
		"formation 0": func() error {
			_, e := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 0, 2)
			return e
		},
		"one group": func() error {
			_, e := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 1)
			return e
		},
		"negative skip": func() error {
			_, e := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2, MomentumOptions{Skip: -1})
			return e
		},
		"negative hold": func() error {
			_, e := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2, MomentumOptions{Hold: -1})
			return e
		},
		"two options": func() error {
			_, e := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2, MomentumOptions{}, MomentumOptions{})
			return e
		},
		"no column": func() error {
			_, e := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("nope"), 1, 2)
			return e
		},
		"nil table": func() error {
			_, e := MomentumPortfolios(nil, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2)
			return e
		},
		"too short": func() error {
			_, e := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 3, 2, MomentumOptions{Hold: 3})
			return e
		},
	} {
		if f() == nil {
			t.Errorf("%s: no error", name)
		}
	}
	dup := insyra.NewDataTable(
		insyra.NewDataList("m0", "m0", "m1", "m1").SetName("date"),
		insyra.NewDataList("A", "A", "A", "B").SetName("stock"),
		insyra.NewDataList(0.1, 0.2, 0.3, 0.4).SetName("ret"),
	)
	if _, err := MomentumPortfolios(dup, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2); err == nil {
		t.Error("a stock twice in one month: no error")
	}
}

// With Hold 2 the group is the average of two portfolios; if one of them has no
// return in the month, the average is unknown, not the other portfolio alone.
func TestMomentumPortfoliosOverlapWithAMissingPortfolioIsNaN(t *testing.T) {
	panel := momentumPanel()
	// Month 2, stocks A and B (the m0 cohort's winners): both returns missing.
	// Rows: m2 starts at row 8, A=8, B=9.
	panel.UpdateElement(8, "C", nil)
	panel.UpdateElement(9, "C", nil)
	res, err := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2, MomentumOptions{Hold: 2})
	if err != nil {
		t.Fatal(err)
	}
	winners := columnFloats(t, res.Groups, "group_2")
	if !math.IsNaN(winners[0]) {
		t.Errorf("m2 winners = %v, want NaN: the portfolio formed at m0 has no return in m2", winners[0])
	}
}

// 1e308 twice then -1 overflows the compounded product to NaN; the stock must be
// left unranked, not hang the ranking.
func TestMomentumPortfoliosDoesNotHangOnAnOverflowingSignal(t *testing.T) {
	dates, stocks, rets := []any{}, []any{}, []any{}
	add := func(m string, vals map[string]float64) {
		for _, s := range []string{"A", "B", "C"} {
			dates = append(dates, m)
			stocks = append(stocks, s)
			rets = append(rets, vals[s])
		}
	}
	add("m0", map[string]float64{"A": 1e308, "B": 0.1, "C": -0.1})
	add("m1", map[string]float64{"A": 1e308, "B": 0.0, "C": 0.0})
	add("m2", map[string]float64{"A": -1, "B": 0.1, "C": -0.1})
	add("m3", map[string]float64{"A": 0.0, "B": 0.1, "C": -0.1})
	panel := insyra.NewDataTable(
		insyra.NewDataList(dates...).SetName("date"),
		insyra.NewDataList(stocks...).SetName("stock"),
		insyra.NewDataList(rets...).SetName("ret"),
	)
	done := make(chan error, 1)
	go func() {
		_, err := MomentumPortfolios(panel, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 3, 2)
		done <- err
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("MomentumPortfolios did not return on an overflowing compounded signal")
	}
}
