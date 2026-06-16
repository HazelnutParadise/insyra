package insyra

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func encodeTestTable() *DataTable {
	return NewDataTable(
		NewDataList("red", "blue", "red").SetName("color"),
		NewDataList("S", "M", "S").SetName("size"),
		NewDataList(10, 20, 30).SetName("value"),
	)
}

func assertEncodeData(t *testing.T, dl *DataList, want []any) {
	t.Helper()
	if dl == nil {
		t.Fatalf("got nil DataList")
	}
	got := dl.Data()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("data = %#v, want %#v", got, want)
	}
}

func assertEncodeCols(t *testing.T, dt *DataTable, want []string) {
	t.Helper()
	got := dt.ColNames()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("columns = %#v, want %#v", got, want)
	}
}

func TestOneHotEncodeBasicDropFirstKeepOriginalAndInverse(t *testing.T) {
	dt := encodeTestTable()
	out, enc, err := dt.OneHotEncode(OneHotOptions{Columns: []string{"color"}})
	if err != nil {
		t.Fatalf("OneHotEncode failed: %v", err)
	}
	assertEncodeCols(t, out, []string{"color_red", "color_blue", "size", "value"})
	assertEncodeData(t, out.GetColByName("color_red"), []any{1, 0, 1})
	assertEncodeData(t, out.GetColByName("color_blue"), []any{0, 1, 0})

	roundTrip, err := enc.InverseTransform(out)
	if err != nil {
		t.Fatalf("InverseTransform failed: %v", err)
	}
	assertEncodeCols(t, roundTrip, []string{"color", "size", "value"})
	assertEncodeData(t, roundTrip.GetColByName("color"), []any{"red", "blue", "red"})

	dropped, encDrop, err := dt.OneHotEncode(OneHotOptions{Columns: []string{"color"}, DropFirst: true, KeepOriginal: true})
	if err != nil {
		t.Fatalf("DropFirst OneHotEncode failed: %v", err)
	}
	assertEncodeCols(t, dropped, []string{"color", "color_blue", "size", "value"})
	assertEncodeData(t, dropped.GetColByName("color_blue"), []any{0, 1, 0})
	if got := encDrop.OutputColumns(); !reflect.DeepEqual(got, []string{"color_blue"}) {
		t.Fatalf("OutputColumns = %#v", got)
	}
	restored, err := encDrop.InverseTransform(dropped)
	if err != nil {
		t.Fatalf("DropFirst inverse failed: %v", err)
	}
	assertEncodeCols(t, restored, []string{"color", "size", "value"})
}

func TestOneHotEncodeMultiColumnPrefixSeparatorAndSort(t *testing.T) {
	dt := NewDataTable(
		NewDataList("b", "a").SetName("color"),
		NewDataList("S", "M").SetName("size"),
		NewDataList(1, 2).SetName("value"),
	)
	out, enc, err := dt.OneHotEncode(OneHotOptions{
		Columns:        []string{"color", "size"},
		Prefix:         "cat",
		Separator:      "__",
		SortCategories: true,
	})
	if err != nil {
		t.Fatalf("OneHotEncode failed: %v", err)
	}
	assertEncodeCols(t, out, []string{"cat__a", "cat__b", "cat__M", "cat__S", "value"})
	assertEncodeData(t, out.GetColByName("cat__a"), []any{0, 1})
	assertEncodeData(t, out.GetColByName("cat__b"), []any{1, 0})
	gotCats := enc.Categories()
	if !reflect.DeepEqual(gotCats["color"], []any{"a", "b"}) {
		t.Fatalf("color categories = %#v", gotCats["color"])
	}
}

func TestOneHotTransformUnknownPolicies(t *testing.T) {
	train := NewDataTable(NewDataList("red", "blue").SetName("color"))
	test := NewDataTable(NewDataList("green").SetName("color"))

	_, ignoreEnc, err := train.OneHotEncode(OneHotOptions{Columns: []string{"color"}, Unknown: UnknownIgnore})
	if err != nil {
		t.Fatalf("fit ignore encoder: %v", err)
	}
	ignored, err := ignoreEnc.Transform(test)
	if err != nil {
		t.Fatalf("ignore transform failed: %v", err)
	}
	assertEncodeData(t, ignored.GetColByName("color_red"), []any{0})
	assertEncodeData(t, ignored.GetColByName("color_blue"), []any{0})

	_, errorEnc, err := train.OneHotEncode(OneHotOptions{Columns: []string{"color"}, Unknown: UnknownError})
	if err != nil {
		t.Fatalf("fit error encoder: %v", err)
	}
	if _, err := errorEnc.Transform(test); err == nil {
		t.Fatalf("expected unknown-category error")
	}

	_, newEnc, err := train.OneHotEncode(OneHotOptions{Columns: []string{"color"}, Unknown: UnknownAsNew})
	if err != nil {
		t.Fatalf("fit new encoder: %v", err)
	}
	extended, err := newEnc.Transform(test)
	if err != nil {
		t.Fatalf("new transform failed: %v", err)
	}
	assertEncodeCols(t, extended, []string{"color_red", "color_blue", "color_green"})
	assertEncodeData(t, extended.GetColByName("color_green"), []any{1})
}

func TestOneHotNaNPoliciesEmptyAndSingleCategory(t *testing.T) {
	withMissing := NewDataTable(
		NewDataList(1, 2, 3).SetName("id"),
		NewDataList("red", nil, "blue").SetName("color"),
	)
	asCategory, enc, err := withMissing.OneHotEncode(OneHotOptions{Columns: []string{"color"}, HandleNaN: NaNAsCategory})
	if err != nil {
		t.Fatalf("NaNAsCategory failed: %v", err)
	}
	assertEncodeCols(t, asCategory, []string{"id", "color_red", "color_<nil>", "color_blue"})
	assertEncodeData(t, asCategory.GetColByName("color_<nil>"), []any{0, 1, 0})
	roundTrip, err := enc.InverseTransform(asCategory)
	if err != nil {
		t.Fatalf("inverse missing category: %v", err)
	}
	assertEncodeData(t, roundTrip.GetColByName("color"), []any{"red", nil, "blue"})

	skipped, _, err := withMissing.OneHotEncode(OneHotOptions{Columns: []string{"color"}, HandleNaN: NaNSkip})
	if err != nil {
		t.Fatalf("NaNSkip failed: %v", err)
	}
	assertEncodeData(t, skipped.GetColByName("color_red"), []any{1, 0, 0})
	assertEncodeData(t, skipped.GetColByName("color_blue"), []any{0, 0, 1})

	if _, _, err := withMissing.OneHotEncode(OneHotOptions{Columns: []string{"color"}, HandleNaN: NaNError}); err == nil {
		t.Fatalf("expected NaNError to fail")
	}
	if _, _, err := NewDataTable(NewDataList(math.NaN()).SetName("x")).OneHotEncode(OneHotOptions{Columns: []string{"x"}, HandleNaN: NaNError}); err == nil {
		t.Fatalf("expected NaNError to catch NaN")
	}

	empty := NewDataTable(NewDataList().SetName("color"))
	emptyOut, _, err := empty.OneHotEncode(OneHotOptions{Columns: []string{"color"}})
	if err != nil {
		t.Fatalf("empty column failed: %v", err)
	}
	if emptyOut.NumRows() != 0 || emptyOut.NumCols() != 0 {
		t.Fatalf("empty one-hot size = %dx%d", emptyOut.NumRows(), emptyOut.NumCols())
	}

	single := NewDataTable(
		NewDataList(1, 2).SetName("id"),
		NewDataList("red", "red").SetName("color"),
	)
	singleOut, singleEnc, err := single.OneHotEncode(OneHotOptions{Columns: []string{"color"}, DropFirst: true})
	if err != nil {
		t.Fatalf("single-category DropFirst failed: %v", err)
	}
	assertEncodeCols(t, singleOut, []string{"id"})
	singleBack, err := singleEnc.InverseTransform(singleOut)
	if err != nil {
		t.Fatalf("single-category inverse failed: %v", err)
	}
	assertEncodeCols(t, singleBack, []string{"id", "color"})
	assertEncodeData(t, singleBack.GetColByName("color"), []any{"red", "red"})
}

func TestLabelEncodeSortNewColumnKeepOriginalInverseAndIdentity(t *testing.T) {
	dt := NewDataTable(
		NewDataList("b", "a", "b").SetName("label"),
		NewDataList(1, 2, 3).SetName("value"),
	)
	firstSeen, enc, err := dt.LabelEncode(LabelEncodeOptions{Column: "label"})
	if err != nil {
		t.Fatalf("LabelEncode failed: %v", err)
	}
	assertEncodeCols(t, firstSeen, []string{"label", "value"})
	assertEncodeData(t, firstSeen.GetColByName("label"), []any{0, 1, 0})
	if got := enc.Classes(); !reflect.DeepEqual(got, []any{"b", "a"}) {
		t.Fatalf("classes = %#v", got)
	}
	if inv, err := enc.Inverse(0, 1, 0); err != nil || !reflect.DeepEqual(inv, []any{"b", "a", "b"}) {
		t.Fatalf("Inverse variadic = %#v, %v", inv, err)
	}
	if inv, err := enc.Inverse([]int{1, 0}); err != nil || !reflect.DeepEqual(inv, []any{"a", "b"}) {
		t.Fatalf("Inverse slice = %#v, %v", inv, err)
	}
	roundTrip, err := enc.InverseTransform(firstSeen)
	if err != nil {
		t.Fatalf("label inverse transform: %v", err)
	}
	assertEncodeData(t, roundTrip.GetColByName("label"), []any{"b", "a", "b"})

	lex, _, err := dt.LabelEncode(LabelEncodeOptions{Column: "label", NewColumn: "label_id", SortBy: LabelSortLexicographic})
	if err != nil {
		t.Fatalf("lex LabelEncode failed: %v", err)
	}
	assertEncodeCols(t, lex, []string{"label_id", "value"})
	assertEncodeData(t, lex.GetColByName("label_id"), []any{1, 0, 1})

	freqSrc := NewDataTable(NewDataList("b", "a", "a", "b", "a").SetName("label"))
	freq, _, err := freqSrc.LabelEncode(LabelEncodeOptions{Column: "label", SortBy: LabelSortByFrequency})
	if err != nil {
		t.Fatalf("freq LabelEncode failed: %v", err)
	}
	assertEncodeData(t, freq.GetColByName("label"), []any{1, 0, 0, 1, 0})

	kept, _, err := dt.LabelEncode(LabelEncodeOptions{Column: "label", NewColumn: "label_id", KeepOriginal: true})
	if err != nil {
		t.Fatalf("keep LabelEncode failed: %v", err)
	}
	assertEncodeCols(t, kept, []string{"label", "label_id", "value"})

	typed, typedEnc, err := NewDataTable(NewDataList(1, "1").SetName("mixed")).LabelEncode(LabelEncodeOptions{Column: "mixed"})
	if err != nil {
		t.Fatalf("typed LabelEncode failed: %v", err)
	}
	assertEncodeData(t, typed.GetColByName("mixed"), []any{0, 1})
	if got := typedEnc.Classes(); !reflect.DeepEqual(got, []any{1, "1"}) {
		t.Fatalf("typed classes = %#v", got)
	}
}

func TestLabelTransformUnknownAndNaNPolicies(t *testing.T) {
	train := NewDataTable(NewDataList("red", "blue").SetName("color"))
	test := NewDataTable(NewDataList("green").SetName("color"))

	_, ignoreEnc, err := train.LabelEncode(LabelEncodeOptions{Column: "color", Unknown: UnknownIgnore})
	if err != nil {
		t.Fatalf("fit ignore label: %v", err)
	}
	ignored, err := ignoreEnc.Transform(test)
	if err != nil {
		t.Fatalf("label ignore transform: %v", err)
	}
	assertEncodeData(t, ignored.GetColByName("color"), []any{nil})

	_, errorEnc, err := train.LabelEncode(LabelEncodeOptions{Column: "color", Unknown: UnknownError})
	if err != nil {
		t.Fatalf("fit error label: %v", err)
	}
	if _, err := errorEnc.Transform(test); err == nil {
		t.Fatalf("expected label unknown error")
	}

	_, newEnc, err := train.LabelEncode(LabelEncodeOptions{Column: "color", Unknown: UnknownAsNew})
	if err != nil {
		t.Fatalf("fit new label: %v", err)
	}
	extended, err := newEnc.Transform(test)
	if err != nil {
		t.Fatalf("label new transform: %v", err)
	}
	assertEncodeData(t, extended.GetColByName("color"), []any{2})

	missing := NewDataTable(NewDataList("red", nil).SetName("color"))
	asCategory, _, err := missing.LabelEncode(LabelEncodeOptions{Column: "color", HandleNaN: NaNAsCategory})
	if err != nil {
		t.Fatalf("label NaNAsCategory: %v", err)
	}
	assertEncodeData(t, asCategory.GetColByName("color"), []any{0, 1})
	skipped, _, err := missing.LabelEncode(LabelEncodeOptions{Column: "color", HandleNaN: NaNSkip})
	if err != nil {
		t.Fatalf("label NaNSkip: %v", err)
	}
	assertEncodeData(t, skipped.GetColByName("color"), []any{0, nil})
	if _, _, err := missing.LabelEncode(LabelEncodeOptions{Column: "color", HandleNaN: NaNError}); err == nil {
		t.Fatalf("expected label NaNError")
	}
}

func TestOrdinalEncodeOrderUnknownNaNAndInverse(t *testing.T) {
	dt := NewDataTable(
		NewDataList("low", "high", "medium").SetName("satisfaction"),
		NewDataList(1, 2, 3).SetName("value"),
	)
	out, enc, err := dt.OrdinalEncode(OrdinalEncodeOptions{
		Column:    "satisfaction",
		Order:     []any{"low", "medium", "high"},
		NewColumn: "rank",
	})
	if err != nil {
		t.Fatalf("OrdinalEncode failed: %v", err)
	}
	assertEncodeCols(t, out, []string{"rank", "value"})
	assertEncodeData(t, out.GetColByName("rank"), []any{0, 2, 1})
	if inv, err := enc.Inverse(0, 2, 1); err != nil || !reflect.DeepEqual(inv, []any{"low", "high", "medium"}) {
		t.Fatalf("ordinal inverse = %#v, %v", inv, err)
	}
	roundTrip, err := enc.InverseTransform(out)
	if err != nil {
		t.Fatalf("ordinal inverse transform: %v", err)
	}
	assertEncodeData(t, roundTrip.GetColByName("satisfaction"), []any{"low", "high", "medium"})

	if _, _, err := dt.OrdinalEncode(OrdinalEncodeOptions{Column: "satisfaction"}); err == nil {
		t.Fatalf("expected empty Order error")
	}
	if _, _, err := dt.OrdinalEncode(OrdinalEncodeOptions{
		Column:  "satisfaction",
		Order:   []any{"low", "medium"},
		Unknown: UnknownError,
	}); err == nil {
		t.Fatalf("expected unknown-in-fit error")
	}

	ignored, _, err := dt.OrdinalEncode(OrdinalEncodeOptions{
		Column:  "satisfaction",
		Order:   []any{"low", "medium"},
		Unknown: UnknownIgnore,
	})
	if err != nil {
		t.Fatalf("ordinal unknown ignore: %v", err)
	}
	assertEncodeData(t, ignored.GetColByName("satisfaction"), []any{0, nil, 1})

	_, errEnc, err := NewDataTable(NewDataList("low").SetName("satisfaction")).OrdinalEncode(OrdinalEncodeOptions{
		Column:  "satisfaction",
		Order:   []any{"low"},
		Unknown: UnknownError,
	})
	if err != nil {
		t.Fatalf("fit ordinal strict: %v", err)
	}
	if _, err := errEnc.Transform(NewDataTable(NewDataList("high").SetName("satisfaction"))); err == nil {
		t.Fatalf("expected ordinal transform unknown error")
	}
	_, newEnc, err := NewDataTable(NewDataList("low").SetName("satisfaction")).OrdinalEncode(OrdinalEncodeOptions{
		Column:  "satisfaction",
		Order:   []any{"low"},
		Unknown: UnknownAsNew,
	})
	if err != nil {
		t.Fatalf("fit ordinal new: %v", err)
	}
	extended, err := newEnc.Transform(NewDataTable(NewDataList("high").SetName("satisfaction")))
	if err != nil {
		t.Fatalf("ordinal new transform: %v", err)
	}
	assertEncodeData(t, extended.GetColByName("satisfaction"), []any{1})

	missing := NewDataTable(NewDataList("low", nil).SetName("satisfaction"))
	asCategory, _, err := missing.OrdinalEncode(OrdinalEncodeOptions{
		Column:    "satisfaction",
		Order:     []any{"low"},
		HandleNaN: NaNAsCategory,
	})
	if err != nil {
		t.Fatalf("ordinal NaNAsCategory: %v", err)
	}
	assertEncodeData(t, asCategory.GetColByName("satisfaction"), []any{0, 1})
	skipped, _, err := missing.OrdinalEncode(OrdinalEncodeOptions{
		Column:    "satisfaction",
		Order:     []any{"low"},
		HandleNaN: NaNSkip,
	})
	if err != nil {
		t.Fatalf("ordinal NaNSkip: %v", err)
	}
	assertEncodeData(t, skipped.GetColByName("satisfaction"), []any{0, nil})
	if _, _, err := missing.OrdinalEncode(OrdinalEncodeOptions{
		Column:    "satisfaction",
		Order:     []any{"low"},
		HandleNaN: NaNError,
	}); err == nil {
		t.Fatalf("expected ordinal NaNError")
	}
}

func TestEncodeErrorsUseColumnResolutionAndDuplicateNames(t *testing.T) {
	dt := encodeTestTable()
	if _, _, err := dt.OneHotEncode(OneHotOptions{Columns: []string{"missing"}}); err == nil {
		t.Fatalf("expected missing column error")
	}
	byIndex, _, err := dt.OneHotEncode(OneHotOptions{Columns: []string{"A"}})
	if err != nil {
		t.Fatalf("Excel-style column reference failed: %v", err)
	}
	assertEncodeData(t, byIndex.GetColByName("color_red"), []any{1, 0, 1})

	_, _, err = dt.LabelEncode(LabelEncodeOptions{Column: "color", NewColumn: "size"})
	if err == nil || !strings.Contains(err.Error(), "duplicate output column name") {
		t.Fatalf("expected duplicate-name error, got %v", err)
	}
}
