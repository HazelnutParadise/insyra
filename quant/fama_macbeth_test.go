package quant

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
	"gonum.org/v1/gonum/stat/distuv"
)

// fmbPanel builds a long table of days × stocks whose return is
// 1 + 2·x1 − 3·x2 + noise(day, stock), with x1 and x2 varying per stock and day.
func fmbPanel(days, stocks int, noise func(d, s int) float64) *insyra.DataTable {
	return fmbPanelEdit(days, stocks, noise, nil)
}

// fmbPanelEdit is fmbPanel with edit applied to the columns (date, ret, x1,
// x2) before the table is built.
func fmbPanelEdit(days, stocks int, noise func(d, s int) float64, edit func(cols [4][]any)) *insyra.DataTable {
	var date, ret, x1, x2 []any
	for d := 0; d < days; d++ {
		for s := 0; s < stocks; s++ {
			a := float64((s*7+d*3)%11) - 5
			b := float64((s*s+d)%13) / 4
			date = append(date, d)
			x1 = append(x1, a)
			x2 = append(x2, b)
			ret = append(ret, 1+2*a-3*b+noise(d, s))
		}
	}
	if edit != nil {
		edit([4][]any{date, ret, x1, x2})
	}
	return insyra.NewDataTable(
		insyra.NewDataList(date...).SetName("date"),
		insyra.NewDataList(ret...).SetName("ret"),
		insyra.NewDataList(x1...).SetName("x1"),
		insyra.NewDataList(x2...).SetName("x2"),
	)
}

func nearRel(a, b, rel float64) bool {
	return math.Abs(a-b) <= rel*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

func TestFamaMacBethExactCrossSection(t *testing.T) {
	res, err := FamaMacBeth(fmbPanel(5, 8, func(int, int) float64 { return 0 }), insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")})
	if err != nil {
		t.Fatal(err)
	}
	if !nearRel(res.Intercept, 1, 1e-12) || !nearRel(res.Coefs[0], 2, 1e-12) || !nearRel(res.Coefs[1], -3, 1e-12) {
		t.Errorf("coefficients: %v %v", res.Intercept, res.Coefs)
	}
	for _, r2 := range column(t, res.Daily, "r2") {
		if !nearRel(r2, 1, 1e-12) {
			t.Errorf("daily R²: %v", r2)
		}
	}
	if !nearRel(res.MeanR2, 1, 1e-12) || res.Days != 5 || len(res.Skipped) != 0 {
		t.Errorf("MeanR2 %v, Days %d, Skipped %v", res.MeanR2, res.Days, res.Skipped)
	}
	if got := res.Daily.ColNames(); strings.Join(got, ",") != "date,intercept,x1,x2,r2,count" {
		t.Errorf("daily columns: %v", got)
	}
	if got := column(t, res.Daily, "count"); got[0] != 8 {
		t.Errorf("count: %v", got)
	}
}

type fmbBaseline struct {
	Rows [][]*float64 `json:"rows"`
	Fits map[string]struct {
		Params    []float64 `json:"params"`
		StdErrors []float64 `json:"std_errors"`
		TStats    []float64 `json:"tstats"`
	} `json:"fits"`
}

func TestFamaMacBethMatchesLinearmodels(t *testing.T) {
	raw, err := os.ReadFile("testdata/fama_macbeth_baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var base fmbBaseline
	if err := json.Unmarshal(raw, &base); err != nil {
		t.Fatal(err)
	}
	cols := make([][]any, 5)
	for _, row := range base.Rows {
		for c, v := range row {
			if v == nil {
				cols[c] = append(cols[c], nil)
			} else {
				cols[c] = append(cols[c], *v)
			}
		}
	}
	table := insyra.NewDataTable(
		insyra.NewDataList(cols[0]...).SetName("date"),
		insyra.NewDataList(cols[1]...).SetName("stock"),
		insyra.NewDataList(cols[2]...).SetName("ret"),
		insyra.NewDataList(cols[3]...).SetName("mom"),
		insyra.NewDataList(cols[4]...).SetName("size"),
	)
	for lagName, lag := range map[string]int{"0": 0, "3": 3, "5": 5} {
		fit := base.Fits[lagName]
		res, err := FamaMacBeth(table, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("mom"), insyra.Name("size")}, FamaMacBethOptions{NeweyWestLag: lag, NoNeweyWest: lagName == "0"})
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Skipped) != 1 || res.Days != 59 {
			t.Fatalf("lag %s: Days %d, Skipped %v", lagName, res.Days, res.Skipped)
		}
		got := [][3]float64{{res.Intercept, res.InterceptStdErr, res.InterceptTValue}}
		for j := range res.Coefs {
			got = append(got, [3]float64{res.Coefs[j], res.StdErrs[j], res.TValues[j]})
		}
		for j, g := range got {
			want := [3]float64{fit.Params[j], fit.StdErrors[j], fit.TStats[j]}
			for m := range g {
				if math.Abs(g[m]-want[m]) > 1e-9*math.Abs(want[m]) {
					t.Errorf("lag %s, parameter %d, field %d: got %.17g, linearmodels %.17g", lagName, j, m, g[m], want[m])
				}
			}
		}
		tdist := distuv.StudentsT{Mu: 0, Sigma: 1, Nu: float64(res.Days - 1)}
		for j, tv := range res.TValues {
			if want := 2 * tdist.Survival(math.Abs(tv)); !nearRel(res.PValues[j], want, 1e-12) {
				t.Errorf("lag %s, p value %d: got %v, want %v", lagName, j, res.PValues[j], want)
			}
		}
		if res.Lag != lag {
			t.Errorf("lag %s: Lag %d", lagName, res.Lag)
		}
	}
}

// NoNeweyWest is the t-test the old negative lag produced: lag 0, variance
// divided by T, not the T-1 of a one-sample t-test.
func TestFamaMacBethNoNeweyWest(t *testing.T) {
	table := fmbPanel(30, 8, func(d, s int) float64 { return math.Sin(float64(d*s + d)) })
	res, err := FamaMacBeth(table, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")}, FamaMacBethOptions{NoNeweyWest: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Lag != 0 {
		t.Errorf("Lag %d, want 0", res.Lag)
	}
	for j, name := range []string{"x1", "x2"} {
		want, err := stats.NeweyWestMeanTest(newFloat64DataList(column(t, res.Daily, name)), 0)
		if err != nil {
			t.Fatal(err)
		}
		if res.Coefs[j] != want.Mean || res.StdErrs[j] != want.StdErr || res.TValues[j] != want.Statistic || res.PValues[j] != want.PValue {
			t.Errorf("%s: got %v %v %v %v, want lag-0 test %+v", name, res.Coefs[j], res.StdErrs[j], res.TValues[j], res.PValues[j], want)
		}
	}
}

func TestFamaMacBethAutomaticLag(t *testing.T) {
	table := fmbPanel(30, 8, func(d, s int) float64 { return math.Sin(float64(d*s + d)) })
	res, err := FamaMacBeth(table, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")})
	if err != nil {
		t.Fatal(err)
	}
	if want := int(math.Floor(4 * math.Pow(30.0/100, 2.0/9.0))); res.Lag != want {
		t.Errorf("automatic lag %d, want %d", res.Lag, want)
	}
}

func TestFamaMacBethSkipsThinAndCollinearDays(t *testing.T) {
	// Day 1: x2 constant. Day 2: only three stocks have a return.
	table := fmbPanelEdit(4, 8, func(d, s int) float64 { return math.Cos(float64(d + 3*s)) }, func(cols [4][]any) {
		for i := 8; i < 16; i++ {
			cols[3][i] = 7.0
		}
		for i := 19; i < 24; i++ {
			cols[1][i] = nil
		}
	})
	res, err := FamaMacBeth(table, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 2 || res.Skipped[0].Date != 1 || !strings.Contains(res.Skipped[0].Reason, "collinear") ||
		res.Skipped[1].Date != 2 || !strings.Contains(res.Skipped[1].Reason, "fewer than 4") {
		t.Errorf("skipped: %v", res.Skipped)
	}
	if res.Days != 2 {
		t.Errorf("days: %d", res.Days)
	}

	// MinObservations raises the floor.
	if _, err := FamaMacBeth(fmbPanel(3, 8, func(int, int) float64 { return 0 }), insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")}, FamaMacBethOptions{MinObservations: 9}); err == nil {
		t.Error("no day has 9 stocks, want an error")
	}
}

func TestFamaMacBethStandardize(t *testing.T) {
	noise := func(d, s int) float64 { return math.Sin(float64(5*d + s)) }
	plain, err := FamaMacBeth(fmbPanel(6, 10, noise), insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")}, FamaMacBethOptions{Standardize: true})
	if err != nil {
		t.Fatal(err)
	}
	scaled := fmbPanelEdit(6, 10, noise, func(cols [4][]any) {
		for i, v := range cols[2] {
			cols[2][i] = 10*v.(float64) + 4
		}
	})
	res, err := FamaMacBeth(scaled, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")}, FamaMacBethOptions{Standardize: true})
	if err != nil {
		t.Fatal(err)
	}
	for j := range res.Coefs {
		if !nearRel(res.Coefs[j], plain.Coefs[j], 1e-12) {
			t.Errorf("coefficient %d: %v vs %v", j, res.Coefs[j], plain.Coefs[j])
		}
	}
	// Standardized x1 has slope 2·sd(x1) on each day, not 2.
	if nearRel(res.Coefs[0], 2, 1e-3) {
		t.Errorf("x1 slope %v was not standardized", res.Coefs[0])
	}
}

func TestFamaMacBethColumnLetters(t *testing.T) {
	table := fmbPanel(4, 8, func(d, s int) float64 { return math.Sin(float64(d*s + s)) })
	byName, err := FamaMacBeth(table, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")})
	if err != nil {
		t.Fatal(err)
	}
	byLetter, err := FamaMacBeth(table, "A", "B", []any{"C", "D"})
	if err != nil {
		t.Fatal(err)
	}
	if byName.Coefs[0] != byLetter.Coefs[0] || byName.Coefs[1] != byLetter.Coefs[1] || byLetter.Names[0] != "x1" {
		t.Errorf("by name %v, by letter %v %v", byName.Coefs, byLetter.Coefs, byLetter.Names)
	}
}

// One selector everywhere: a bare string is an Excel-style index even when a
// column is named like another column's letters, Name picks by name, an int is
// a position counting from the end when negative.
func TestFamaMacBethSelectorContract(t *testing.T) {
	base := fmbPanel(4, 8, func(d, s int) float64 { return math.Sin(float64(d*s + s)) })
	want, err := FamaMacBeth(base, "A", "B", []any{"C", "D"})
	if err != nil {
		t.Fatal(err)
	}
	calls := map[string]func() (*FamaMacBethResult, error){
		"names": func() (*FamaMacBethResult, error) {
			return FamaMacBeth(base, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")})
		},
		"positions": func() (*FamaMacBethResult, error) { return FamaMacBeth(base, 0, 1, []any{2, -1}) },
		"mixed":     func() (*FamaMacBethResult, error) { return FamaMacBeth(base, "A", insyra.Name("ret"), []any{2, "D"}) },
	}
	for name, call := range calls {
		res, err := call()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if res.Coefs[0] != want.Coefs[0] || res.Coefs[1] != want.Coefs[1] {
			t.Errorf("%s: %v, want %v", name, res.Coefs, want.Coefs)
		}
		if res.Names[0] != "x1" || res.Names[1] != "x2" {
			t.Errorf("%s: Names %v, want the column names", name, res.Names)
		}
	}
	// Columns named like each other's letters: "A" is still the first column.
	swapped := insyra.NewDataTable(
		insyra.NewDataList(base.GetColByNumber(0).Data()...).SetName("B"),
		insyra.NewDataList(base.GetColByNumber(1).Data()...).SetName("A"),
		insyra.NewDataList(base.GetColByNumber(2).Data()...).SetName("x1"),
		insyra.NewDataList(base.GetColByNumber(3).Data()...).SetName("x2"),
	)
	got, err := FamaMacBeth(swapped, "A", "B", []any{"C", "D"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Coefs[0] != want.Coefs[0] {
		t.Errorf("a column named A captured the bare string: %v, want %v", got.Coefs, want.Coefs)
	}
	for _, bad := range []any{nil, 1.5, "date", insyra.Name("nope"), 9, "Z"} {
		if _, err := FamaMacBeth(base, bad, "B", []any{"C"}); err == nil {
			t.Errorf("date selector %v: want an error", bad)
		}
	}
	if _, err := FamaMacBeth(base, "A", "B", []any{nil}); err == nil {
		t.Error("nil characteristic selector: want an error")
	}
	// The reserved check applies to the resolved name, whatever the selector.
	reserved := insyra.NewDataTable(
		insyra.NewDataList(base.GetColByNumber(0).Data()...).SetName("d"),
		insyra.NewDataList(base.GetColByNumber(1).Data()...).SetName("ret"),
		insyra.NewDataList(base.GetColByNumber(2).Data()...).SetName("r2"),
	)
	if _, err := FamaMacBeth(reserved, "A", "B", []any{"C"}); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Errorf("reserved name picked by letter: error %v", err)
	}
	if _, err := FamaMacBeth(reserved, 0, 1, []any{-1}); err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Errorf("reserved name picked by position: error %v", err)
	}
}

// A characteristic named like a fixed Daily column would make NewDataTable
// rename one of the two, so the documented schema would not hold.
func TestFamaMacBethRefusesReservedCharacteristicNames(t *testing.T) {
	for _, name := range []string{"date", "intercept", "r2", "count"} {
		var date, ret, x []any
		for d := 0; d < 4; d++ {
			for s := 0; s < 8; s++ {
				a := float64((s*7+d*3)%11) - 5
				date, x, ret = append(date, d), append(x, a), append(ret, 1+2*a+float64(s%3))
			}
		}
		tables := map[string]*insyra.DataTable{
			"date": insyra.NewDataTable(insyra.NewDataList(date...).SetName("d"), insyra.NewDataList(ret...).SetName("ret"), insyra.NewDataList(x...).SetName(name)),
		}
		_, err := FamaMacBeth(tables["date"], insyra.Name("d"), insyra.Name("ret"), []any{insyra.Name(name)})
		if err == nil || !strings.Contains(err.Error(), "reserved") {
			t.Errorf("characteristic %q: error %v, want one containing %q", name, err, "reserved")
		}
	}
	// The date column itself is not a characteristic either.
	_, err := FamaMacBeth(fmbPanel(4, 8, func(int, int) float64 { return 0 }), insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), "A"})
	if err == nil || !strings.Contains(err.Error(), "date column") {
		t.Errorf("date as characteristic: error %v", err)
	}
}

// Dates are grouped by value: strings that differ only by whitespace are
// different dates.
func TestFamaMacBethKeepsWhitespaceDatesApart(t *testing.T) {
	table := fmbPanelEdit(6, 8, func(d, s int) float64 { return math.Sin(float64(d*s + d)) }, func(cols [4][]any) {
		for i, d := range cols[0] {
			if d.(int)%2 == 1 {
				cols[0][i] = " day" + fmt.Sprint(d.(int)-1) + " "
			} else {
				cols[0][i] = "day" + fmt.Sprint(d.(int))
			}
		}
	})
	res, err := FamaMacBeth(table, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1"), insyra.Name("x2")})
	if err != nil {
		t.Fatal(err)
	}
	if res.Days != 6 {
		t.Errorf("Days %d, want 6: dates differing by whitespace were merged", res.Days)
	}
}

// UnixNano wraps outside 1678-2262; two distinct instants share a nanosecond
// count there and must still be different dates.
func TestFamaMacBethDateKeyDistinguishesFarInstants(t *testing.T) {
	a := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	b := time.Date(585, 7, 21, 23, 34, 33, 709551616, time.UTC)
	if a.UnixNano() != b.UnixNano() {
		t.Skip("UnixNano no longer collides for these instants")
	}
	ka, _ := famaMacBethDateKey(a)
	kb, _ := famaMacBethDateKey(b)
	kz, _ := famaMacBethDateKey(a.In(time.FixedZone("x", 3600)))
	if ka == kb {
		t.Errorf("distinct instants share the key %q", ka)
	}
	if ka != kz {
		t.Error("the same instant in another zone got a different key")
	}
}

type fmbStringer struct{ n int }

func (fmbStringer) String() string { return "same" }

type fmbMyInt int

// A date is a string, time.Time, bool, integer or float; anything whose
// printed form could be shared by two values is refused.
func TestFamaMacBethRefusesUnsupportedDateTypes(t *testing.T) {
	one := 1
	for name, bad := range map[string]any{
		"slice":    []string{"a b"},
		"stringer": fmbStringer{1},
		"named":    fmbMyInt(1),
		"pointer":  &one,
		"map":      map[string]int{"a": 1},
		"NaN":      math.NaN(),
	} {
		table := fmbPanelEdit(4, 8, func(int, int) float64 { return 0 }, func(cols [4][]any) { cols[0][5] = insyra.Cell(bad) })
		_, err := FamaMacBeth(table, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1")})
		if err == nil {
			t.Errorf("%s: no error", name)
		} else if name != "NaN" && !strings.Contains(err.Error(), "not a supported date type") {
			t.Errorf("%s: error %v", name, err)
		}
	}
	for name, ok := range map[string]any{"int64": int64(1), "float": 1.5, "bool": true, "uint8": uint8(1), "string": "x", "time": time.Unix(1, 0)} {
		table := fmbPanelEdit(4, 8, func(d, s int) float64 { return math.Sin(float64(d*s + 1)) }, func(cols [4][]any) {
			for i := range cols[0] {
				cols[0][i] = ok
				if i >= 16 {
					cols[0][i] = "other"
				}
			}
		})
		if _, err := FamaMacBeth(table, insyra.Name("date"), insyra.Name("ret"), []any{insyra.Name("x1")}); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestFamaMacBethRefusesDateAsReturn(t *testing.T) {
	table := fmbPanel(4, 8, func(int, int) float64 { return 0 })
	for _, ret := range []any{insyra.Name("date"), "A", 0} {
		_, err := FamaMacBeth(table, insyra.Name("date"), ret, []any{insyra.Name("x1")})
		if err == nil || !strings.Contains(err.Error(), "same column") {
			t.Errorf("return %v: error %v, want one containing %q", ret, err, "same column")
		}
	}
}

func TestFamaMacBethRefusesBadInput(t *testing.T) {
	good := func() *insyra.DataTable { return fmbPanel(4, 8, func(int, int) float64 { return 0 }) }
	set := func(col int, v any) *insyra.DataTable {
		return fmbPanelEdit(4, 8, func(int, int) float64 { return 0 }, func(cols [4][]any) { cols[col][3] = v })
	}
	text, inf, noDate := set(2, "abc"), set(1, math.Inf(1)), set(0, nil)

	cases := []struct {
		name    string
		table   insyra.IDataTable
		chars   []any
		opts    []FamaMacBethOptions
		message string
	}{
		{"nil table", nil, []any{insyra.Name("x1")}, nil, "nil"},
		{"no characteristic", good(), nil, nil, "characteristic"},
		{"repeated characteristic", good(), []any{insyra.Name("x1"), "C"}, nil, "same column"},
		{"return as characteristic", good(), []any{insyra.Name("ret")}, nil, "return column"},
		{"missing column", good(), []any{insyra.Name("size")}, nil, `"size"`},
		{"text value", text, []any{insyra.Name("x1")}, nil, "non-numeric"},
		{"infinite value", inf, []any{insyra.Name("x1")}, nil, "non-finite"},
		{"missing date", noDate, []any{insyra.Name("x1")}, nil, "date is missing"},
		{"two options", good(), []any{insyra.Name("x1")}, []FamaMacBethOptions{{}, {}}, "at most one"},
		{"negative lag", good(), []any{insyra.Name("x1")}, []FamaMacBethOptions{{NeweyWestLag: -1}}, "NeweyWestLag"},
		{"no correction with a lag", good(), []any{insyra.Name("x1")}, []FamaMacBethOptions{{NoNeweyWest: true, NeweyWestLag: 2}}, "NoNeweyWest"},
		{"lag too long", good(), []any{insyra.Name("x1")}, []FamaMacBethOptions{{NeweyWestLag: 4}}, "lag"},
	}
	for _, c := range cases {
		_, err := FamaMacBeth(c.table, insyra.Name("date"), insyra.Name("ret"), c.chars, c.opts...)
		if err == nil || !strings.Contains(err.Error(), c.message) {
			t.Errorf("%s: error %v, want one containing %q", c.name, err, c.message)
		}
	}
}
