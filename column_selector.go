package insyra

import (
	"github.com/HazelnutParadise/insyra/internal/prep"
)

// NameSelector marks a selector as a column or row name rather than an index.
// Build one with Name.
type NameSelector struct{ value string }

// Name says that a selector is a name rather than an index. Every selector in
// the library reads a bare string as an Excel-style index ("A", "B", ... "AA"),
// an int as a 0-based position, and a Name as a name compared exactly. The rule
// is the same everywhere, so a call means the same thing whatever the columns
// happen to be called.
func Name(value string) NameSelector { return NameSelector{value: value} }

// String returns the name itself, so a selector reads as what it selects.
func (n NameSelector) String() string { return n.value }

// Value returns the name a selector carries.
func (n NameSelector) Value() string { return n.value }

// selectorName reports the name a NameSelector carries, for the selector rule
// in internal/prep, which cannot name this type.
func selectorName(selector any) (string, bool) {
	n, ok := selector.(NameSelector)
	return n.value, ok
}

// lookupColSelector resolves a selector to a 0-based column position. It
// returns the problem to report when nothing resolved, and a warning when the
// caller's choice was sound but worth mentioning. Both are returned rather than
// recorded, because Err() is sticky and the public method the caller made must
// be the one that names the failure.
//
// The table must already be locked.
func (dt *DataTable) lookupColSelector(selector any) (num int, warning, problem string) {
	return lookupColIn(dt.columns, selector)
}

// lookupColIn is the one rule, applied to any list of columns, in
// prep.LookupColumn: a string is an Excel-style index, a Name is a name compared
// exactly, an int is a position counting from the end when negative. A grouped
// snapshot resolves through the same function as a live table, so a selector
// cannot mean two things.
func lookupColIn(columns []*DataList, selector any) (num int, warning, problem string) {
	return prep.LookupColumn(columns, columnListName, selectorName, selector)
}

func columnListName(dl *DataList) string { return dl.name }

// resolveColSelector is lookupColSelector with the reporting done, for the
// callers that have nothing else to add. The table must already be locked.
func (dt *DataTable) resolveColSelector(funcName string, selector any) (int, bool) {
	num, warning, problem := dt.lookupColSelector(selector)
	if problem != "" {
		dt.fail(funcName, "%s", problem)
		return -1, false
	}
	if warning != "" {
		dt.warn(funcName, "%s", warning)
	}
	return num, true
}

// isEmptySelector reports a selector that picks nothing: a nil, or the empty
// string a zero-valued config field carries.
func isEmptySelector(selector any) bool { return prep.IsEmptySelector(selector, selectorName) }
