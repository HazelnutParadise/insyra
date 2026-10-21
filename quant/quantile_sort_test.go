package quant

import (
	"math"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// panel builds a long table: two days, six stocks a day, signal 1..6 on day one
// and 6..1 on day two, with the return equal to ten times the signal.
func panel() *insyra.DataTable {
	var date, code, signal, ret, weight []any
	for d, day := range []string{"2026-01-02", "2026-01-05"} {
		for s := 1; s <= 6; s++ {
			sg := float64(s)
			if d == 1 {
				sg = float64(7 - s)
			}
			date = append(date, day)
			code = append(code, s)
			signal = append(signal, sg)
			ret = append(ret, 10*sg)
			weight = append(weight, float64(s))
		}
	}
	return insyra.NewDataTable(
		insyra.NewDataList(date...).SetName("date"),
		insyra.NewDataList(code...).SetName("code"),
		insyra.NewDataList(signal...).SetName("signal"),
		insyra.NewDataList(ret...).SetName("ret"),
		insyra.NewDataList(weight...).SetName("weight"),
	)
}

func column(t *testing.T, dt *insyra.DataTable, name string) []float64 {
	t.Helper()
	return floats(t, dt.GetColByName(name))
}

func TestQuantileSortTertiles(t *testing.T) {
	res, err := QuantileSort(panel(), "date", "signal", []string{"ret"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	// Day one: groups {1,2},{3,4},{5,6} with mean returns 15, 35, 55.
	// Day two has the same signals the other way round, but a group is by signal, so identical.
	if got := column(t, res.Groups, "ret"); len(got) != 6 || got[0] != 15 || got[1] != 35 || got[2] != 55 || got[3] != 15 || got[5] != 55 {
		t.Errorf("group means: %v", got)
	}
	if got := column(t, res.Groups, "count"); got[0] != 2 || got[4] != 2 {
		t.Errorf("counts: %v", got)
	}
	if got := column(t, res.Spread, "spread_ret"); len(got) != 2 || got[0] != 40 || got[1] != 40 {
		t.Errorf("spread: %v", got)
	}
	if len(res.Skipped) != 0 {
		t.Errorf("skipped: %v", res.Skipped)
	}
}

func TestQuantileSortWeightsAndDemean(t *testing.T) {
	res, err := QuantileSort(panel(), "date", "signal", []string{"ret"}, 3, QuantileSortOptions{WeightCol: "weight"})
	if err != nil {
		t.Fatal(err)
	}
	// Day one, weights equal the signal: group 1 = (10*1+20*2)/3, group 3 = (50*5+60*6)/11.
	got := column(t, res.Groups, "ret")
	if math.Abs(got[0]-50.0/3) > 1e-12 || math.Abs(got[2]-610.0/11) > 1e-12 {
		t.Errorf("weighted means: %v", got)
	}
	demeaned, err := QuantileSort(panel(), "date", "signal", []string{"ret"}, 3, QuantileSortOptions{Demean: true})
	if err != nil {
		t.Fatal(err)
	}
	// The day's mean return is 35, so the groups are -20, 0, 20 and the spread is unchanged.
	g := column(t, demeaned.Groups, "ret")
	if g[0] != -20 || g[1] != 0 || g[2] != 20 {
		t.Errorf("demeaned: %v", g)
	}
	if s := column(t, demeaned.Spread, "spread_ret"); s[0] != 40 {
		t.Errorf("spread: %v", s)
	}
}

func TestQuantileSortSkipsThinDaysAndMissingSignals(t *testing.T) {
	tbl := insyra.NewDataTable(
		insyra.NewDataList("d1", "d1", "d1", "d2", "d2", "d2", "d2").SetName("date"),
		insyra.NewDataList(1.0, 2.0, 3.0, 1.0, nil, math.NaN(), 4.0).SetName("signal"),
		insyra.NewDataList(1.0, 2.0, 3.0, 1.0, 2.0, 3.0, 4.0).SetName("ret"),
	)
	res, err := QuantileSort(tbl, "date", "signal", []string{"ret"}, 2, QuantileSortOptions{MinStocks: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Date != "d2" {
		t.Errorf("skipped: %+v", res.Skipped)
	}
	if n := res.Spread.NumRows(); n != 1 {
		t.Errorf("spread rows: %d", n)
	}
}

func TestQuantileSortTies(t *testing.T) {
	// Six stocks, signal 1,1,1,1,2,3 into 2 groups: average ranks 2.5 x4, 5, 6 -> groups 1,1,1,1,2,2.
	tbl := insyra.NewDataTable(
		insyra.NewDataList("d", "d", "d", "d", "d", "d").SetName("date"),
		insyra.NewDataList(1.0, 1.0, 1.0, 1.0, 2.0, 3.0).SetName("signal"),
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0).SetName("ret"),
	)
	res, err := QuantileSort(tbl, "date", "signal", []string{"ret"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if c := column(t, res.Groups, "count"); c[0] != 4 || c[1] != 2 {
		t.Errorf("rank ties: counts %v", c)
	}
	// qcut on the same signal: the median edge equals the minimum, so the edges
	// coincide and pandas.qcut would raise; the day is skipped.
	q, err := QuantileSort(tbl, "date", "signal", []string{"ret"}, 2, QuantileSortOptions{Ties: QuantileTiesQcut})
	if err != nil || len(q.Skipped) != 1 {
		t.Errorf("qcut with coinciding edges: %+v, %v", q, err)
	}
	// Signal 1,2,2,3,4,5: edges 1, 2.5, 5, so (1..2.5] is group 1 and (2.5..5] group 2.
	cut := insyra.NewDataTable(
		insyra.NewDataList("d", "d", "d", "d", "d", "d").SetName("date"),
		insyra.NewDataList(1.0, 2.0, 2.0, 3.0, 4.0, 5.0).SetName("signal"),
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0).SetName("ret"),
	)
	qc, err := QuantileSort(cut, "date", "signal", []string{"ret"}, 2, QuantileSortOptions{Ties: QuantileTiesQcut})
	if err != nil {
		t.Fatal(err)
	}
	if c := column(t, qc.Groups, "count"); c[0] != 3 || c[1] != 3 {
		t.Errorf("qcut: counts %v", c)
	}
	// All signals equal: qcut cannot cut, rank ties put everything in one group.
	flat := insyra.NewDataTable(
		insyra.NewDataList("d", "d", "d", "d").SetName("date"),
		insyra.NewDataList(1.0, 1.0, 1.0, 1.0).SetName("signal"),
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0).SetName("ret"),
	)
	skipped, err := QuantileSort(flat, "date", "signal", []string{"ret"}, 2, QuantileSortOptions{Ties: QuantileTiesQcut})
	if err != nil || len(skipped.Skipped) != 1 {
		t.Errorf("qcut on a constant signal: %+v, %v", skipped, err)
	}
}

func TestQuantileSortAcceptsColumnLetters(t *testing.T) {
	res, err := QuantileSort(panel(), "A", "C", []string{"D"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if res.Groups.NumRows() != 6 {
		t.Errorf("rows: %d", res.Groups.NumRows())
	}
}

func TestQuantileSortRefusesBadInput(t *testing.T) {
	var nilTable *insyra.DataTable
	cases := map[string]func() error{
		"n":          func() error { _, e := QuantileSort(panel(), "date", "signal", []string{"ret"}, 1); return e },
		"no returns": func() error { _, e := QuantileSort(panel(), "date", "signal", nil, 3); return e },
		"nil":        func() error { _, e := QuantileSort(nilTable, "date", "signal", []string{"ret"}, 3); return e },
		"no column":  func() error { _, e := QuantileSort(panel(), "day", "signal", []string{"ret"}, 3); return e },
		"ties": func() error {
			_, e := QuantileSort(panel(), "date", "signal", []string{"ret"}, 3, QuantileSortOptions{Ties: 9})
			return e
		},
		"bad weight": func() error {
			tbl := insyra.NewDataTable(
				insyra.NewDataList("d", "d").SetName("date"), insyra.NewDataList(1.0, 2.0).SetName("signal"),
				insyra.NewDataList(1.0, 2.0).SetName("ret"), insyra.NewDataList(1.0, -1.0).SetName("w"))
			_, e := QuantileSort(tbl, "date", "signal", []string{"ret"}, 2, QuantileSortOptions{WeightCol: "w"})
			return e
		},
		"text signal": func() error {
			tbl := insyra.NewDataTable(
				insyra.NewDataList("d", "d").SetName("date"), insyra.NewDataList(1.0, "x").SetName("signal"),
				insyra.NewDataList(1.0, 2.0).SetName("ret"))
			_, e := QuantileSort(tbl, "date", "signal", []string{"ret"}, 2)
			return e
		},
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func dateKeyTable(dates []any) *insyra.DataTable {
	n := len(dates)
	sig, ret := make([]any, n), make([]any, n)
	for i := range dates {
		sig[i] = float64(i%4 + 1)
		ret[i] = float64(i%4+1) * 10
	}
	return insyra.NewDataTable(insyra.NewDataList().Append(dates...).SetName("date"), insyra.NewDataList(sig...).SetName("signal"), insyra.NewDataList(ret...).SetName("ret"))
}

func quantileSortDateCount(t *testing.T, dates []any) int {
	t.Helper()
	res, err := QuantileSort(dateKeyTable(dates), "date", "signal", []string{"ret"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	rows, _ := res.Spread.Size()
	return rows
}

func TestQuantileSortWhitespaceDatesStayDifferent(t *testing.T) {
	d := []any{"2026-01-02", "2026-01-02", "2026-01-02", "2026-01-02", " 2026-01-02 ", " 2026-01-02 ", " 2026-01-02 ", " 2026-01-02 "}
	if got := quantileSortDateCount(t, d); got != 2 {
		t.Errorf("dates = %d, want 2", got)
	}
}

func TestQuantileSortFarInstantsStayDifferent(t *testing.T) {
	a := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(585, 7, 21, 23, 34, 33, 709551616, time.UTC)
	d := []any{a, a, a, a, b, b, b, b}
	if got := quantileSortDateCount(t, d); got != 2 {
		t.Errorf("dates = %d, want 2", got)
	}
}

func TestQuantileSortSameInstantInOtherZoneIsOneDate(t *testing.T) {
	a := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	z := a.In(time.FixedZone("x", 3600))
	d := []any{a, a, z, z, a, a, z, z}
	if got := quantileSortDateCount(t, d); got != 1 {
		t.Errorf("dates = %d, want 1", got)
	}
}

func TestQuantileSortRefusesUnsupportedDateTypes(t *testing.T) {
	for _, bad := range []any{[]int{1}, struct{}{}, math.NaN()} {
		d := []any{bad, bad, bad, bad}
		if _, err := QuantileSort(dateKeyTable(d), "date", "signal", []string{"ret"}, 2); err == nil {
			t.Errorf("date %T %v: want an error", bad, bad)
		}
	}
}
