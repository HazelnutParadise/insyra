package insyra

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"
)

// Integers match by value across Go integer types. Every other searched value
// must match exactly as it did before that change. The two oracles below are
// the comparisons the methods applied then, and the matrix runs every value
// lookup, count, replace and drop over floats, NaN, strings, bools, nil and
// named types that are not one of Go's integer types.

// legacyMatch is the comparison FindFirst, FindLast, FindAll, Count, the
// Replace methods, DropAll, FindRowsIfContains, FindColsIfContains(All),
// DropRowsContain and DropColsContain applied: a float64 NaN matched a float64
// NaN, and anything else compared with ==.
func legacyMatch(cell, want any) bool {
	if w, ok := want.(float64); ok && math.IsNaN(w) {
		c, ok := cell.(float64)
		return ok && math.IsNaN(c)
	}
	return cell == want
}

// legacyMatchNoNaN is FindRowsIfContainsAll's comparison: a bare ==, under
// which NaN never matches.
func legacyMatchNoNaN(cell, want any) bool { return cell == want }

// parityInt is a named integer type. It is not one of Go's integer types, so
// it keeps matching only its own type.
type parityInt int

func parityCells() []any {
	return []any{
		2.0, math.NaN(), float32(2), float32(math.NaN()), "2", "", true, false, nil,
		0.0, math.Copysign(0, -1), time.Duration(2), parityInt(2), 2, int64(2), uint8(2),
	}
}

func parityWants() []any {
	return []any{
		2.0, math.NaN(), float32(2), float32(math.NaN()), "2", "", true, false, nil,
		0.0, math.Copysign(0, -1), time.Duration(2), parityInt(2),
	}
}

func renderCells(data []any) string {
	var b strings.Builder
	for _, v := range data {
		fmt.Fprintf(&b, "%T(%v) ", v, v)
	}
	return b.String()
}

const paritySentinel = "REPLACED"

func matchingIndices(cells []any, want any, match func(cell, want any) bool) []int {
	var idx []int
	for i, c := range cells {
		if match(c, want) {
			idx = append(idx, i)
		}
	}
	return idx
}

func replacedAt(cells []any, idx ...int) []any {
	out := slices.Clone(cells)
	for _, i := range idx {
		out[i] = paritySentinel
	}
	return out
}

func TestDataListValueLookupsKeepNonIntegerMatching(t *testing.T) {
	quietLogs(t)
	cells := parityCells()
	if got := renderCells(NewDataList(cells...).Data()); got != renderCells(cells) {
		t.Fatalf("NewDataList changed the cells: %s", got)
	}
	for _, w := range parityWants() {
		name := fmt.Sprintf("%T(%v)", w, w)
		idx := matchingIndices(cells, w, legacyMatch)

		if got := NewDataList(cells...).FindAll(w); !slices.Equal(got, idx) {
			t.Errorf("%s: FindAll = %v, want %v", name, got, idx)
		}
		if got := NewDataList(cells...).Count(w); got != len(idx) {
			t.Errorf("%s: Count = %d, want %d", name, got, len(idx))
		}
		var first, last any
		if len(idx) > 0 {
			first, last = idx[0], idx[len(idx)-1]
		}
		if got := NewDataList(cells...).FindFirst(w); got != first {
			t.Errorf("%s: FindFirst = %v, want %v", name, got, first)
		}
		if got := NewDataList(cells...).FindLast(w); got != last {
			t.Errorf("%s: FindLast = %v, want %v", name, got, last)
		}

		wantAll := replacedAt(cells, idx...)
		wantFirst, wantLast := slices.Clone(cells), slices.Clone(cells)
		if len(idx) > 0 {
			wantFirst = replacedAt(cells, idx[0])
			wantLast = replacedAt(cells, idx[len(idx)-1])
		}
		for _, c := range []struct {
			method string
			got    []any
			want   []any
		}{
			{"ReplaceAll", NewDataList(cells...).ReplaceAll(w, paritySentinel).Data(), wantAll},
			{"ReplaceFirst", NewDataList(cells...).ReplaceFirst(w, paritySentinel).Data(), wantFirst},
			{"ReplaceLast", NewDataList(cells...).ReplaceLast(w, paritySentinel).Data(), wantLast},
		} {
			if renderCells(c.got) != renderCells(c.want) {
				t.Errorf("%s: %s left %s, want %s", name, c.method, renderCells(c.got), renderCells(c.want))
			}
		}

		var kept []any
		for i, c := range cells {
			if !slices.Contains(idx, i) {
				kept = append(kept, c)
			}
		}
		if got := NewDataList(cells...).DropAll(w).Data(); renderCells(got) != renderCells(kept) {
			t.Errorf("%s: DropAll left %s, want %s", name, renderCells(got), renderCells(kept))
		}
	}
}

func TestDataTableValueLookupsKeepNonIntegerMatching(t *testing.T) {
	quietLogs(t)
	colA := parityCells()
	colB := slices.Clone(colA)
	slices.Reverse(colB)
	cols := [][]any{colA, colB}
	n := len(colA)
	mk := func() *DataTable {
		return NewDataTable(NewDataList(colA...).SetName("a"), NewDataList(colB...).SetName("b"))
	}
	tableData := func(dt *DataTable) string {
		var b strings.Builder
		for i := 0; i < dt.NumCols(); i++ {
			b.WriteString(renderCells(dt.GetColByNumber(i).Data()))
			b.WriteString("| ")
		}
		return b.String()
	}
	render := func(columns ...[]any) string {
		var b strings.Builder
		for _, c := range columns {
			b.WriteString(renderCells(c))
			b.WriteString("| ")
		}
		return b.String()
	}

	for _, w := range parityWants() {
		name := fmt.Sprintf("%T(%v)", w, w)
		idxA := matchingIndices(colA, w, legacyMatch)
		idxB := matchingIndices(colB, w, legacyMatch)

		if got := mk().Count(w); got != len(idxA)+len(idxB) {
			t.Errorf("%s: Count = %d, want %d", name, got, len(idxA)+len(idxB))
		}

		var rows []int
		for r := range n {
			if legacyMatch(colA[r], w) || legacyMatch(colB[r], w) {
				rows = append(rows, r)
			}
		}
		if got := mk().FindRowsIfContains(w); !slices.Equal(got, rows) {
			t.Errorf("%s: FindRowsIfContains = %v, want %v", name, got, rows)
		}

		for _, values := range [][]any{{w}, {w, "2"}} {
			var all []int
			for r := range n {
				foundAll := true
				for _, v := range values {
					if !legacyMatchNoNaN(colA[r], v) && !legacyMatchNoNaN(colB[r], v) {
						foundAll = false
						break
					}
				}
				if foundAll {
					all = append(all, r)
				}
			}
			if got := mk().FindRowsIfContainsAll(values...); !slices.Equal(got, all) {
				t.Errorf("%s: FindRowsIfContainsAll%v = %v, want %v", name, values, got, all)
			}
		}

		var colsWith []string
		if len(idxA) > 0 {
			colsWith = append(colsWith, "A")
		}
		if len(idxB) > 0 {
			colsWith = append(colsWith, "B")
		}
		if got := mk().FindColsIfContains(w); !slices.Equal(got, colsWith) {
			t.Errorf("%s: FindColsIfContains = %v, want %v", name, got, colsWith)
		}
		var colsWithAll []string
		for i, c := range cols {
			if len(matchingIndices(c, w, legacyMatch)) > 0 && len(matchingIndices(c, "", legacyMatch)) > 0 {
				colsWithAll = append(colsWithAll, string(rune('A'+i)))
			}
		}
		if got := mk().FindColsIfContainsAll(w, ""); !slices.Equal(got, colsWithAll) {
			t.Errorf("%s: FindColsIfContainsAll = %v, want %v", name, got, colsWithAll)
		}

		if got, want := tableData(mk().Replace(w, paritySentinel)), render(replacedAt(colA, idxA...), replacedAt(colB, idxB...)); got != want {
			t.Errorf("%s: Replace left %s, want %s", name, got, want)
		}

		pick := func(idx []int, mode int) []int {
			switch {
			case len(idx) == 0 || mode == 0:
				return idx
			case mode == 1:
				return idx[:1]
			default:
				return idx[len(idx)-1:]
			}
		}
		for _, mode := range []int{1, 0, -1} {
			got := tableData(mk().ReplaceInCol("A", w, paritySentinel, mode))
			want := render(replacedAt(colA, pick(idxA, mode)...), colB)
			if got != want {
				t.Errorf("%s: ReplaceInCol(A, mode %d) left %s, want %s", name, mode, got, want)
			}
			for r := range n {
				a, b := slices.Clone(colA), slices.Clone(colB)
				matchA, matchB := legacyMatch(colA[r], w), legacyMatch(colB[r], w)
				switch mode {
				case 0:
					if matchA {
						a[r] = paritySentinel
					}
					if matchB {
						b[r] = paritySentinel
					}
				case 1:
					if matchA {
						a[r] = paritySentinel
					} else if matchB {
						b[r] = paritySentinel
					}
				case -1:
					if matchB {
						b[r] = paritySentinel
					} else if matchA {
						a[r] = paritySentinel
					}
				}
				if got, want := tableData(mk().ReplaceInRow(r, w, paritySentinel, mode)), render(a, b); got != want {
					t.Errorf("%s: ReplaceInRow(%d, mode %d) left %s, want %s", name, r, mode, got, want)
				}
			}
		}

		var keptA, keptB []any
		for r := range n {
			if !slices.Contains(rows, r) {
				keptA = append(keptA, colA[r])
				keptB = append(keptB, colB[r])
			}
		}
		if got, want := tableData(mk().DropRowsContain(w)), render(keptA, keptB); got != want {
			t.Errorf("%s: DropRowsContain left %s, want %s", name, got, want)
		}
		var keptCols [][]any
		if len(idxA) == 0 {
			keptCols = append(keptCols, colA)
		}
		if len(idxB) == 0 {
			keptCols = append(keptCols, colB)
		}
		if got, want := tableData(mk().DropColsContain(w)), render(keptCols...); got != want {
			t.Errorf("%s: DropColsContain left %s, want %s", name, got, want)
		}
	}
}

// lookupUncomparableCell holds a slice, so Go panics when two of them are compared
// with ==. A lookup must treat such a cell as unequal rather than crash.
type lookupUncomparableCell struct{ s []int }

func TestValueLookupsDoNotPanicOnUncomparableCells(t *testing.T) {
	quietLogs(t)
	cell := func() any { return lookupUncomparableCell{[]int{1}} }
	list := func() *DataList { return NewDataList(1, cell(), "x") }
	table := func() *DataTable {
		return NewDataTable(NewDataList(1, cell()).SetName("a"), NewDataList(cell(), "x").SetName("b"))
	}
	for name, f := range map[string]func(){
		"FindFirst":             func() { list().FindFirst(cell()) },
		"FindLast":              func() { list().FindLast(cell()) },
		"FindAll":               func() { list().FindAll(cell()) },
		"Count":                 func() { list().Count(cell()) },
		"ReplaceAll":            func() { list().ReplaceAll(cell(), 0) },
		"ReplaceFirst":          func() { list().ReplaceFirst(cell(), 0) },
		"ReplaceLast":           func() { list().ReplaceLast(cell(), 0) },
		"DropAll":               func() { list().DropAll(cell()) },
		"DT.Count":              func() { table().Count(cell()) },
		"FindRowsIfContains":    func() { table().FindRowsIfContains(cell()) },
		"FindRowsIfContainsAll": func() { table().FindRowsIfContainsAll(cell()) },
		"FindColsIfContains":    func() { table().FindColsIfContains(cell()) },
		"Replace":               func() { table().Replace(cell(), 0) },
		"ReplaceInRow":          func() { table().ReplaceInRow(1, cell(), 0) },
		"ReplaceInCol":          func() { table().ReplaceInCol("B", cell(), 0) },
		"DropRowsContain":       func() { table().DropRowsContain(cell()) },
		"DropColsContain":       func() { table().DropColsContain(cell()) },
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s panicked on an uncomparable cell: %v", name, r)
				}
			}()
			f()
		}()
	}
}

// Searching for a value Go cannot compare over a column of other types costs
// about a type check per cell. The searched value used to be encoded again for
// every cell, so a 64 KB []byte over 20,000 int64 cells took seconds.
func TestSearchingForAnUncomparableValueDoesNotReencodeItPerCell(t *testing.T) {
	quietLogs(t)
	cells := make([]any, 20000)
	for i := range cells {
		cells[i] = int64(i)
	}
	blob := make([]byte, 64<<10)
	blob[0] = 1
	cells[7] = Cell(append([]byte(nil), blob...))
	dl := NewDataList(cells...)
	dt := NewDataTable(NewDataList(cells...))

	start := time.Now()
	found := dl.FindAll(blob)
	count := dl.Count(blob)
	dt.DropRowsContain(blob)
	elapsed := time.Since(start)
	t.Logf("FindAll + Count + DropRowsContain with a 64 KB []byte over 20,000 cells: %v", elapsed)

	if len(found) != 1 || found[0] != 7 || count != 1 {
		t.Fatalf("FindAll = %v, Count = %d; want [7] and 1", found, count)
	}
	if dt.NumRows() != 19999 {
		t.Fatalf("DropRowsContain left %d rows, want 19999", dt.NumRows())
	}
	if elapsed > time.Second {
		t.Fatalf("searching took %v; the searched value is being re-encoded per cell", elapsed)
	}
}
