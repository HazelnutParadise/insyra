package insyra

import (
	"strings"
	"testing"
)

// A column name given as a bare string is read as an Excel-style index, and
// the failure has to say so and teach Name(...). The fitters resolved columns
// through a helper that dropped that explanation, so they answered "column Age
// not found" about a column that is plainly there.
func TestFittersTeachNameForABareColumnName(t *testing.T) {
	table := func() *DataTable {
		return NewDataTable(
			NewDataList(20.0, 30.0, 40.0).SetName("Age"),
			NewDataList("a", "b", "a").SetName("segment"),
		)
	}
	cases := []struct {
		name string
		want string
		run  func() error
	}{
		{"StandardScaler.FitTransform", `Name("Age")`, func() error {
			_, err := NewStandardScaler().FitTransform(table(), "Age")
			return err
		}},
		{"SimpleImputer.Fit", `Name("Age")`, func() error {
			return NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeMean}).Fit(table(), "Age")
		}},
		{"OneHotEncode", `Name("segment")`, func() error {
			_, _, err := table().OneHotEncode(OneHotOptions{Columns: []any{"segment"}})
			return err
		}},
		{"LabelEncode", `Name("segment")`, func() error {
			_, _, err := table().LabelEncode(LabelEncodeOptions{Column: "segment"})
			return err
		}},
		{"OrdinalEncode", `Name("segment")`, func() error {
			_, _, err := table().OrdinalEncode(OrdinalEncodeOptions{Column: "segment", Order: []any{"a", "b"}})
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatal("a bare column name was accepted")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not say to write %s", err, tc.want)
			}
		})
	}
}
