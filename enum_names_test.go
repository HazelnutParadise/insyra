package insyra

import (
	"fmt"
	"strings"
	"testing"
)

// TestEnumsPrintTheirNames pins enum-conventions: every numeric enum
// prints a name, and a value outside its constants prints TypeName(n).
func TestEnumsPrintTheirNames(t *testing.T) {
	cases := []struct {
		got  fmt.Stringer
		want string
	}{
		{AggregateOp(99), "AggregateOp(99)"},
		{DataType(99), "DataType(99)"},
		{ErrPoppingModeFIFO, "ErrPoppingModeFIFO"},
		{ErrPoppingModeLIFO, "ErrPoppingModeLIFO"},
		{ErrPoppingMode(99), "ErrPoppingMode(99)"},
		{LabelSortFirstSeen, "LabelSortFirstSeen"},
		{LabelSortLexicographic, "LabelSortLexicographic"},
		{LabelSortByFrequency, "LabelSortByFrequency"},
		{LabelSort(99), "LabelSort(99)"},
		{LogLevel(99), "LogLevel(99)"},
		{MergeDirectionHorizontal, "MergeDirectionHorizontal"},
		{MergeDirectionVertical, "MergeDirectionVertical"},
		{MergeDirection(99), "MergeDirection(99)"},
		{MergeModeInner, "MergeModeInner"},
		{MergeModeOuter, "MergeModeOuter"},
		{MergeModeLeft, "MergeModeLeft"},
		{MergeModeRight, "MergeModeRight"},
		{MergeMode(99), "MergeMode(99)"},
		{NaNAsCategory, "NaNAsCategory"},
		{NaNError, "NaNError"},
		{NaNSkip, "NaNSkip"},
		{NaNPolicy(99), "NaNPolicy(99)"},
		{ResampleWeekly, "ResampleWeekly"},
		{ResampleMonthly, "ResampleMonthly"},
		{ResampleQuarterly, "ResampleQuarterly"},
		{ResampleYearly, "ResampleYearly"},
		{ResampleFreq(99), "ResampleFreq(99)"},
		{SQLActionIfTableExistsFail, "SQLActionIfTableExistsFail"},
		{SQLActionIfTableExistsReplace, "SQLActionIfTableExistsReplace"},
		{SQLActionIfTableExistsAppend, "SQLActionIfTableExistsAppend"},
		{SQLActionIfTableExists(99), "SQLActionIfTableExists(99)"},
		{SheetExistsFail, "SheetExistsFail"},
		{SheetExistsReplace, "SheetExistsReplace"},
		{SheetExistsPolicy(99), "SheetExistsPolicy(99)"},
		{UnknownIgnore, "UnknownIgnore"},
		{UnknownError, "UnknownError"},
		{UnknownAsNew, "UnknownAsNew"},
		{UnknownPolicy(99), "UnknownPolicy(99)"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(c.got); got != c.want {
			t.Errorf("%T: got %q, want %q", c.got, got, c.want)
		}
	}
}

// TestEnumErrorsNameTheValue pins that an error reporting an unknown enum
// value names it instead of printing a bare number.
func TestEnumErrorsNameTheValue(t *testing.T) {
	dt := NewDataTable(NewDataList("a", "b"))
	_, _, err := dt.LabelEncode(LabelEncodeOptions{Column: "A", SortBy: LabelSort(9)})
	wantErrContains(t, err, "LabelSort(9)")
	_, _, err = dt.OrdinalEncode(OrdinalEncodeOptions{Column: "A", Order: []any{"a"}, Unknown: UnknownPolicy(9)})
	wantErrContains(t, err, "UnknownPolicy(9)")
	_, err = dt.Resample("A", ResampleFreq(9), ResampleAgg{Col: "A", Op: OpCount})
	wantErrContains(t, err, "ResampleFreq(9)")
}

func wantErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("error %v does not contain %q", err, want)
	}
}
