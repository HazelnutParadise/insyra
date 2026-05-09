package insyra

import (
	"math"
	"reflect"
	"testing"
)

func pivotLongTable() *DataTable {
	region := NewDataList("APAC", "APAC", "EMEA").SetName("region")
	product := NewDataList("A", "B", "A").SetName("product")
	sales := NewDataList(10, 20, 30).SetName("sales")
	t := NewDataTable()
	t.AppendCols(region, product, sales)
	return t
}

func pivotDuplicateLongTable() *DataTable {
	region := NewDataList("APAC", "APAC", "APAC", "EMEA").SetName("region")
	product := NewDataList("A", "A", "B", "A").SetName("product")
	sales := NewDataList(10, 5, 20, 30).SetName("sales")
	t := NewDataTable()
	t.AppendCols(region, product, sales)
	return t
}

func TestPivot_UniqueIndexColumns(t *testing.T) {
	dt := pivotLongTable()
	wide, err := dt.Pivot(PivotConfig{
		Index:    []string{"region"},
		Columns:  "product",
		Values:   "sales",
		FillNA:   0,
		SortCols: true,
	})
	if err != nil {
		t.Fatalf("Pivot failed: %v", err)
	}
	headers := wide.ColNames()
	want := []string{"region", "A", "B"}
	if !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers = %v, want %v", headers, want)
	}
	if wide.NumRows() != 2 {
		t.Fatalf("rows = %d, want 2", wide.NumRows())
	}

	regions := wide.GetColByName("region").Data()
	if !reflect.DeepEqual(regions, []any{"APAC", "EMEA"}) {
		t.Errorf("region order = %v, want [APAC EMEA]", regions)
	}
	colA := wide.GetColByName("A").Data()
	colB := wide.GetColByName("B").Data()

	if got, _ := ToFloat64Safe(colA[0]); got != 10 {
		t.Errorf("APAC.A = %v, want 10", colA[0])
	}
	if got, _ := ToFloat64Safe(colA[1]); got != 30 {
		t.Errorf("EMEA.A = %v, want 30", colA[1])
	}
	if got, _ := ToFloat64Safe(colB[0]); got != 20 {
		t.Errorf("APAC.B = %v, want 20", colB[0])
	}
	// EMEA had no row for product=B; should have filled with FillNA=0
	if got, _ := ToFloat64Safe(colB[1]); got != 0 {
		t.Errorf("EMEA.B (FillNA) = %v, want 0", colB[1])
	}
}

func TestPivot_AggSum(t *testing.T) {
	dt := pivotDuplicateLongTable()
	wide, err := dt.Pivot(PivotConfig{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
		AggFunc: "sum",
		FillNA:  0,
	})
	if err != nil {
		t.Fatalf("Pivot failed: %v", err)
	}
	colA := wide.GetColByName("A").Data()
	if got, _ := ToFloat64Safe(colA[0]); got != 15 {
		t.Errorf("APAC.A sum = %v, want 15", colA[0])
	}
}

func TestPivot_AggMean(t *testing.T) {
	dt := pivotDuplicateLongTable()
	wide, err := dt.Pivot(PivotConfig{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
		AggFunc: "mean",
	})
	if err != nil {
		t.Fatalf("Pivot failed: %v", err)
	}
	colA := wide.GetColByName("A").Data()
	if got, _ := ToFloat64Safe(colA[0]); got != 7.5 {
		t.Errorf("APAC.A mean = %v, want 7.5", colA[0])
	}
}

func TestPivot_AggCount(t *testing.T) {
	dt := pivotDuplicateLongTable()
	wide, err := dt.Pivot(PivotConfig{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
		AggFunc: "count",
		FillNA:  0,
	})
	if err != nil {
		t.Fatalf("Pivot failed: %v", err)
	}
	colA := wide.GetColByName("A").Data()
	if got, _ := ToFloat64Safe(colA[0]); got != 2 {
		t.Errorf("APAC.A count = %v, want 2", colA[0])
	}
}

func TestPivot_AggCustom(t *testing.T) {
	dt := pivotDuplicateLongTable()
	wide, err := dt.Pivot(PivotConfig{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
		AggFunc: "custom",
		Custom: func(group *DataList) any {
			return group.Max()
		},
	})
	if err != nil {
		t.Fatalf("Pivot failed: %v", err)
	}
	colA := wide.GetColByName("A").Data()
	if got, _ := ToFloat64Safe(colA[0]); got != 10 {
		t.Errorf("APAC.A custom max = %v, want 10", colA[0])
	}
}

func TestPivot_DuplicateWithoutAggErrors(t *testing.T) {
	dt := pivotDuplicateLongTable()
	_, err := dt.Pivot(PivotConfig{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
	})
	if err == nil {
		t.Fatalf("expected error when duplicates exist with no AggFunc")
	}
}

func TestPivot_MissingColumnsError(t *testing.T) {
	dt := pivotLongTable()
	if _, err := dt.Pivot(PivotConfig{Index: []string{"nope"}, Columns: "product", Values: "sales"}); err == nil {
		t.Errorf("expected error for missing index column")
	}
	if _, err := dt.Pivot(PivotConfig{Index: []string{"region"}, Columns: "nope", Values: "sales"}); err == nil {
		t.Errorf("expected error for missing columns column")
	}
	if _, err := dt.Pivot(PivotConfig{Index: []string{"region"}, Columns: "product", Values: "nope"}); err == nil {
		t.Errorf("expected error for missing values column")
	}
}

func TestPivot_MissingRequiredFields(t *testing.T) {
	dt := pivotLongTable()
	if _, err := dt.Pivot(PivotConfig{Columns: "product", Values: "sales"}); err == nil {
		t.Errorf("expected error for empty Index")
	}
	if _, err := dt.Pivot(PivotConfig{Index: []string{"region"}, Values: "sales"}); err == nil {
		t.Errorf("expected error for empty Columns")
	}
	if _, err := dt.Pivot(PivotConfig{Index: []string{"region"}, Columns: "product"}); err == nil {
		t.Errorf("expected error for empty Values")
	}
}

func TestPivot_EmptyTable(t *testing.T) {
	region := NewDataList().SetName("region")
	product := NewDataList().SetName("product")
	sales := NewDataList().SetName("sales")
	dt := NewDataTable()
	dt.AppendCols(region, product, sales)

	wide, err := dt.Pivot(PivotConfig{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
	})
	if err != nil {
		t.Fatalf("Pivot on empty table failed: %v", err)
	}
	if wide.NumRows() != 0 {
		t.Errorf("expected 0 rows, got %d", wide.NumRows())
	}
	if got := wide.ColNames(); !reflect.DeepEqual(got, []string{"region"}) {
		t.Errorf("expected only index column, got %v", got)
	}
}

func TestPivot_MultipleIndexCols(t *testing.T) {
	year := NewDataList(2024, 2024, 2025, 2025).SetName("year")
	region := NewDataList("APAC", "APAC", "APAC", "EMEA").SetName("region")
	product := NewDataList("A", "B", "A", "A").SetName("product")
	sales := NewDataList(10, 20, 30, 40).SetName("sales")
	dt := NewDataTable()
	dt.AppendCols(year, region, product, sales)

	wide, err := dt.Pivot(PivotConfig{
		Index:    []string{"year", "region"},
		Columns:  "product",
		Values:   "sales",
		FillNA:   0,
		SortCols: true,
	})
	if err != nil {
		t.Fatalf("Pivot failed: %v", err)
	}
	if wide.NumRows() != 3 {
		t.Fatalf("rows = %d, want 3", wide.NumRows())
	}
	if got := wide.ColNames(); !reflect.DeepEqual(got, []string{"year", "region", "A", "B"}) {
		t.Errorf("headers = %v", got)
	}
}

func TestPivot_SortColsFalse_KeepsFirstSeen(t *testing.T) {
	region := NewDataList("APAC", "APAC", "APAC").SetName("region")
	product := NewDataList("Z", "A", "M").SetName("product")
	sales := NewDataList(1, 2, 3).SetName("sales")
	dt := NewDataTable()
	dt.AppendCols(region, product, sales)

	wide, err := dt.Pivot(PivotConfig{
		Index:    []string{"region"},
		Columns:  "product",
		Values:   "sales",
		SortCols: false,
	})
	if err != nil {
		t.Fatalf("Pivot failed: %v", err)
	}
	if got := wide.ColNames(); !reflect.DeepEqual(got, []string{"region", "Z", "A", "M"}) {
		t.Errorf("expected first-seen order, got %v", got)
	}
}

func TestPivot_IndexColumnOverlap(t *testing.T) {
	dt := pivotLongTable()
	if _, err := dt.Pivot(PivotConfig{
		Index:   []string{"region", "product"},
		Columns: "product",
		Values:  "sales",
	}); err == nil {
		t.Errorf("expected error when Index includes Columns column")
	}
}

func TestPivot_CustomMissingFuncErrors(t *testing.T) {
	dt := pivotDuplicateLongTable()
	if _, err := dt.Pivot(PivotConfig{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
		AggFunc: "custom",
	}); err == nil {
		t.Errorf("expected error when AggFunc=custom but Custom is nil")
	}
}

func TestPivot_UnknownAggFuncErrors(t *testing.T) {
	dt := pivotDuplicateLongTable()
	if _, err := dt.Pivot(PivotConfig{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
		AggFunc: "wat",
	}); err == nil {
		t.Errorf("expected error for unknown AggFunc")
	}
}

func TestPivot_DoesNotMutateSource(t *testing.T) {
	dt := pivotLongTable()
	origRows := dt.NumRows()
	origHeaders := append([]string(nil), dt.ColNames()...)
	if _, err := dt.Pivot(PivotConfig{
		Index:   []string{"region"},
		Columns: "product",
		Values:  "sales",
	}); err != nil {
		t.Fatalf("Pivot failed: %v", err)
	}
	if dt.NumRows() != origRows {
		t.Errorf("source row count changed: %d -> %d", origRows, dt.NumRows())
	}
	if !reflect.DeepEqual(dt.ColNames(), origHeaders) {
		t.Errorf("source headers changed: %v -> %v", origHeaders, dt.ColNames())
	}
}

// ---------------- Unpivot ----------------

func unpivotWideTable() *DataTable {
	id := NewDataList(1, 2).SetName("id")
	q1 := NewDataList(5, 6).SetName("Q1")
	q2 := NewDataList(4, 7).SetName("Q2")
	q3 := NewDataList(3, 8).SetName("Q3")
	t := NewDataTable()
	t.AppendCols(id, q1, q2, q3)
	return t
}

func TestUnpivot_ExplicitValueVars(t *testing.T) {
	dt := unpivotWideTable()
	long, err := dt.Unpivot(UnpivotConfig{
		IDVars:    []string{"id"},
		ValueVars: []string{"Q1", "Q2", "Q3"},
		VarName:   "question",
		ValueName: "score",
	})
	if err != nil {
		t.Fatalf("Unpivot failed: %v", err)
	}
	if got := long.ColNames(); !reflect.DeepEqual(got, []string{"id", "question", "score"}) {
		t.Fatalf("headers = %v", got)
	}
	if long.NumRows() != 6 {
		t.Fatalf("rows = %d, want 6", long.NumRows())
	}
	ids := long.GetColByName("id").Data()
	questions := long.GetColByName("question").Data()
	scores := long.GetColByName("score").Data()
	wantIDs := []any{1, 1, 1, 2, 2, 2}
	wantQ := []any{"Q1", "Q2", "Q3", "Q1", "Q2", "Q3"}
	wantS := []any{5, 4, 3, 6, 7, 8}
	if !reflect.DeepEqual(ids, wantIDs) {
		t.Errorf("ids = %v, want %v", ids, wantIDs)
	}
	if !reflect.DeepEqual(questions, wantQ) {
		t.Errorf("questions = %v, want %v", questions, wantQ)
	}
	if !reflect.DeepEqual(scores, wantS) {
		t.Errorf("scores = %v, want %v", scores, wantS)
	}
}

func TestUnpivot_DefaultValueVars(t *testing.T) {
	dt := unpivotWideTable()
	long, err := dt.Unpivot(UnpivotConfig{
		IDVars: []string{"id"},
	})
	if err != nil {
		t.Fatalf("Unpivot failed: %v", err)
	}
	if long.NumRows() != 6 {
		t.Errorf("rows = %d, want 6", long.NumRows())
	}
	if got := long.ColNames(); !reflect.DeepEqual(got, []string{"id", "variable", "value"}) {
		t.Errorf("default var/value names = %v", got)
	}
}

func TestUnpivot_DropNA(t *testing.T) {
	id := NewDataList(1, 2, 3).SetName("id")
	q1 := NewDataList(5, nil, 7).SetName("Q1")
	q2 := NewDataList(nil, math.NaN(), 8).SetName("Q2")
	dt := NewDataTable()
	dt.AppendCols(id, q1, q2)

	long, err := dt.Unpivot(UnpivotConfig{
		IDVars: []string{"id"},
		DropNA: true,
	})
	if err != nil {
		t.Fatalf("Unpivot failed: %v", err)
	}
	// Original is 3 rows × 2 value cols = 6, minus 3 nil/NaN = 3.
	if long.NumRows() != 3 {
		t.Errorf("rows = %d, want 3 after DropNA", long.NumRows())
	}
}

func TestUnpivot_DropNAFalse_KeepsNils(t *testing.T) {
	id := NewDataList(1, 2).SetName("id")
	q1 := NewDataList(5, nil).SetName("Q1")
	dt := NewDataTable()
	dt.AppendCols(id, q1)

	long, err := dt.Unpivot(UnpivotConfig{
		IDVars: []string{"id"},
		DropNA: false,
	})
	if err != nil {
		t.Fatalf("Unpivot failed: %v", err)
	}
	if long.NumRows() != 2 {
		t.Errorf("rows = %d, want 2", long.NumRows())
	}
}

func TestUnpivot_MixedTypes(t *testing.T) {
	id := NewDataList(1).SetName("id")
	a := NewDataList("text").SetName("A")
	b := NewDataList(42).SetName("B")
	c := NewDataList(3.14).SetName("C")
	dt := NewDataTable()
	dt.AppendCols(id, a, b, c)

	long, err := dt.Unpivot(UnpivotConfig{
		IDVars: []string{"id"},
	})
	if err != nil {
		t.Fatalf("Unpivot failed: %v", err)
	}
	if long.NumRows() != 3 {
		t.Fatalf("rows = %d, want 3", long.NumRows())
	}
	values := long.GetColByName("value").Data()
	want := []any{"text", 42, 3.14}
	if !reflect.DeepEqual(values, want) {
		t.Errorf("mixed value column = %v, want %v", values, want)
	}
}

func TestUnpivot_EmptyValueVars(t *testing.T) {
	id := NewDataList(1, 2).SetName("id")
	dt := NewDataTable()
	dt.AppendCols(id)

	long, err := dt.Unpivot(UnpivotConfig{
		IDVars: []string{"id"},
	})
	if err != nil {
		t.Fatalf("Unpivot with no value cols failed: %v", err)
	}
	if long.NumRows() != 0 {
		t.Errorf("rows = %d, want 0", long.NumRows())
	}
	if got := long.ColNames(); !reflect.DeepEqual(got, []string{"id", "variable", "value"}) {
		t.Errorf("schema = %v", got)
	}
}

func TestUnpivot_MissingColumnError(t *testing.T) {
	dt := unpivotWideTable()
	if _, err := dt.Unpivot(UnpivotConfig{IDVars: []string{"nope"}}); err == nil {
		t.Errorf("expected error for missing IDVars column")
	}
	if _, err := dt.Unpivot(UnpivotConfig{IDVars: []string{"id"}, ValueVars: []string{"Q9"}}); err == nil {
		t.Errorf("expected error for missing ValueVars column")
	}
}

func TestUnpivot_VarValueNameClashError(t *testing.T) {
	dt := unpivotWideTable()
	if _, err := dt.Unpivot(UnpivotConfig{
		IDVars:    []string{"id"},
		VarName:   "x",
		ValueName: "x",
	}); err == nil {
		t.Errorf("expected error when VarName == ValueName")
	}
}

func TestUnpivot_OverlapError(t *testing.T) {
	dt := unpivotWideTable()
	if _, err := dt.Unpivot(UnpivotConfig{
		IDVars:    []string{"id", "Q1"},
		ValueVars: []string{"Q1", "Q2"},
	}); err == nil {
		t.Errorf("expected error when IDVars and ValueVars overlap")
	}
}

func TestUnpivot_DoesNotMutateSource(t *testing.T) {
	dt := unpivotWideTable()
	origRows := dt.NumRows()
	origHeaders := append([]string(nil), dt.ColNames()...)
	if _, err := dt.Unpivot(UnpivotConfig{IDVars: []string{"id"}}); err != nil {
		t.Fatalf("Unpivot failed: %v", err)
	}
	if dt.NumRows() != origRows {
		t.Errorf("source row count changed: %d -> %d", origRows, dt.NumRows())
	}
	if !reflect.DeepEqual(dt.ColNames(), origHeaders) {
		t.Errorf("source headers changed: %v -> %v", origHeaders, dt.ColNames())
	}
}

func TestPivotUnpivot_RoundTrip(t *testing.T) {
	// Wide -> Unpivot -> Pivot back -> compare to original
	id := NewDataList(1, 2).SetName("id")
	a := NewDataList(10, 30).SetName("A")
	b := NewDataList(20, 40).SetName("B")
	wide := NewDataTable()
	wide.AppendCols(id, a, b)

	long, err := wide.Unpivot(UnpivotConfig{
		IDVars:    []string{"id"},
		VarName:   "var",
		ValueName: "val",
	})
	if err != nil {
		t.Fatalf("Unpivot failed: %v", err)
	}
	if long.NumRows() != 4 {
		t.Fatalf("long rows = %d, want 4", long.NumRows())
	}
	round, err := long.Pivot(PivotConfig{
		Index:    []string{"id"},
		Columns:  "var",
		Values:   "val",
		SortCols: true,
	})
	if err != nil {
		t.Fatalf("Pivot back failed: %v", err)
	}
	if got := round.ColNames(); !reflect.DeepEqual(got, []string{"id", "A", "B"}) {
		t.Errorf("round-trip headers = %v", got)
	}
	if round.NumRows() != 2 {
		t.Errorf("round-trip rows = %d, want 2", round.NumRows())
	}
	colA := round.GetColByName("A").Data()
	colB := round.GetColByName("B").Data()
	if got, _ := ToFloat64Safe(colA[0]); got != 10 {
		t.Errorf("round-trip A[0] = %v, want 10", colA[0])
	}
	if got, _ := ToFloat64Safe(colA[1]); got != 30 {
		t.Errorf("round-trip A[1] = %v, want 30", colA[1])
	}
	if got, _ := ToFloat64Safe(colB[0]); got != 20 {
		t.Errorf("round-trip B[0] = %v, want 20", colB[0])
	}
	if got, _ := ToFloat64Safe(colB[1]); got != 40 {
		t.Errorf("round-trip B[1] = %v, want 40", colB[1])
	}
}
