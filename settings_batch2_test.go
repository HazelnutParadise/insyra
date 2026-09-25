package insyra

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func fillTable() *DataTable {
	return NewDataTable(
		NewDataList(10.0, nil, 30.0).SetName("price"),
		NewDataList("Taipei", nil, "Taichung").SetName("city"),
		NewDataList(nil, nil, nil).SetName("blank"),
	)
}

// A table-level fill skips a column it cannot fill when the caller named no
// columns, because it was asked to do what it can. A column the caller named
// is different: skipping it silently let them think it was filled.
func TestTableFillReportsANamedColumnItCannotFill(t *testing.T) {
	fills := map[string]func(dt *DataTable, cols ...any) *DataTable{
		"FillWithMean":   func(dt *DataTable, cols ...any) *DataTable { return dt.FillWithMean(cols...) },
		"FillWithMedian": func(dt *DataTable, cols ...any) *DataTable { return dt.FillWithMedian(cols...) },
		"FillByInterpolation": func(dt *DataTable, cols ...any) *DataTable {
			return dt.FillByInterpolation(false, cols...)
		},
	}
	for name, fill := range fills {
		t.Run(name+"/text column named", func(t *testing.T) {
			dt := fillTable()
			fill(dt, Name("price"), Name("city"))
			err := dt.Err()
			if err == nil || !strings.Contains(err.Error(), "city") || !strings.Contains(err.Error(), "string") {
				t.Fatalf("got %v, want an error naming city and its string values", err)
			}
			if dt.GetColByName("price").Get(1) == nil {
				t.Error("the fillable named column was left unfilled")
			}
		})
		t.Run(name+"/empty column named", func(t *testing.T) {
			dt := fillTable()
			fill(dt, Name("blank"))
			if err := dt.Err(); err == nil || !strings.Contains(err.Error(), "blank") {
				t.Fatalf("got %v, want an error naming the column with no values", err)
			}
		})
		t.Run(name+"/no column named", func(t *testing.T) {
			dt := fillTable()
			fill(dt)
			if err := dt.Err(); err != nil {
				t.Fatalf("filling every column reported %v", err)
			}
			if dt.GetColByName("price").Get(1) == nil {
				t.Error("price was not filled")
			}
			if dt.GetColByName("city").Get(1) != nil {
				t.Error("city was changed")
			}
		})
	}
}

// The table version could not extrapolate at all, so the CLI's extrapolate
// option was dropped for tables.
func TestTableFillByInterpolationCanExtrapolate(t *testing.T) {
	build := func() *DataTable { return NewDataTable(NewDataList(nil, 2.0, 3.0, nil).SetName("x")) }
	if got := build().FillByInterpolation(true).GetColByName("x").Data(); !reflect.DeepEqual(got, []any{1.0, 2.0, 3.0, 4.0}) {
		t.Fatalf("with extrapolation: %v", got)
	}
	if got := build().FillByInterpolation(false).GetColByName("x").Data(); got[0] != nil || got[3] != nil {
		t.Fatalf("without extrapolation the ends were filled: %v", got)
	}
}

func TestSimpleImputerTakesOptions(t *testing.T) {
	train := NewDataTable(NewDataList(10.0, 30.0, nil).SetName("v"))

	out, err := NewSimpleImputer().FitTransform(train, Name("v"))
	if err != nil || out.GetColByName("v").Get(2) != 20.0 {
		t.Fatalf("default strategy: %v, %v; want the mean, 20", err, out.GetColByName("v").Data())
	}
	out, err = NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeConstant, FillValue: 7.0}).FitTransform(train, Name("v"))
	if err != nil || out.GetColByName("v").Get(2) != 7.0 {
		t.Fatalf("constant: %v, %v", err, out.GetColByName("v").Data())
	}
	refused := map[string]*SimpleImputer{
		"a fill value with the mean":     NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeMean, FillValue: 7.0}),
		"the constant without a value":   NewSimpleImputer(SimpleImputerOptions{Strategy: ImputeConstant}),
		"two options structs":            NewSimpleImputer(SimpleImputerOptions{}, SimpleImputerOptions{}),
		"a strategy that does not exist": NewSimpleImputer(SimpleImputerOptions{Strategy: "mystery"}),
	}
	for name, imputer := range refused {
		if err := imputer.Fit(train, Name("v")); err == nil {
			t.Errorf("%s was accepted", name)
		}
	}
}

// ShowRange takes nothing, a count, or a start and an end. A third value or a
// value that is not an int used to be ignored and the whole table shown, so
// the caller never learned that the range they asked for was not the one
// displayed.
func TestShowRangeRefusesWhatItCannotRead(t *testing.T) {
	colored := Config.GetDoesUseColoredOutput()
	Config.SetUseColoredOutput(false)
	defer Config.SetUseColoredOutput(colored)

	dt := fillTable()
	dl := NewDataList(1, 2, 3)
	bad := [][]any{{1, 2, 3}, {"a"}, {0, "b"}}
	for _, args := range bad {
		for name, show := range map[string]func(*bytes.Buffer){
			"DataTable.ShowRangeTo":      func(b *bytes.Buffer) { dt.ShowRangeTo(b, args...) },
			"DataTable.ShowTypesRangeTo": func(b *bytes.Buffer) { dt.ShowTypesRangeTo(b, args...) },
			"DataList.ShowRangeTo":       func(b *bytes.Buffer) { dl.ShowRangeTo(b, args...) },
			"DataList.ShowTypesRangeTo":  func(b *bytes.Buffer) { dl.ShowTypesRangeTo(b, args...) },
		} {
			var buf bytes.Buffer
			show(&buf)
			if !strings.Contains(buf.String(), "ERROR") {
				t.Errorf("%s(%v) showed:\n%s", name, args, buf.String())
			}
		}
	}
}

func TestShowHeadAndTail(t *testing.T) {
	colored := Config.GetDoesUseColoredOutput()
	Config.SetUseColoredOutput(false)
	defer Config.SetUseColoredOutput(colored)

	dl := NewDataList("first", "middle", "last")
	var head, tail, want bytes.Buffer
	dl.ShowHeadTo(&head, 2)
	dl.ShowRangeTo(&want, 2)
	if head.String() != want.String() {
		t.Errorf("ShowHead(2) differs from ShowRange(2):\n%s\n%s", head.String(), want.String())
	}
	want.Reset()
	dl.ShowTailTo(&tail, 1)
	dl.ShowRangeTo(&want, -1)
	if tail.String() != want.String() {
		t.Errorf("ShowTail(1) differs from ShowRange(-1):\n%s\n%s", tail.String(), want.String())
	}

	dt := fillTable()
	var th, tw bytes.Buffer
	dt.ShowHeadTo(&th, 1)
	dt.ShowRangeTo(&tw, 1)
	if th.String() != tw.String() {
		t.Errorf("DataTable.ShowHead(1) differs from ShowRange(1)")
	}
	th.Reset()
	tw.Reset()
	dt.ShowTailTo(&th, 1)
	dt.ShowRangeTo(&tw, -1)
	if th.String() != tw.String() {
		t.Errorf("DataTable.ShowTail(1) differs from ShowRange(-1)")
	}

	var bad bytes.Buffer
	dl.ShowHeadTo(&bad, 0)
	dt.ShowTailTo(&bad, -2)
	if strings.Count(bad.String(), "ERROR") != 2 {
		t.Errorf("a count that is not positive was accepted:\n%s", bad.String())
	}
}
