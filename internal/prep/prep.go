// Package prep holds the pure helpers the preprocess package needs: the column
// selector rule, the type-7 quantile, the missing-value tests and the integer
// split. On 0.4 the insyra core calls them as well, so the core and preprocess
// cannot disagree about what a selector or a missing value means; on 0.3.x the
// core keeps its own copies, which read a bare string column differently.
//
// The package imports nothing from insyra, so insyra can import it.
package prep

import (
	"fmt"
	"math"
	"strings"

	"github.com/HazelnutParadise/insyra/internal/utils"
)

// NameOf reports the name a name selector carries. The selector type itself
// is insyra.NameSelector, which this package cannot import, so each caller
// passes a function that recognises it.
type NameOf func(selector any) (name string, ok bool)

// LookupColumn resolves a selector to a 0-based column position among
// columns, whose names nameOf reads; asName recognises a name selector. It returns the problem to report when
// nothing resolved, and a warning when the caller's choice was sound but worth
// mentioning. Both are returned rather than recorded, because Err() is sticky
// and the public method the caller made must be the one that names the failure.
//
// The rule is the one every selector in the library follows: a string is an
// Excel-style index, a Name is a name compared exactly, an int is a position
// counting from the end when negative. A bare string is never looked up as a
// name; the names only shape the failure message.
func LookupColumn[T any](columns []T, nameOf func(T) string, asName NameOf, selector any) (num int, warning, problem string) {
	if name, ok := asName(selector); ok {
		position, found := columnNumberByName(columns, nameOf, name)
		if !found {
			return -1, "", fmt.Sprintf("no column is named %s", name)
		}
		return position, "", ""
	}
	switch v := selector.(type) {
	case nil:
		return -1, "", "no column given, use an Excel-style index such as \"A\", Name(\"price\"), or an int position"
	case string:
		return lookupIndexString(columns, nameOf, v)
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

// lookupIndexString resolves a bare string, which is always an Excel-style
// index. The column names are read only to shape what a failure says, never to
// decide which column is used, so renaming a column cannot change what a call
// means.
func lookupIndexString[T any](columns []T, nameOf func(T) string, index string) (num int, warning, problem string) {
	position, ok := utils.ParseColIndex(index)
	if !ok {
		if _, named := columnNumberByName(columns, nameOf, index); named {
			return -1, "", fmt.Sprintf("%q is not an Excel-style column index, and this table has a column named %s, so write Name(%q)", index, index, index)
		}
		return -1, "", fmt.Sprintf("%q is not an Excel-style column index, and no column is named %s, write Name(%q) for a column name", index, index, index)
	}
	if position >= len(columns) {
		letters, _ := utils.CalcColIndex(position)
		if _, named := columnNumberByName(columns, nameOf, index); named {
			return -1, "", fmt.Sprintf("%q reads as column index %s, past the last of %d column(s), and this table has a column named %s, so write Name(%q)", index, letters, len(columns), index, index)
		}
		return -1, "", fmt.Sprintf("column %s does not exist, the table has %d column(s)", letters, len(columns))
	}
	if named, ok := columnNumberByName(columns, nameOf, index); ok && named != position {
		letters, _ := utils.CalcColIndex(position)
		warning = fmt.Sprintf("%q was read as column index %s (column %d), while column %d is named %s; write Name(%q) for the named one", index, letters, position, named, index, index)
	}
	return position, warning, ""
}

// columnNumberByName finds a column by exact name.
func columnNumberByName[T any](columns []T, nameOf func(T) string, name string) (int, bool) {
	for i, column := range columns {
		if nameOf(column) == name {
			return i, true
		}
	}
	return -1, false
}

// IsEmptySelector reports a selector that picks nothing: a nil, or the empty
// string a zero-valued config field carries.
func IsEmptySelector(selector any, asName NameOf) bool {
	if name, ok := asName(selector); ok {
		return strings.TrimSpace(name) == ""
	}
	switch v := selector.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(v) == ""
	default:
		return false
	}
}

// QuantileType7 returns the p-quantile (p in [0,1]) of an ascending-sorted,
// non-empty slice using R's type-7 method (the R / numpy / pandas default):
// linear interpolation at position h = p*(n-1). This is the single quantile
// definition shared by Quantile, Quartile, Percentile, preprocess.RobustScaler and
// Describe, so they never disagree with each other.
func QuantileType7(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return math.NaN()
	}
	if n == 1 {
		return sorted[0]
	}
	h := p * float64(n-1)
	lower := int(math.Floor(h))
	upper := int(math.Ceil(h))
	if lower < 0 {
		lower = 0
	}
	if upper >= n {
		upper = n - 1
	}
	if lower == upper {
		return sorted[lower]
	}
	return sorted[lower] + (h-float64(lower))*(sorted[upper]-sorted[lower])
}

// IntegerParts splits a Go integer into its value, widened to int64 when the
// type is signed and to uint64 when it is not.
func IntegerParts(v any) (s int64, u uint64, signed, ok bool) {
	switch x := v.(type) {
	case int:
		return int64(x), 0, true, true
	case int8:
		return int64(x), 0, true, true
	case int16:
		return int64(x), 0, true, true
	case int32:
		return int64(x), 0, true, true
	case int64:
		return x, 0, true, true
	case uint:
		return 0, uint64(x), false, true
	case uint8:
		return 0, uint64(x), false, true
	case uint16:
		return 0, uint64(x), false, true
	case uint32:
		return 0, uint64(x), false, true
	case uint64:
		return 0, x, false, true
	case uintptr:
		return 0, uint64(x), false, true
	}
	return 0, 0, false, false
}

// IsMissing reports whether v is nil or a float64 NaN, the cells the fill and
// impute operations treat as missing.
func IsMissing(v any) bool {
	if v == nil {
		return true
	}
	if f, ok := v.(float64); ok && math.IsNaN(f) {
		return true
	}
	return false
}

// IsNilOrNaN reports whether v is nil or a NaN float.
func IsNilOrNaN(v any) bool {
	if v == nil {
		return true
	}
	switch f := v.(type) {
	case float32:
		return math.IsNaN(float64(f))
	case float64:
		return math.IsNaN(f)
	}
	return false
}

// ExtraOptional returns the failure for a trailing optional parameter given
// more than one value, or "" when it got at most one. A trailing ...T or
// ...XxxOptions stands for one optional value; keeping the first and dropping
// the rest would hide from the caller that part of what they passed was
// ignored.
func ExtraOptional(what string, n int) string {
	if n <= 1 {
		return ""
	}
	return fmt.Sprintf("at most one %s may be given, got %d", what, n)
}

// AlphaColIndex computes the Excel-style label for a numeric column index.
func AlphaColIndex(i int) (string, bool) {
	if i < 0 {
		return "", false
	}
	name := ""
	for i >= 0 {
		name = string(rune('A'+(i%26))) + name
		i = i/26 - 1
	}
	return name, true
}
