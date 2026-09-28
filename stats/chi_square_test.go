package stats_test

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

const (
	tolChi  = 1e-12
	tolChiP = 1e-12
)

var chiRef = &refTable{path: "testdata/chi_square_reference.txt"}

// labelledStrings is a string-list counterpart to labelledFloats — reads a
// categorical-data dump file produced by an R script.
type labelledStrings struct {
	once sync.Once
	data map[string][]string
	path string
}

func (l *labelledStrings) get(t *testing.T, label string) []string {
	t.Helper()
	l.once.Do(func() {
		raw, err := os.ReadFile(l.path)
		if err != nil {
			t.Fatalf("read %s: %v", l.path, err)
		}
		l.data = map[string][]string{}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			i := strings.IndexByte(line, ':')
			if i < 0 {
				continue
			}
			name := line[:i]
			parts := strings.Split(line[i+1:], ",")
			vals := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					vals = append(vals, p)
				}
			}
			l.data[name] = vals
		}
	})
	v, ok := l.data[label]
	if !ok {
		t.Fatalf("label %q not found in %s", label, l.path)
	}
	return v
}

var chiDump = &labelledStrings{path: "testdata/chi_square_data_dump.txt"}

func chiClose(a, b, tol float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if b == 0 {
		return math.Abs(a) <= tol
	}
	return math.Abs(a-b) <= tol*math.Max(1, math.Abs(b))
}

// asAnyStrings converts a string slice to []any so it can be passed to
// insyra.NewDataList (which accepts variadic data of arbitrary types).
func asAnyStrings(ss []string) insyra.IDataList {
	dl := insyra.NewDataList()
	for _, s := range ss {
		dl.Append(s)
	}
	return dl
}

// ============================================================
// Goodness-of-fit
// ============================================================

func TestChiSquareGoodnessOfFit_R(t *testing.T) {
	cases := []struct {
		name    string
		data    []string
		p       map[string]float64
		rescale bool
		prefix  string
		nCats   int
	}{
		{name: "uniform_no_p",
			data:   []string{"A", "A", "A", "B", "B", "C", "D", "D", "D", "D"},
			prefix: "gof_uniform", nCats: 4},
		{name: "custom_probs",
			data:   []string{"red", "red", "blue", "green", "blue", "red", "green", "red", "blue", "red"},
			p:      map[string]float64{"blue": 0.5, "green": 0.3, "red": 0.2},
			prefix: "gof_custom", nCats: 3},
		{name: "rescale_true",
			data:    []string{"A", "B", "B", "C", "C", "C", "D", "D", "D", "D"},
			p:       map[string]float64{"A": 1, "B": 2, "C": 3, "D": 4},
			rescale: true,
			prefix:  "gof_rescale", nCats: 4},
		{name: "largeN_two_cats",
			data:   chiDump.get(t, "gof_largeN"),
			prefix: "gof_largeN", nCats: 2},
		{name: "many_categories",
			data: func() []string {
				out := []string{}
				cats := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
				counts := []int{12, 8, 15, 10, 7, 13, 9, 11}
				for i, c := range cats {
					for range counts[i] {
						out = append(out, c)
					}
				}
				return out
			}(),
			prefix: "gof_many", nCats: 8},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.ChiSquareGoodnessOfFit(asAnyStrings(c.data), c.p, c.rescale)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expChi := chiRef.get(t, c.prefix+".chi")
			expP := chiRef.get(t, c.prefix+".p")
			expDF := int(chiRef.get(t, c.prefix+".df"))

			if !chiClose(r.Statistic, expChi, tolChi) {
				t.Errorf("chi: got %.17g, want %.17g (Δ=%g)", r.Statistic, expChi, math.Abs(r.Statistic-expChi))
			}
			if !chiClose(r.PValue, expP, tolChiP) {
				t.Errorf("p: got %.17g, want %.17g (Δ=%g)", r.PValue, expP, math.Abs(r.PValue-expP))
			}
			if r.DF == nil || int(*r.DF) != expDF {
				t.Errorf("df: got %v, want %d", r.DF, expDF)
			}
			// Observed and Expected: nCats rows x 1 column each
			if r.Observed == nil {
				t.Fatal("Observed is nil")
			}
			if r.Expected == nil {
				t.Fatal("Expected is nil")
			}
			rows, cols := r.Observed.Size()
			if rows != c.nCats || cols != 1 {
				t.Errorf("Observed size: got (%d,%d), want (%d,1)", rows, cols, c.nCats)
			}
			rows, cols = r.Expected.Size()
			if rows != c.nCats || cols != 1 {
				t.Errorf("Expected size: got (%d,%d), want (%d,1)", rows, cols, c.nCats)
			}
			for i := range c.nCats {
				obsVal, ok := r.Observed.GetElementByNumberIndex(i, 0).(float64)
				if !ok {
					t.Fatalf("Observed[%d,0] not float64: %T", i, r.Observed.GetElementByNumberIndex(i, 0))
				}
				expVal, ok := r.Expected.GetElementByNumberIndex(i, 0).(float64)
				if !ok {
					t.Fatalf("Expected[%d,0] not float64: %T", i, r.Expected.GetElementByNumberIndex(i, 0))
				}
				expObs := chiRef.get(t, c.prefix+".obs["+itoa(i)+"]")
				expExp := chiRef.get(t, c.prefix+".exp["+itoa(i)+"]")
				if !chiClose(obsVal, expObs, tolChi) {
					t.Errorf("observed[%d]: got %v, want %v", i, obsVal, expObs)
				}
				if !chiClose(expVal, expExp, tolChi) {
					t.Errorf("expected[%d]: got %v, want %v", i, expVal, expExp)
				}
			}
		})
	}
}

func TestChiSquareGoodnessOfFit_Errors(t *testing.T) {
	if _, err := stats.ChiSquareGoodnessOfFit(insyra.NewDataList(), nil, false); err == nil {
		t.Error("expected error for empty input")
	}
	d := asAnyStrings([]string{"A", "B", "A", "B"})
	if _, err := stats.ChiSquareGoodnessOfFit(d, map[string]float64{"A": 0.5, "B": 0.5, "C": 0.5}, false); err == nil {
		t.Error("expected error for unknown category key")
	}
	if _, err := stats.ChiSquareGoodnessOfFit(d, map[string]float64{"A": -0.1, "B": 1.1}, false); err == nil {
		t.Error("expected error for negative p")
	}
	if _, err := stats.ChiSquareGoodnessOfFit(d, map[string]float64{"A": 0.4, "B": 0.4}, false); err == nil {
		t.Error("expected error for p sum != 1 without rescale")
	}
	if _, err := stats.ChiSquareGoodnessOfFit(d, map[string]float64{"A": 0.4, "B": 0.4}, true); err != nil {
		t.Errorf("expected no error for p sum != 1 with rescale, got %v", err)
	}
}

// ============================================================
// Independence
// ============================================================

func TestChiSquareIndependenceTest_R(t *testing.T) {
	cases := []struct {
		name         string
		rows, cols   []string
		prefix       string
		nRows, nCols int
	}{
		{name: "existing_3x2",
			rows:   []string{"A", "A", "B", "B", "B", "C"},
			cols:   []string{"X", "Y", "X", "Y", "Y", "Y"},
			prefix: "ind_existing", nRows: 3, nCols: 2},
		{name: "2x2_strong",
			rows: append(repeatStr("M", 50), repeatStr("F", 50)...),
			cols: append(append(repeatStr("Y", 40), repeatStr("N", 10)...),
				append(repeatStr("Y", 12), repeatStr("N", 38)...)...),
			prefix: "ind_2x2_strong", nRows: 2, nCols: 2},
		{name: "3x3",
			rows: func() []string {
				return append(append(repeatStr("low", 30), repeatStr("mid", 30)...), repeatStr("high", 30)...)
			}(),
			cols: func() []string {
				lowBlock := append(append(repeatStr("a", 15), repeatStr("b", 10)...), repeatStr("c", 5)...)
				midBlock := append(append(repeatStr("a", 10), repeatStr("b", 12)...), repeatStr("c", 8)...)
				highBlock := append(append(repeatStr("a", 5), repeatStr("b", 8)...), repeatStr("c", 17)...)
				return append(append(lowBlock, midBlock...), highBlock...)
			}(),
			prefix: "ind_3x3", nRows: 3, nCols: 3},
		{name: "4x3_largeN",
			rows:   chiDump.get(t, "ind_4x3_largeN_rows"),
			cols:   chiDump.get(t, "ind_4x3_largeN_cols"),
			prefix: "ind_4x3_largeN", nRows: 4, nCols: 3},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := stats.ChiSquareIndependenceTest(
				asAnyStrings(c.rows), asAnyStrings(c.cols))
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			expChi := chiRef.get(t, c.prefix+".chi")
			expP := chiRef.get(t, c.prefix+".p")
			expDF := int(chiRef.get(t, c.prefix+".df"))

			if !chiClose(r.Statistic, expChi, tolChi) {
				t.Errorf("chi: got %.17g, want %.17g", r.Statistic, expChi)
			}
			if !chiClose(r.PValue, expP, tolChiP) {
				t.Errorf("p: got %.17g, want %.17g", r.PValue, expP)
			}
			if r.DF == nil || int(*r.DF) != expDF {
				t.Errorf("df: got %v, want %d", r.DF, expDF)
			}
			if r.Observed == nil {
				t.Fatal("Observed is nil")
			}
			if r.Expected == nil {
				t.Fatal("Expected is nil")
			}
			rows, cols := r.Observed.Size()
			if rows != c.nRows || cols != c.nCols {
				t.Errorf("Observed size: got (%d,%d), want (%d,%d)",
					rows, cols, c.nRows, c.nCols)
			}
			rows, cols = r.Expected.Size()
			if rows != c.nRows || cols != c.nCols {
				t.Errorf("Expected size: got (%d,%d), want (%d,%d)",
					rows, cols, c.nRows, c.nCols)
			}
			// Observed and Expected are column-major: col j contains all rows for category j
			for j := range c.nCols {
				for i := range c.nRows {
					obsVal, ok := r.Observed.GetElementByNumberIndex(i, j).(float64)
					if !ok {
						t.Fatalf("Observed(%d,%d) not float64: %T", i, j, r.Observed.GetElementByNumberIndex(i, j))
					}
					expVal, ok := r.Expected.GetElementByNumberIndex(i, j).(float64)
					if !ok {
						t.Fatalf("Expected(%d,%d) not float64: %T", i, j, r.Expected.GetElementByNumberIndex(i, j))
					}
					idx := i*c.nCols + j
					expObs := chiRef.get(t, c.prefix+".obs["+itoa(idx)+"]")
					expExp := chiRef.get(t, c.prefix+".exp["+itoa(idx)+"]")
					if !chiClose(obsVal, expObs, tolChi) {
						t.Errorf("obs(%d,%d): got %v, want %v", i, j, obsVal, expObs)
					}
					if !chiClose(expVal, expExp, tolChi) {
						t.Errorf("exp(%d,%d): got %v, want %v", i, j, expVal, expExp)
					}
				}
			}
		})
	}
}

func TestChiSquareIndependenceTest_Errors(t *testing.T) {
	d := asAnyStrings([]string{"a", "b"})
	if _, err := stats.ChiSquareIndependenceTest(insyra.NewDataList(), d); err == nil {
		t.Error("expected error for empty input")
	}
	if _, err := stats.ChiSquareIndependenceTest(d, asAnyStrings([]string{"x"})); err == nil {
		t.Error("expected error for length mismatch")
	}
	// Single category in row → fewer than 2 row categories
	allSame := asAnyStrings([]string{"A", "A", "A", "A"})
	other := asAnyStrings([]string{"X", "Y", "X", "Y"})
	if _, err := stats.ChiSquareIndependenceTest(allSame, other); err == nil {
		t.Error("expected error for single-category row data")
	}
}

func repeatStr(s string, n int) []string {
	out := make([]string, n)
	for i := range n {
		out[i] = s
	}
	return out
}

// TestChiSquareIndependenceMatchesChisqTest verifies independence test results
// against the chisq_test_reference.txt file for multiple table sizes.
func TestChiSquareIndependenceMatchesChisqTest(t *testing.T) {
	chisqTestRef := &refTable{path: "testdata/chisq_test_reference.txt"}
	prefixes := []string{"ind_3x2", "ind_2x2", "ind_3x3"}
	for _, prefix := range prefixes {
		t.Run(prefix, func(t *testing.T) {
			rowsStr := chisqTestRef.getString(t, prefix+".rows")
			colsStr := chisqTestRef.getString(t, prefix+".cols")
			rawRows := strings.Split(rowsStr, ",")
			rawCols := strings.Split(colsStr, ",")

			r, err := stats.ChiSquareIndependenceTest(asAnyStrings(rawRows), asAnyStrings(rawCols))
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			expStat := chisqTestRef.get(t, prefix+".stat")
			expP := chisqTestRef.get(t, prefix+".p")
			expDF := int(chisqTestRef.get(t, prefix+".df"))

			if !chiClose(r.Statistic, expStat, tolChi) {
				t.Errorf("stat: got %.17g, want %.17g", r.Statistic, expStat)
			}
			if !chiClose(r.PValue, expP, tolChiP) {
				t.Errorf("p: got %.17g, want %.17g", r.PValue, expP)
			}
			if r.DF == nil || int(*r.DF) != expDF {
				t.Errorf("df: got %v, want %d", r.DF, expDF)
			}

			// Get unique sorted categories from the result tables (which match reference)
			expRowLabels := r.Observed.RowNames()
			expColLabels := r.Observed.ColNames()
			nRows := len(expRowLabels)
			nCols := len(expColLabels)

			// Check row/column labels against reference
			refRowLabels := make([]string, nRows)
			refColLabels := make([]string, nCols)
			for i := range nRows {
				refRowLabels[i] = chisqTestRef.getString(t, fmt.Sprintf("%s.rowlabel[%d]", prefix, i))
			}
			for j := range nCols {
				refColLabels[j] = chisqTestRef.getString(t, fmt.Sprintf("%s.collabel[%d]", prefix, j))
			}
			if !sliceEqual(expRowLabels, refRowLabels) {
				t.Errorf("Observed row labels: got %v, want %v", expRowLabels, refRowLabels)
			}
			if !sliceEqual(expColLabels, refColLabels) {
				t.Errorf("Observed col labels: got %v, want %v", expColLabels, refColLabels)
			}
			if !sliceEqual(r.Expected.RowNames(), refRowLabels) {
				t.Errorf("Expected row labels: got %v, want %v", r.Expected.RowNames(), refRowLabels)
			}
			if !sliceEqual(r.Expected.ColNames(), refColLabels) {
				t.Errorf("Expected col labels: got %v, want %v", r.Expected.ColNames(), refColLabels)
			}

			// Check every cell
			for i := range nRows {
				for j := range nCols {
					obsKey := fmt.Sprintf("%s.obs[%d][%d]", prefix, i, j)
					expKey := fmt.Sprintf("%s.exp[%d][%d]", prefix, i, j)
					expObs := chisqTestRef.get(t, obsKey)
					expExp := chisqTestRef.get(t, expKey)
					gotObs, ok := r.Observed.GetElementByNumberIndex(i, j).(float64)
					if !ok {
						t.Fatalf("Observed[%d][%d] not float64", i, j)
					}
					gotExp, ok := r.Expected.GetElementByNumberIndex(i, j).(float64)
					if !ok {
						t.Fatalf("Expected[%d][%d] not float64", i, j)
					}
					if !chiClose(gotObs, expObs, tolChi) {
						t.Errorf("Observed[%d][%d]: got %v, want %v", i, j, gotObs, expObs)
					}
					if !chiClose(gotExp, expExp, tolChi) {
						t.Errorf("Expected[%d][%d]: got %v, want %v", i, j, gotExp, expExp)
					}
				}
			}
		})
	}
}

// sliceEqual compares two string slices for equality.
func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestChiSquareExpectedTableSums checks that expected column totals match observed column totals.
func TestChiSquareExpectedTableSums(t *testing.T) {
	rows := []string{"A", "A", "B", "B", "B", "C"}
	cols := []string{"X", "Y", "X", "Y", "Y", "Y"}

	r, err := stats.ChiSquareIndependenceTest(asAnyStrings(rows), asAnyStrings(cols))
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if r.Expected.Err() != nil {
		t.Fatalf("Expected table has error: %v", r.Expected.Err())
	}

	_, nCols := r.Expected.Size()
	for j := range nCols {
		obsSum := r.Observed.GetColByNumber(j).Sum()
		expSum := r.Expected.GetColByNumber(j).Sum()
		if math.IsNaN(expSum) {
			t.Errorf("Expected column %d sum is NaN", j)
		}
		if !chiClose(expSum, obsSum, tolChi) {
			t.Errorf("Expected column %d sum: got %v, want %v (observed sum)", j, expSum, obsSum)
		}
	}
}

var chisqTestRef = &refTable{path: "testdata/chisq_test_reference.txt"}

// TestChiSquareGoodnessOfFitMatchesChisqTest verifies GOF results against
// chisq_test_reference.txt for multiple cases.
func TestChiSquareGoodnessOfFitMatchesChisqTest(t *testing.T) {
	cases := []struct {
		prefix string
		nP     int // number of p keys; 0 means nil p (uniform)
		nCats  int // expected number of categories (rows)
	}{
		{prefix: "gof_uniform", nP: 0, nCats: 4},
		{prefix: "gof_named", nP: 3, nCats: 3},
		{prefix: "gof_rescale", nP: 4, nCats: 4},
		{prefix: "gof_many", nP: 0, nCats: 8},
	}

	for _, c := range cases {
		t.Run(c.prefix, func(t *testing.T) {
			dataStr := chisqTestRef.getString(t, c.prefix+".data")
			data := strings.Split(dataStr, ",")
			rescaleStr := chisqTestRef.getString(t, c.prefix+".rescale")
			rescale := rescaleStr == "true"

			var pMap map[string]float64
			if c.nP > 0 {
				pMap = make(map[string]float64, c.nP)
				for i := 0; i < c.nP; i++ {
					key := chisqTestRef.getString(t, fmt.Sprintf("%s.pkey[%d]", c.prefix, i))
					val := chisqTestRef.get(t, fmt.Sprintf("%s.pval[%d]", c.prefix, i))
					pMap[key] = val
				}
			}

			r, err := stats.ChiSquareGoodnessOfFit(asAnyStrings(data), pMap, rescale)
			if err != nil {
				t.Fatalf("ChiSquareGoodnessOfFit error: %v", err)
			}

			expStat := chisqTestRef.get(t, c.prefix+".stat")
			expP := chisqTestRef.get(t, c.prefix+".p")
			expDF := int(chisqTestRef.get(t, c.prefix+".df"))

			if !chiClose(r.Statistic, expStat, tolChi) {
				t.Errorf("stat: got %.17g, want %.17g", r.Statistic, expStat)
			}
			if !chiClose(r.PValue, expP, tolChiP) {
				t.Errorf("p: got %.17g, want %.17g", r.PValue, expP)
			}
			if r.DF == nil || int(*r.DF) != expDF {
				t.Errorf("df: got %v, want %d", r.DF, expDF)
			}

			if !sliceEqual(r.Observed.ColNames(), []string{"Observed"}) {
				t.Errorf("Observed column names: got %v, want [Observed]", r.Observed.ColNames())
			}
			if !sliceEqual(r.Expected.ColNames(), []string{"Expected"}) {
				t.Errorf("Expected column names: got %v, want [Expected]", r.Expected.ColNames())
			}
			if !sliceEqual(r.Expected.RowNames(), r.Observed.RowNames()) {
				t.Errorf("Expected row names mismatch Observed: got %v, want %v", r.Expected.RowNames(), r.Observed.RowNames())
			}
			if r.Observed.NumRows() != c.nCats {
				t.Errorf("Observed row count: got %d, want %d", r.Observed.NumRows(), c.nCats)
			}

			nRows := r.Observed.NumRows()
			for i := 0; i < nRows; i++ {
				expLabel := chisqTestRef.getString(t, fmt.Sprintf("%s.label[%d]", c.prefix, i))
				if r.Observed.RowNames()[i] != expLabel {
					t.Errorf("row %d label: got %q, want %q", i, r.Observed.RowNames()[i], expLabel)
				}
				obsVal, ok := r.Observed.GetElementByNumberIndex(i, 0).(float64)
				if !ok {
					t.Fatalf("Observed[%d,0] not float64: %T", i, r.Observed.GetElementByNumberIndex(i, 0))
				}
				expVal, ok := r.Expected.GetElementByNumberIndex(i, 0).(float64)
				if !ok {
					t.Fatalf("Expected[%d,0] not float64: %T", i, r.Expected.GetElementByNumberIndex(i, 0))
				}
				expObs := chisqTestRef.get(t, fmt.Sprintf("%s.obs[%d]", c.prefix, i))
				expExp := chisqTestRef.get(t, fmt.Sprintf("%s.exp[%d]", c.prefix, i))
				if !chiClose(obsVal, expObs, tolChi) {
					t.Errorf("observed[%d]: got %v, want %v", i, obsVal, expObs)
				}
				if !chiClose(expVal, expExp, tolChi) {
					t.Errorf("expected[%d]: got %v, want %v", i, expVal, expExp)
				}
			}
		})
	}
}

// TestChiSquareGoodnessOfFitMatchesByName verifies that the map-based API
// produces the same result regardless of input order.
func TestChiSquareGoodnessOfFitMatchesByName(t *testing.T) {
	// Input in one order
	data1 := []string{"green", "red", "blue", "red", "red", "green", "blue", "red", "blue", "red"}
	p := map[string]float64{"red": 0.5, "green": 0.3, "blue": 0.2}

	r1, err := stats.ChiSquareGoodnessOfFit(asAnyStrings(data1), p, false)
	if err != nil {
		t.Fatalf("first run error: %v", err)
	}

	// Check row names are sorted
	if !sliceEqual(r1.Observed.RowNames(), []string{"blue", "green", "red"}) {
		t.Errorf("row names: got %v, want [blue green red]", r1.Observed.RowNames())
	}
	// Check observed counts in sorted order: blue=3, green=2, red=5
	obs1 := make([]float64, 3)
	exp1 := make([]float64, 3)
	for i := range 3 {
		obs1[i], _ = r1.Observed.GetElementByNumberIndex(i, 0).(float64)
		exp1[i], _ = r1.Expected.GetElementByNumberIndex(i, 0).(float64)
	}
	if !sliceEqualFloat(obs1, []float64{3, 2, 5}) {
		t.Errorf("observed: got %v, want [3 2 5]", obs1)
	}
	if !sliceEqualFloat(exp1, []float64{2, 3, 5}) {
		t.Errorf("expected: got %v, want [2 3 5]", exp1)
	}
	// Statistic = (3-2)^2/2 + (2-3)^2/3 + (5-5)^2/5 = 1/2 + 1/3 = 5/6
	wantStat := 1.0/2 + 1.0/3
	if !chiClose(r1.Statistic, wantStat, tolChi) {
		t.Errorf("stat: got %.17g, want %.17g", r1.Statistic, wantStat)
	}
	if r1.DF == nil || int(*r1.DF) != 2 {
		t.Errorf("df: got %v, want 2", r1.DF)
	}

	// Same values in reverse order - should produce identical results
	data2 := []string{"red", "blue", "red", "green", "blue", "red", "red", "blue", "green", "red"}
	r2, err := stats.ChiSquareGoodnessOfFit(asAnyStrings(data2), p, false)
	if err != nil {
		t.Fatalf("second run error: %v", err)
	}

	if r2.Statistic != r1.Statistic {
		t.Errorf("stat mismatch: first=%v, second=%v", r1.Statistic, r2.Statistic)
	}
	if r2.PValue != r1.PValue {
		t.Errorf("p mismatch: first=%v, second=%v", r1.PValue, r2.PValue)
	}
	if r2.DF == nil || r1.DF == nil || *r2.DF != *r1.DF {
		t.Errorf("df mismatch: first=%v, second=%v", r1.DF, r2.DF)
	}
}

// TestChiSquareGoodnessOfFitProbabilityKeys tests the error cases for
// probability map keys.
func TestChiSquareGoodnessOfFitProbabilityKeys(t *testing.T) {
	data := []string{"green", "red", "blue", "red", "red", "green", "blue", "red", "blue", "red"}

	// Case 1: key that doesn't match any category (case-sensitive)
	p1 := map[string]float64{"red": 0.5, "green": 0.3, "Blue": 0.2}
	r, err := stats.ChiSquareGoodnessOfFit(asAnyStrings(data), p1, false)
	if r != nil {
		t.Errorf("expected nil result for unknown category key 'Blue', got %v", r)
	}
	if err == nil {
		t.Error("expected error for unknown category key 'Blue'")
	} else if err.Error() != `p names category "Blue", which does not occur in input` {
		t.Errorf("wrong error: got %q, want %q", err.Error(), `p names category "Blue", which does not occur in input`)
	}

	// Case 2: missing key for a category that exists
	p2 := map[string]float64{"red": 0.5, "green": 0.5}
	r, err = stats.ChiSquareGoodnessOfFit(asAnyStrings(data), p2, false)
	if r != nil {
		t.Errorf("expected nil result for missing category 'blue', got %v", r)
	}
	if err == nil {
		t.Error("expected error for missing category 'blue'")
	} else if err.Error() != `p has no probability for category "blue"` {
		t.Errorf("wrong error: got %q, want %q", err.Error(), `p has no probability for category "blue"`)
	}

	// Case 3: nil p and empty p map should give same result as uniform
	pUniform := map[string]float64{"red": 1.0 / 3, "green": 1.0 / 3, "blue": 1.0 / 3}
	rNil, err := stats.ChiSquareGoodnessOfFit(asAnyStrings(data), nil, false)
	if err != nil {
		t.Fatalf("nil p error: %v", err)
	}
	rEmpty, err := stats.ChiSquareGoodnessOfFit(asAnyStrings(data), map[string]float64{}, false)
	if err != nil {
		t.Fatalf("empty p error: %v", err)
	}
	rUniform, err := stats.ChiSquareGoodnessOfFit(asAnyStrings(data), pUniform, false)
	if err != nil {
		t.Fatalf("uniform p error: %v", err)
	}
	if rNil.Statistic != rEmpty.Statistic {
		t.Errorf("nil vs empty p stat mismatch: %v != %v", rNil.Statistic, rEmpty.Statistic)
	}
	if rNil.Statistic != rUniform.Statistic {
		t.Errorf("nil vs uniform p stat mismatch: %v != %v", rNil.Statistic, rUniform.Statistic)
	}
	if rNil.PValue != rEmpty.PValue {
		t.Errorf("nil vs empty p p-value mismatch: %v != %v", rNil.PValue, rEmpty.PValue)
	}
	if rNil.PValue != rUniform.PValue {
		t.Errorf("nil vs uniform p p-value mismatch: %v != %v", rNil.PValue, rUniform.PValue)
	}

	// Case 4: rescaleP true should succeed with non-summing-to-1 values
	// and the original map should not be modified
	wantStat := 1.0/2 + 1.0/3
	p4 := map[string]float64{"red": 5, "green": 3, "blue": 2}
	r4, err := stats.ChiSquareGoodnessOfFit(asAnyStrings(data), p4, true)
	if err != nil {
		t.Fatalf("rescale error: %v", err)
	}
	if p4["red"] != 5 || p4["green"] != 3 || p4["blue"] != 2 {
		t.Errorf("original p map was modified: got %v", p4)
	}
	// Verify the result is correct (5+3+2=10, so probs are 0.5, 0.3, 0.2)
	if !chiClose(r4.Statistic, wantStat, tolChi) {
		t.Errorf("rescale stat: got %.17g, want %.17g", r4.Statistic, wantStat)
	}
}

// sliceEqualFloat compares two float64 slices for equality.
func sliceEqualFloat(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestChiSquareGoodnessOfFitNumericLabels tests that numeric labels use
// fmt.Sprint formatting (so 1.0 becomes "1", not "1.0").
func TestChiSquareGoodnessOfFitNumericLabels(t *testing.T) {
	// Input with float64 values that fmt.Sprint renders as "1" and "2"
	dl := insyra.NewDataList(1.0, 2.0, 1.0, 1.0)

	// p with labels matching fmt.Sprint output ("1", "2") should succeed
	p := map[string]float64{"1": 0.5, "2": 0.5}
	r, err := stats.ChiSquareGoodnessOfFit(dl, p, true)
	if err != nil {
		t.Fatalf("expected success with fmt.Sprint labels, got error: %v", err)
	}
	if !sliceEqual(r.Observed.RowNames(), []string{"1", "2"}) {
		t.Errorf("row names: got %v, want [1 2]", r.Observed.RowNames())
	}

	// p with labels "1.0", "2.0" should fail because those are not the fmt.Sprint labels
	p2 := map[string]float64{"1.0": 0.5, "2.0": 0.5}
	r, err = stats.ChiSquareGoodnessOfFit(dl, p2, true)
	if r != nil {
		t.Errorf("expected nil result for non-fmt.Sprint labels, got %v", r)
	}
	if err == nil {
		t.Error("expected error for label '1.0' not matching input")
	} else if err.Error() != `p names category "1.0", which does not occur in input` {
		t.Errorf("wrong error: got %q, want %q", err.Error(), `p names category "1.0", which does not occur in input`)
	}
}
