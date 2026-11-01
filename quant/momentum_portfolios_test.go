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

// Four stocks over four months for the turnover tests; D has no return in m3.
//
//	      m0     m1     m2     m3
//	A   0.10   0.05   0.00   0.10
//	B   0.00   0.10   0.20   0.00
//	C  -0.10   0.00   0.10   0.05
//	D   0.20  -0.05   0.15    nil
func turnoverPanel() *insyra.DataTable {
	data := map[string][]any{
		"A": {0.10, 0.05, 0.00, 0.10},
		"B": {0.00, 0.10, 0.20, 0.00},
		"C": {-0.10, 0.00, 0.10, 0.05},
		"D": {0.20, -0.05, 0.15, nil},
	}
	dates, stocks, rets := []any{}, []any{}, []any{}
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

// Hold 1. Losers are C,B then D,C then A,C; winners A,D then A,B then D,B.
//
// m2 winners: m1 weights A .5 D .5 drift to A .525 D .475; m2 holds A .5 B .5,
// so |.5-.525| + .5 + .475 = 1. Losers: C .5 B .5 drift to C .5/1.05 and
// B .55/1.05; m2 holds D .5 C .5, so the sum is 22/21.
// m3 winners: A .5 B .5 drift to 5/11 and 6/11; D has no return in m3, so B
// holds everything: 5/11 + 5/11 = 10/11. Losers: D .5 C .5 drift to .575/1.125
// and .55/1.125; m3 holds A .5 C .5, so the sum is 46/45.
func TestMomentumTurnoverOneMonthHold(t *testing.T) {
	res, err := MomentumPortfolios(turnoverPanel(), insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Turnover.GetColByName("date").Data(); len(got) != 3 || got[0] != "m1" || got[2] != "m3" {
		t.Fatalf("turnover dates %v, want the dates of Groups", got)
	}
	nan := math.NaN()
	expectCloseNaN(t, "group_1", columnFloats(t, res.Turnover, "group_1"), []float64{nan, 22.0 / 21, 46.0 / 45})
	expectCloseNaN(t, "group_2", columnFloats(t, res.Turnover, "group_2"), []float64{nan, 1, 10.0 / 11})
	expectCloseNaN(t, "long_short", listFloats(res.LongShortTurnover), []float64{nan, 1 + 22.0/21, 10.0/11 + 46.0/45})
	if res.LongShortTurnover.GetName() != "long_short_turnover" {
		t.Errorf("name %q", res.LongShortTurnover.GetName())
	}
}

// Hold 2: the weights of the overlapping cohorts add up per stock before the
// difference is taken. m2 winners: cohort m1 A .25 B .25 plus cohort m0 A .25
// D .25 is A .5 B .25 D .25, which drifts by m2's returns to A .5, B .3,
// D .2875 over 1.0875. m3 winners: cohort m2 holds B alone (D has no return),
// .5, and cohort m1 holds A .25 B .25, so A .25 B .75. Turnover 55/58.
func TestMomentumTurnoverSumsOverlappingCohorts(t *testing.T) {
	res, err := MomentumPortfolios(turnoverPanel(), insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2, MomentumOptions{Hold: 2})
	if err != nil {
		t.Fatal(err)
	}
	expectCloseNaN(t, "group_2", columnFloats(t, res.Turnover, "group_2"), []float64{math.NaN(), 55.0 / 58})
}

// A month after a skipped one has no turnover: the previous holdings are unknown.
func TestMomentumTurnoverIsNaNAfterASkippedMonth(t *testing.T) {
	// In m1 only A and B have a return, so the m1 portfolio of 4 groups
	// cannot be formed and m2 is skipped; m3 follows a skipped month.
	data := map[string][]any{
		"A": {0.10, 0.01, 0.02, 0.03, 0.01},
		"B": {0.20, 0.02, 0.01, 0.02, 0.02},
		"C": {0.30, nil, 0.03, 0.01, 0.03},
		"D": {0.40, nil, 0.04, 0.04, 0.01},
	}
	dates, stocks, rets := []any{}, []any{}, []any{}
	for m := 0; m < 5; m++ {
		for _, s := range []string{"A", "B", "C", "D"} {
			dates = append(dates, "m"+string(rune('0'+m)))
			stocks = append(stocks, s)
			rets = append(rets, data[s][m])
		}
	}
	dt := insyra.NewDataTable(
		insyra.NewDataList(dates...).SetName("date"),
		insyra.NewDataList(stocks...).SetName("stock"),
		insyra.NewDataList(rets...).SetName("ret"),
	)
	res, err := MomentumPortfolios(dt, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Date != "m2" {
		t.Fatalf("skipped %v, want m2", res.Skipped)
	}
	got := res.Turnover.GetColByName("date").Data()
	if len(got) != 3 || got[0] != "m1" || got[1] != "m3" || got[2] != "m4" {
		t.Fatalf("turnover dates %v", got)
	}
	g1 := columnFloats(t, res.Turnover, "group_1")
	if !math.IsNaN(g1[0]) || !math.IsNaN(g1[1]) || math.IsNaN(g1[2]) {
		t.Fatalf("group_1 turnover %v, want NaN in m1 and m3 only", g1)
	}
}

// A group with no return in a month has no weights there, so neither that
// month nor the next has a turnover.
func TestMomentumTurnoverIsNaNAroundAGroupWithNoReturn(t *testing.T) {
	// The winners are D and B from m0's and m1's rankings, and neither has a
	// return in m2; m2's ranking has A and C only.
	data := map[string][]any{
		"A": {0.10, 0.01, 0.02, 0.03},
		"B": {0.40, 0.04, nil, 0.02},
		"C": {0.20, 0.02, 0.03, 0.01},
		"D": {0.30, 0.03, nil, 0.04},
	}
	dates, stocks, rets := []any{}, []any{}, []any{}
	for m := 0; m < 4; m++ {
		for _, s := range []string{"A", "B", "C", "D"} {
			dates = append(dates, "m"+string(rune('0'+m)))
			stocks = append(stocks, s)
			rets = append(rets, data[s][m])
		}
	}
	dt := insyra.NewDataTable(
		insyra.NewDataList(dates...).SetName("date"),
		insyra.NewDataList(stocks...).SetName("stock"),
		insyra.NewDataList(rets...).SetName("ret"),
	)
	res, err := MomentumPortfolios(dt, insyra.Name("date"), insyra.Name("stock"), insyra.Name("ret"), 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if g2 := columnFloats(t, res.Groups, "group_2"); !math.IsNaN(g2[1]) {
		t.Fatalf("group_2 return %v, want NaN in m2", g2)
	}
	tg2 := columnFloats(t, res.Turnover, "group_2")
	if !math.IsNaN(tg2[1]) || !math.IsNaN(tg2[2]) {
		t.Fatalf("group_2 turnover %v, want NaN in m2 and m3", tg2)
	}
	ls := listFloats(res.LongShortTurnover)
	if !math.IsNaN(ls[1]) || !math.IsNaN(ls[2]) {
		t.Fatalf("long-short turnover %v, want NaN in m2 and m3", ls)
	}
	// The losers A and C are held in m1 and m2, then A alone in m3.
	// m2: A .5 C .5 drift by m1's returns to .505/1.015 and .51/1.015.
	// m3: A .5 C .5 drift by m2's returns to .51 and .515 over 1.025; A holds 1.
	wantM2 := math.Abs(.5-.505/1.015) + math.Abs(.5-.51/1.015)
	wantM3 := (1 - .51/1.025) + .515/1.025
	expectCloseNaN(t, "group_1", columnFloats(t, res.Turnover, "group_1"), []float64{math.NaN(), wantM2, wantM3})
}

func listFloats(dl *insyra.DataList) []float64 {
	out := []float64{}
	for _, v := range dl.Data() {
		f, _ := insyra.ToFloat64Safe(v)
		out = append(out, f)
	}
	return out
}

func expectCloseNaN(t *testing.T, name string, got, want []float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", name, got, want)
	}
	for i := range want {
		if math.IsNaN(want[i]) != math.IsNaN(got[i]) || (!math.IsNaN(want[i]) && math.Abs(got[i]-want[i]) > 1e-12) {
			t.Fatalf("%s: row %d: got %v, want %v (all %v)", name, i, got[i], want[i], got)
		}
	}
}
