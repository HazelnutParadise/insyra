package preprocess

import (
	"fmt"
	"slices"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestEncoderCategoriesMatchIntegersAcrossTypes(t *testing.T) {
	src := insyra.NewDataTable(insyra.NewDataList(int64(1), int64(2), int64(3), int64(2)).SetName("v"))
	out, _, err := OrdinalEncode(src, OrdinalOptions{Column: insyra.Name("v"), Order: []any{1, 2, 3}, Unknown: UnknownError})
	if err != nil {
		t.Fatalf("OrdinalEncode with Order written as int literals: %v", err)
	}
	if got := fmt.Sprint(out.GetColByNumber(0).Data()); got != "[0 1 2 1]" {
		t.Errorf("OrdinalEncode gave %s, want [0 1 2 1]", got)
	}

	fitted := insyra.NewDataTable(insyra.NewDataList(1, 2, 3).SetName("v"))
	_, enc, err := LabelEncode(fitted, LabelOptions{Column: insyra.Name("v"), Unknown: UnknownError})
	if err != nil {
		t.Fatalf("LabelEncode: %v", err)
	}
	moved, err := enc.Transform(insyra.NewDataTable(insyra.NewDataList(int64(3), int64(1)).SetName("v")))
	if err != nil {
		t.Fatalf("an encoder fitted on int data rejected int64 data: %v", err)
	}
	if got := fmt.Sprint(moved.GetColByNumber(0).Data()); got != "[2 0]" {
		t.Errorf("Transform gave %s, want [2 0]", got)
	}

	// int(1) and int64(1) in one column are one category, not two that
	// render as the same column name.
	mixed := insyra.NewDataTable(insyra.NewDataList(1, int64(1), 2).SetName("v"))
	_, oh, err := OneHotEncode(mixed, OneHotOptions{Columns: []any{insyra.Name("v")}})
	if err != nil {
		t.Fatalf("OneHotEncode on a column mixing int and int64: %v", err)
	}
	if cats := oh.Categories()["v"]; len(cats) != 2 {
		t.Errorf("categories %v, want two (1 and 2)", cats)
	}
}

// Column names generated from values follow the same rule.
// restoreConfig puts back the logging and error settings a test changed.
func restoreConfig(t *testing.T) {
	t.Helper()
	level := insyra.Config.GetLogLevel()
	panicOnError := insyra.Config.GetPanicOnError()
	colored := insyra.Config.GetUseColoredOutput()
	hook := insyra.Config.GetDefaultErrHandlingFunc()
	t.Cleanup(func() {
		insyra.Config.SetLogLevel(level)
		insyra.Config.SetPanicOnError(panicOnError)
		insyra.Config.SetUseColoredOutput(colored)
		insyra.Config.SetDefaultErrHandlingFunc(hook)
	})
}

func TestNumberTextInOneHotColumnNames(t *testing.T) {
	restoreConfig(t)
	insyra.Config.SetLogLevel(insyra.LogLevelFatal)

	dt := insyra.NewDataTable(insyra.NewDataList(1500000.0, 2500000.0, 1500000.0).SetName("price"))
	out, _, err := OneHotEncode(dt, OneHotOptions{Columns: []any{insyra.Name("price")}})
	if err != nil {
		t.Fatal(err)
	}
	names := out.ColNames()
	for _, want := range []string{"price_1500000", "price_2500000"} {
		if !slices.Contains(names, want) {
			t.Errorf("one-hot columns = %v, want %q among them", names, want)
		}
	}
}

// nil is not a number and keeps its documented name.
func TestNilOneHotCategoryNameUnchanged(t *testing.T) {
	if got := oneHotCategoryColumnName("c", "", nil); got != "c_<nil>" {
		t.Errorf("one-hot nil name = %q, want %q", got, "c_<nil>")
	}
}
