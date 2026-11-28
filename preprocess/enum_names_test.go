package preprocess

import (
	"fmt"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// TestEnumsPrintTheirNames pins enum-conventions for the encoder enums: every
// numeric enum prints a name, and a value outside its constants prints
// TypeName(n).
func TestEnumsPrintTheirNames(t *testing.T) {
	cases := []struct {
		got  fmt.Stringer
		want string
	}{
		{LabelSort(99), "LabelSort(99)"},
		{NaNAsCategory, "NaNAsCategory"},
		{NaNError, "NaNError"},
		{NaNSkip, "NaNSkip"},
		{NaNPolicy(99), "NaNPolicy(99)"},
		{UnknownIgnore, "UnknownIgnore"},
		{UnknownError, "UnknownError"},
		{UnknownAsNew, "UnknownAsNew"},
		{UnknownPolicy(99), "UnknownPolicy(99)"},
		{LabelSortFirstSeen, "LabelSortFirstSeen"},
		{LabelSortLexicographic, "LabelSortLexicographic"},
		{LabelSortByFrequency, "LabelSortByFrequency"},
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
	dt := insyra.NewDataTable(insyra.NewDataList("a", "b"))
	_, _, err := LabelEncode(dt, LabelOptions{Column: "A", SortBy: LabelSort(9)})
	wantErrContains(t, err, "LabelSort(9)")
	_, _, err = OrdinalEncode(dt, OrdinalOptions{Column: "A", Order: []any{"a"}, Unknown: UnknownPolicy(9)})
	wantErrContains(t, err, "UnknownPolicy(9)")
}

func wantErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("error %v does not contain %q", err, want)
	}
}
