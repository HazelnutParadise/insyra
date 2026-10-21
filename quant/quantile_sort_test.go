package quant

import (
	"math"
	"testing"

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
	res, err := QuantileSort(panel(), insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 3)
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
	res, err := QuantileSort(panel(), insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 3, QuantileSortOptions{WeightCol: insyra.Name("weight")})
	if err != nil {
		t.Fatal(err)
	}
	// Day one, weights equal the signal: group 1 = (10*1+20*2)/3, group 3 = (50*5+60*6)/11.
	got := column(t, res.Groups, "ret")
	if math.Abs(got[0]-50.0/3) > 1e-12 || math.Abs(got[2]-610.0/11) > 1e-12 {
		t.Errorf("weighted means: %v", got)
	}
	demeaned, err := QuantileSort(panel(), insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 3, QuantileSortOptions{Demean: true})
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
	res, err := QuantileSort(tbl, insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 2, QuantileSortOptions{MinStocks: 3})
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
	res, err := QuantileSort(tbl, insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if c := column(t, res.Groups, "count"); c[0] != 4 || c[1] != 2 {
		t.Errorf("rank ties: counts %v", c)
	}
	// qcut on the same signal: the median edge equals the minimum, so the edges
	// coincide and pandas.qcut would raise; the day is skipped.
	q, err := QuantileSort(tbl, insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 2, QuantileSortOptions{Ties: QuantileTiesQcut})
	if err != nil || len(q.Skipped) != 1 {
		t.Errorf("qcut with coinciding edges: %+v, %v", q, err)
	}
	// Signal 1,2,2,3,4,5: edges 1, 2.5, 5, so (1..2.5] is group 1 and (2.5..5] group 2.
	cut := insyra.NewDataTable(
		insyra.NewDataList("d", "d", "d", "d", "d", "d").SetName("date"),
		insyra.NewDataList(1.0, 2.0, 2.0, 3.0, 4.0, 5.0).SetName("signal"),
		insyra.NewDataList(1.0, 2.0, 3.0, 4.0, 5.0, 6.0).SetName("ret"),
	)
	qc, err := QuantileSort(cut, insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 2, QuantileSortOptions{Ties: QuantileTiesQcut})
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
	skipped, err := QuantileSort(flat, insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 2, QuantileSortOptions{Ties: QuantileTiesQcut})
	if err != nil || len(skipped.Skipped) != 1 {
		t.Errorf("qcut on a constant signal: %+v, %v", skipped, err)
	}
}

func TestQuantileSortAcceptsColumnLetters(t *testing.T) {
	res, err := QuantileSort(panel(), "A", "C", []any{"D"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if res.Groups.NumRows() != 6 {
		t.Errorf("rows: %d", res.Groups.NumRows())
	}
}

// One selector everywhere: a bare string is an Excel-style index and never a
// name, Name picks by exact name, an int is a position counting from the end
// when negative, and anything else is an error.
func TestQuantileSortSelectorContract(t *testing.T) {
	want, err := QuantileSort(panel(), "A", "C", []any{"D"}, 3, QuantileSortOptions{WeightCol: "E"})
	if err != nil {
		t.Fatal(err)
	}
	same := func(label string, got *QuantileSortResult, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if got.Groups.NumRows() != want.Groups.NumRows() {
			t.Fatalf("%s: rows %d, want %d", label, got.Groups.NumRows(), want.Groups.NumRows())
		}
		a, b := column(t, got.Groups, "ret"), column(t, want.Groups, "ret")
		for i := range a {
			if a[i] != b[i] {
				t.Errorf("%s: group %d mean %v, want %v", label, i, a[i], b[i])
			}
		}
	}
	got, err := QuantileSort(panel(), insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 3, QuantileSortOptions{WeightCol: insyra.Name("weight")})
	same("names", got, err)
	got, err = QuantileSort(panel(), 0, 2, []any{3}, 3, QuantileSortOptions{WeightCol: -1})
	same("positions", got, err)
	got, err = QuantileSort(panel(), "A", insyra.Name("signal"), []any{-2}, 3, QuantileSortOptions{WeightCol: "E"})
	same("mixed", got, err)

	// A column named like another column's letter does not capture the bare
	// string: "C" is still the third column.
	src := panel()
	tbl := insyra.NewDataTable(
		insyra.NewDataList(src.GetColByName("date").Data()...).SetName("date"),
		insyra.NewDataList(src.GetColByName("code").Data()...).SetName("code"),
		insyra.NewDataList(src.GetColByName("signal").Data()...).SetName("signal"),
		insyra.NewDataList(src.GetColByName("ret").Data()...).SetName("C"),
	)
	if res, err := QuantileSort(tbl, "A", "C", []any{"D"}, 3); err != nil {
		t.Fatal(err)
	} else if res.Groups.GetColByName("C") == nil {
		t.Errorf("the return column keeps its own name, got %v", res.Groups.ColNames())
	}
	for label, call := range map[string]func() error{
		"nil date":     func() error { _, e := QuantileSort(panel(), nil, "C", []any{"D"}, 3); return e },
		"float":        func() error { _, e := QuantileSort(panel(), 1.5, "C", []any{"D"}, 3); return e },
		"bare name":    func() error { _, e := QuantileSort(panel(), "signal", "C", []any{"D"}, 3); return e },
		"unknown Name": func() error { _, e := QuantileSort(panel(), "A", insyra.Name("nope"), []any{"D"}, 3); return e },
		"out of range": func() error { _, e := QuantileSort(panel(), "A", "C", []any{9}, 3); return e },
		"past the end": func() error { _, e := QuantileSort(panel(), "A", "C", []any{"Z"}, 3); return e },
		"nil return":   func() error { _, e := QuantileSort(panel(), "A", "C", []any{nil}, 3); return e },
		"bad weight": func() error {
			_, e := QuantileSort(panel(), "A", "C", []any{"D"}, 3, QuantileSortOptions{WeightCol: 1.5})
			return e
		},
	} {
		if call() == nil {
			t.Errorf("%s: want an error", label)
		}
	}
}

func TestQuantileSortRefusesBadInput(t *testing.T) {
	var nilTable *insyra.DataTable
	cases := map[string]func() error{
		"n": func() error {
			_, e := QuantileSort(panel(), insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 1)
			return e
		},
		"no returns": func() error {
			_, e := QuantileSort(panel(), insyra.Name("date"), insyra.Name("signal"), nil, 3)
			return e
		},
		"nil": func() error {
			_, e := QuantileSort(nilTable, insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 3)
			return e
		},
		"no column": func() error {
			_, e := QuantileSort(panel(), insyra.Name("day"), insyra.Name("signal"), []any{insyra.Name("ret")}, 3)
			return e
		},
		"ties": func() error {
			_, e := QuantileSort(panel(), insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 3, QuantileSortOptions{Ties: 9})
			return e
		},
		"bad weight": func() error {
			tbl := insyra.NewDataTable(
				insyra.NewDataList("d", "d").SetName("date"), insyra.NewDataList(1.0, 2.0).SetName("signal"),
				insyra.NewDataList(1.0, 2.0).SetName("ret"), insyra.NewDataList(1.0, -1.0).SetName("w"))
			_, e := QuantileSort(tbl, insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 2, QuantileSortOptions{WeightCol: insyra.Name("w")})
			return e
		},
		"text signal": func() error {
			tbl := insyra.NewDataTable(
				insyra.NewDataList("d", "d").SetName("date"), insyra.NewDataList(1.0, "x").SetName("signal"),
				insyra.NewDataList(1.0, 2.0).SetName("ret"))
			_, e := QuantileSort(tbl, insyra.Name("date"), insyra.Name("signal"), []any{insyra.Name("ret")}, 2)
			return e
		},
	}
	for name, f := range cases {
		if f() == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}
