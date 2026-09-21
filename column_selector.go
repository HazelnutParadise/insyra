package insyra

import (
	"fmt"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/utils"
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

// lookupColIn is the one rule, applied to any list of columns: a string is an
// Excel-style index, a Name is a name compared exactly, an int is a position
// counting from the end when negative. A grouped snapshot resolves through the
// same function as a live table, so a selector cannot mean two things.
func lookupColIn(columns []*DataList, selector any) (num int, warning, problem string) {
	switch v := selector.(type) {
	case nil:
		return -1, "", "no column given, use an Excel-style index such as \"A\", Name(\"price\"), or an int position"
	case string:
		return lookupColIndexStringIn(columns, v)
	case NameSelector:
		position, ok := columnNumberByName(columns, v.value)
		if !ok {
			return -1, "", fmt.Sprintf("no column is named %s", v.value)
		}
		return position, "", ""
	case int:
		position := v
		if position < 0 {
			position = len(columns) + position
		}
		if position < 0 || position >= len(columns) {
			return -1, "", fmt.Sprintf("column number %d is out of range, the table has %d column(s)", v, len(columns))
		}
		return position, "", ""
	default:
		return -1, "", fmt.Sprintf("column selector of type %T, use an Excel-style index such as \"A\", Name(\"price\"), or an int position", selector)
	}
}

// lookupColIndexStringIn resolves a bare string, which is always an Excel-style
// index. The column names are read only to shape what a failure says, never to
// decide which column is used, so renaming a column cannot change what a call
// means.
func lookupColIndexStringIn(columns []*DataList, index string) (num int, warning, problem string) {
	position, ok := utils.ParseColIndex(index)
	if !ok {
		if _, named := columnNumberByName(columns, index); named {
			return -1, "", fmt.Sprintf("%q is not an Excel-style column index, and this table has a column named %s, so write Name(%q)", index, index, index)
		}
		return -1, "", fmt.Sprintf("%q is not an Excel-style column index, and no column is named %s, write Name(%q) for a column name", index, index, index)
	}
	if position >= len(columns) {
		letters, _ := utils.CalcColIndex(position)
		if _, named := columnNumberByName(columns, index); named {
			return -1, "", fmt.Sprintf("%q reads as column index %s, past the last of %d column(s), and this table has a column named %s, so write Name(%q)", index, letters, len(columns), index, index)
		}
		return -1, "", fmt.Sprintf("column %s does not exist, the table has %d column(s)", letters, len(columns))
	}
	if named, ok := columnNumberByName(columns, index); ok && named != position {
		letters, _ := utils.CalcColIndex(position)
		warning = fmt.Sprintf("%q was read as column index %s (column %d), while column %d is named %s; write Name(%q) for the named one", index, letters, position, named, index, index)
	}
	return position, warning, ""
}

// columnNumberByName finds a column by exact name.
func columnNumberByName(columns []*DataList, name string) (int, bool) {
	for i, column := range columns {
		if column.name == name {
			return i, true
		}
	}
	return -1, false
}

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
func isEmptySelector(selector any) bool {
	switch v := selector.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	case NameSelector:
		return strings.TrimSpace(v.value) == ""
	default:
		return false
	}
}
