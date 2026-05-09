package stats_test

import (
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
		name     string
		data     []string
		p        []float64
		rescale  bool
		prefix   string
		nCats    int
	}{
		{name: "uniform_no_p",
			data:   []string{"A", "A", "A", "B", "B", "C", "D", "D", "D", "D"},
			prefix: "gof_uniform", nCats: 4},
		{name: "custom_probs",
			data:   []string{"red", "red", "blue", "green", "blue", "red", "green", "red", "blue", "red"},
			p:      []float64{0.5, 0.3, 0.2}, // sorted: blue, green, red
			prefix: "gof_custom", nCats: 3},
		{name: "rescale_true",
			data:    []string{"A", "B", "B", "C", "C", "C", "D", "D", "D", "D"},
			p:       []float64{1, 2, 3, 4},
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
			// ContingencyTable: 1 column with [observed, expected] pairs
			if r.ContingencyTable == nil {
				t.Fatal("ContingencyTable is nil")
			}
			rows, cols := r.ContingencyTable.Size()
			if rows != c.nCats || cols != 1 {
				t.Errorf("ContingencyTable size: got (%d,%d), want (%d,1)", rows, cols, c.nCats)
			}
			col := r.ContingencyTable.GetColByNumber(0)
			for i := range c.nCats {
				pair, ok := col.Get(i).([2]float64)
				if !ok {
					t.Fatalf("ContingencyTable[%d] not [2]float64: %T", i, col.Get(i))
				}
				expObs := chiRef.get(t, c.prefix+".obs["+itoa(i)+"]")
				expExp := chiRef.get(t, c.prefix+".exp["+itoa(i)+"]")
				if !chiClose(pair[0], expObs, tolChi) {
					t.Errorf("observed[%d]: got %v, want %v", i, pair[0], expObs)
				}
				if !chiClose(pair[1], expExp, tolChi) {
					t.Errorf("expected[%d]: got %v, want %v", i, pair[1], expExp)
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
	if _, err := stats.ChiSquareGoodnessOfFit(d, []float64{0.5, 0.5, 0.5}, false); err == nil {
		t.Error("expected error for length mismatch")
	}
	if _, err := stats.ChiSquareGoodnessOfFit(d, []float64{-0.1, 1.1}, false); err == nil {
		t.Error("expected error for negative p")
	}
	if _, err := stats.ChiSquareGoodnessOfFit(d, []float64{0.4, 0.4}, false); err == nil {
		t.Error("expected error for p sum != 1 without rescale")
	}
	if _, err := stats.ChiSquareGoodnessOfFit(d, []float64{0.4, 0.4}, true); err != nil {
		t.Errorf("expected no error for p sum != 1 with rescale, got %v", err)
	}
}

// ============================================================
// Independence
// ============================================================

func TestChiSquareIndependenceTest_R(t *testing.T) {
	cases := []struct {
		name        string
		rows, cols  []string
		prefix      string
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
			if r.ContingencyTable == nil {
				t.Fatal("ContingencyTable is nil")
			}
			rows, cols := r.ContingencyTable.Size()
			if rows != c.nRows || cols != c.nCols {
				t.Errorf("ContingencyTable size: got (%d,%d), want (%d,%d)",
					rows, cols, c.nRows, c.nCols)
			}
			// Insyra's ContingencyTable is column-major: col j contains all
			// rows for category j. Reference R script uses the same ordering.
			for j := range c.nCols {
				col := r.ContingencyTable.GetColByNumber(j)
				for i := range c.nRows {
					pair, ok := col.Get(i).([2]float64)
					if !ok {
						t.Fatalf("cell(%d,%d) not [2]float64: %T", i, j, col.Get(i))
					}
					idx := i*c.nCols + j
					expObs := chiRef.get(t, c.prefix+".obs["+itoa(idx)+"]")
					expExp := chiRef.get(t, c.prefix+".exp["+itoa(idx)+"]")
					if !chiClose(pair[0], expObs, tolChi) {
						t.Errorf("obs(%d,%d): got %v, want %v", i, j, pair[0], expObs)
					}
					if !chiClose(pair[1], expExp, tolChi) {
						t.Errorf("exp(%d,%d): got %v, want %v", i, j, pair[1], expExp)
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
