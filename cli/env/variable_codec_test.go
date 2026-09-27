package env

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	insyra "github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// roundTripVariable encodes v, writes the variable as JSON, reads it back
// through a decoder that keeps numbers as json.Number, and decodes the variable
// again — the path a stored variable takes between two sessions. It hands back
// the stored form as well, so a test can check the shape the file has and not
// only the value it restores.
func roundTripVariable(t *testing.T, v any) (SerializedVariable, any) {
	t.Helper()
	sv, err := encodeVariable(v)
	if err != nil {
		t.Fatalf("encodeVariable(%T) returned %v", v, err)
	}
	raw, err := json.Marshal(sv)
	if err != nil {
		t.Fatalf("json.Marshal of the encoded %T: %v", v, err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var read SerializedVariable
	if err := dec.Decode(&read); err != nil {
		t.Fatalf("decoding %s: %v", raw, err)
	}
	got, ok, err := decodeVariable(read)
	if !ok {
		t.Fatalf("decodeVariable(%s) reported the type %q as one it does not read", raw, read.Type)
	}
	if err != nil {
		t.Fatalf("decodeVariable(%s) returned %v", raw, err)
	}
	return sv, got
}

// sameCell compares two cells the way each one asks to be compared: NaN by
// being NaN, a time by the instant it names, a byte slice by content.
func sameCell(got, want any) bool {
	switch expected := want.(type) {
	case float64:
		f, isFloat := got.(float64)
		return isFloat && sameSpecialFloat64(f, expected)
	case time.Time:
		t, isTime := got.(time.Time)
		return isTime && t.Equal(expected)
	case []byte:
		b, isBytes := got.([]byte)
		return isBytes && bytes.Equal(b, expected)
	default:
		return reflect.DeepEqual(got, want)
	}
}

// asTable asserts that v is a restored table and hands it back.
func asTable(t *testing.T, v any) *insyra.DataTable {
	t.Helper()
	table, isTable := v.(*insyra.DataTable)
	if !isTable {
		t.Fatalf("restored value is %T, want *insyra.DataTable", v)
	}
	return table
}

// sameBits compares two tables cell by cell, every float64 by its bit pattern:
// a restored scaler that lands one ULP off the original still scales to values
// the caller must not be handed.
func sameBits(t *testing.T, got, want *insyra.DataTable) {
	t.Helper()
	if got.NumCols() != want.NumCols() {
		t.Fatalf("transformed %d columns, want %d", got.NumCols(), want.NumCols())
	}
	if got.NumRows() != want.NumRows() {
		t.Fatalf("transformed %d rows, want %d", got.NumRows(), want.NumRows())
	}
	for i := 0; i < want.NumCols(); i++ {
		gotCells, wantCells := got.GetColByNumber(i).Data(), want.GetColByNumber(i).Data()
		if len(gotCells) != len(wantCells) {
			t.Fatalf("column %d has %d cells, want %d", i, len(gotCells), len(wantCells))
		}
		for j, wantCell := range wantCells {
			// A cell the scaler never read stays as it was, nil included, so a
			// filled-in zero here is as wrong as a value one ULP off.
			if wantCell == nil {
				if gotCells[j] != nil {
					t.Errorf("column %d cell %d = %#v, want nil", i, j, gotCells[j])
				}
				continue
			}
			gotValue, gotIsFloat := gotCells[j].(float64)
			wantValue, wantIsFloat := wantCell.(float64)
			if !gotIsFloat || !wantIsFloat {
				t.Fatalf("column %d cell %d is %#v (%T), want a float64", i, j, gotCells[j], gotCells[j])
			}
			if math.Float64bits(gotValue) != math.Float64bits(wantValue) {
				t.Errorf("column %d cell %d = %#v (%#016x), want %#v (%#016x)",
					i, j, gotValue, math.Float64bits(gotValue), wantValue, math.Float64bits(wantValue))
			}
		}
	}
}

func TestVariableCodec_RoundTripsDataTable(t *testing.T) {
	names := []string{"zeta", "alpha", "when", "mix"}
	want := [][]any{
		{3.0, 1.5, math.NaN()},
		{"x", "y", nil},
		{
			time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
			time.Date(2025, 6, 7, 8, 9, 10, 0, time.UTC),
			time.Date(2026, 11, 12, 13, 14, 15, 0, time.UTC),
		},
		{int64(1), "a", true},
	}
	table := insyra.NewDataTable()
	for i, cells := range want {
		table.AppendCols(insyra.NewDataList().Append(cells...).SetName(names[i]))
	}
	table.SetRowNames([]string{"r1", "", "r3"})
	table.SetName("T")

	sv, got := roundTripVariable(t, table)
	if sv.Type != "table" {
		t.Errorf("stored type = %q, want %q", sv.Type, "table")
	}
	restored := asTable(t, got)
	if restored.NumCols() != len(want) {
		t.Fatalf("restored %d columns, want %d", restored.NumCols(), len(want))
	}
	for i, colName := range names {
		col := restored.GetColByNumber(i)
		if col.GetName() != colName {
			t.Errorf("column %d name = %q, want %q", i, col.GetName(), colName)
		}
		cells := col.Data()
		if len(cells) != len(want[i]) {
			t.Fatalf("column %q has %d cells, want %d", colName, len(cells), len(want[i]))
		}
		for j, cell := range cells {
			// The type is the part a plain == cannot see: a 3.0 stored as a 3
			// is an integer to everything that reads it back.
			if got, wantCell := fmt.Sprintf("%T", cell), fmt.Sprintf("%T", want[i][j]); got != wantCell {
				t.Errorf("column %q cell %d type = %s, want %s", colName, j, got, wantCell)
				continue
			}
			if !sameCell(cell, want[i][j]) {
				t.Errorf("column %q cell %d = %#v, want %#v", colName, j, cell, want[i][j])
			}
		}
	}
	if gotNames := restored.RowNames(); !reflect.DeepEqual(gotNames, []string{"r1", "", "r3"}) {
		t.Errorf("restored row names = %#v, want [\"r1\" \"\" \"r3\"]", gotNames)
	}
	if restored.GetName() != "T" {
		t.Errorf("restored table name = %q, want %q", restored.GetName(), "T")
	}
}

func TestVariableCodec_TableWithoutRowNamesStoresNone(t *testing.T) {
	// A table whose rows are all unnamed writes no array, so the file says
	// nothing about row names and reading it back adds none.
	table := insyra.NewDataTable()
	table.AppendCols(insyra.NewDataList().Append(1.0, 2.0).SetName("a"))

	sv, got := roundTripVariable(t, table)
	raw, err := json.Marshal(sv)
	if err != nil {
		t.Fatalf("json.Marshal of the encoded table: %v", err)
	}
	if bytes.Contains(raw, []byte("rowNames")) {
		t.Errorf("stored variable = %s, want no \"rowNames\" for a table with no row names", raw)
	}
	for i, name := range asTable(t, got).RowNames() {
		if name != "" {
			t.Errorf("restored row name %d = %q, want %q", i, name, "")
		}
	}
}

func TestVariableCodec_RoundTripsDataList(t *testing.T) {
	// A []byte cell is one cell, not two: the list is built with Append, which
	// does not flatten it the way the constructor does.
	want := []any{
		int(1), int64(2), 3.0, true, "s", nil, math.NaN(),
		time.Date(2024, 5, 6, 0, 0, 0, 0, time.UTC), []byte{1, 2},
	}
	list := insyra.NewDataList().Append(want...).SetName("L")

	sv, got := roundTripVariable(t, list)
	if sv.Type != "list" {
		t.Errorf("stored type = %q, want %q", sv.Type, "list")
	}
	restored, isList := got.(*insyra.DataList)
	if !isList {
		t.Fatalf("restored value is %T, want *insyra.DataList", got)
	}
	if restored.Len() != len(want) {
		t.Fatalf("restored %d cells, want %d", restored.Len(), len(want))
	}
	for i, cell := range restored.Data() {
		if got, wantCell := fmt.Sprintf("%T", cell), fmt.Sprintf("%T", want[i]); got != wantCell {
			t.Errorf("cell %d type = %s, want %s", i, got, wantCell)
			continue
		}
		if !sameCell(cell, want[i]) {
			t.Errorf("cell %d = %#v, want %#v", i, cell, want[i])
		}
	}
	if restored.GetName() != "L" {
		t.Errorf("restored list name = %q, want %q", restored.GetName(), "L")
	}
}

func TestVariableCodec_RoundTripsScalars(t *testing.T) {
	// 3.0 and 7 are the two that matter: the file stores both as the number
	// they print as, so only the tag can tell them apart again.
	for _, want := range []any{
		1.25,
		3.0,
		7,
		int64(9007199254740993),
		true,
		"hi",
		time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	} {
		sv, got := roundTripVariable(t, want)
		if sv.Type != "scalar" {
			t.Errorf("stored type of %#v = %q, want %q", want, sv.Type, "scalar")
		}
		if gotType, wantType := fmt.Sprintf("%T", got), fmt.Sprintf("%T", want); gotType != wantType {
			t.Errorf("%#v round trip type = %s, want %s", want, gotType, wantType)
			continue
		}
		if !sameCell(got, want) {
			t.Errorf("%#v round trip = %#v", want, got)
		}
	}
}

func TestVariableCodec_NilRoundTripsAsNil(t *testing.T) {
	sv, got := roundTripVariable(t, nil)
	if sv.Type != "scalar" {
		t.Errorf("stored type = %q, want %q", sv.Type, "scalar")
	}
	if got != nil {
		t.Errorf("restored value = %#v (%T), want nil", got, got)
	}
}

func TestVariableCodec_RoundTripsSlices(t *testing.T) {
	for _, want := range []any{
		[]int{3, 4},
		[]float64{1.5, math.NaN()},
		[]bool{true, false},
		[]string{},
		[]time.Time{time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)},
	} {
		sv, got := roundTripVariable(t, want)
		if sv.Type != "slice" {
			t.Errorf("stored type of %T = %q, want %q", want, sv.Type, "slice")
		}
		wantValue, gotValue := reflect.ValueOf(want), reflect.ValueOf(got)
		if gotValue.Type() != wantValue.Type() {
			t.Errorf("%T round trip type = %s, want %s", want, gotValue.Type(), wantValue.Type())
			continue
		}
		if gotValue.Len() != wantValue.Len() {
			t.Errorf("%T round trip has %d elements, want %d", want, gotValue.Len(), wantValue.Len())
			continue
		}
		for i := 0; i < wantValue.Len(); i++ {
			if !sameCell(gotValue.Index(i).Interface(), wantValue.Index(i).Interface()) {
				t.Errorf("%T element %d = %#v, want %#v", want, i, gotValue.Index(i).Interface(), wantValue.Index(i).Interface())
			}
		}
	}
}

// The element types a slice may have are a closed list, and the file names one
// of them: a type missing from the map is a slice the environment cannot hold,
// and a type added to it is one it can.
func TestVariableCodec_SliceElementTypesAreTheOnesTheFileNames(t *testing.T) {
	want := []string{
		"bool", "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64",
		"time.Time", "time.Duration", "decimal.Decimal", "json.Number", "[]uint8",
	}
	if len(sliceElemTypes) != len(want) {
		t.Errorf("sliceElemTypes holds %d types (%v), want %d", len(sliceElemTypes), sliceElemTypes, len(want))
	}
	for _, tag := range want {
		if sliceElemTypes[tag] == nil {
			t.Errorf("sliceElemTypes has no entry for %q", tag)
		}
	}
}

func TestVariableCodec_RoundTripsASliceOfByteSlices(t *testing.T) {
	// []uint8 is in the element table for this case: a []byte on its own is one
	// value and the cell branch claims it first, so a slice of them is the only
	// thing that reads that entry.
	want := [][]byte{{1, 2}, {}}
	sv, got := roundTripVariable(t, want)
	if sv.Type != "slice" {
		t.Errorf("stored type = %q, want %q", sv.Type, "slice")
	}
	gotValue, wantValue := reflect.ValueOf(got), reflect.ValueOf(want)
	if gotValue.Type() != wantValue.Type() {
		t.Fatalf("restored type = %s, want %s", gotValue.Type(), wantValue.Type())
	}
	for i := 0; i < wantValue.Len(); i++ {
		if !sameCell(gotValue.Index(i).Interface(), wantValue.Index(i).Interface()) {
			t.Errorf("element %d = %#v, want %#v", i, gotValue.Index(i).Interface(), wantValue.Index(i).Interface())
		}
	}
}

func TestVariableCodec_TypedNilIsStoredAsAScalar(t *testing.T) {
	// A nil *DataTable is not a table with no columns, and encoding it as one
	// would restore an empty table where the caller had nothing at all.
	for _, v := range []any{(*insyra.DataTable)(nil), (*insyra.DataList)(nil)} {
		sv, got := roundTripVariable(t, v)
		if sv.Type != "scalar" {
			t.Errorf("stored type of %T = %q, want %q", v, sv.Type, "scalar")
		}
		if got != nil {
			t.Errorf("%T round trip = %#v (%T), want nil", v, got, got)
		}
	}
}

func TestVariableCodec_RejectsValuesItCannotStore(t *testing.T) {
	for _, v := range []any{
		struct{ X int }{1},
		[]struct{ X int }{{1}},
		map[int]int{},
	} {
		if _, err := encodeVariable(v); err == nil {
			t.Errorf("encodeVariable(%T) returned no error", v)
		}
	}
	// The message has to name the type it refused, so the user knows which value
	// to look at.
	_, err := encodeVariable(struct{ X int }{1})
	if err == nil {
		t.Fatal("encodeVariable(struct{ X int }{1}) returned no error")
	}
	if !strings.Contains(err.Error(), "struct { X int }") {
		t.Errorf("encodeVariable error = %q, want it to name the type it refused", err.Error())
	}
}

// A cell the environment cannot store is reported by the column and the row it
// sits in, because that is the cell the user has to look at.
func TestVariableCodec_RejectsACellItCannotStore(t *testing.T) {
	table := insyra.NewDataTable()
	table.AppendCols(insyra.NewDataList().Append(1.0, struct{ X int }{1}, 3.0).SetName("payload"))
	table.AppendCols(insyra.NewDataList().Append("a", "b", "c").SetName("other"))

	_, err := encodeVariable(table)
	if err == nil {
		t.Fatal("encodeVariable accepted a table holding a cell it cannot store")
	}
	if message := err.Error(); !strings.Contains(message, `column "payload"`) || !strings.Contains(message, "row 1") {
		t.Errorf("encodeVariable error = %q, want it to name column \"payload\" and row 1", message)
	}
}

// The kinds earlier releases wrote are not this codec's business: the caller
// has to be able to tell them apart and hand them to the legacy reader.
func TestVariableCodec_DecodeLeavesEarlierKindsToTheLegacyReader(t *testing.T) {
	for _, kind := range []string{"DataTable", "DataList", "Raw"} {
		got, ok, err := decodeVariable(SerializedVariable{Type: kind, Data: "x"})
		if ok || err != nil || got != nil {
			t.Errorf("decodeVariable(%q) = (%#v, %v, %v), want (nil, false, nil)", kind, got, ok, err)
		}
	}
}

// A kind the codec owns but a body it cannot read is an error the caller has to
// see. Reporting ok == false would send it to a reader that cannot read it
// either, and the variable would come back as the raw JSON.
func TestVariableCodec_DecodeRejectsUnreadableData(t *testing.T) {
	cases := []struct {
		name string
		sv   SerializedVariable
	}{
		{"a table that is not an object", SerializedVariable{Type: "table", Data: "garbage"}},
		{"a table with no columns", SerializedVariable{Type: "table", Data: map[string]any{}}},
		{"columns that are not an array", SerializedVariable{Type: "table", Data: map[string]any{"columns": "zeta"}}},
		{"a column that is not an object", SerializedVariable{Type: "table", Data: map[string]any{"columns": []any{"zeta"}}}},
		{
			"rowNames that are not an array",
			SerializedVariable{Type: "table", Data: map[string]any{
				"columns": []any{map[string]any{"values": []any{json.Number("1")}}}, "rowNames": "r1",
			}},
		},
		{
			"a rowName that is not a string",
			SerializedVariable{Type: "table", Data: map[string]any{
				"columns":  []any{map[string]any{"values": []any{json.Number("1")}}},
				"rowNames": []any{json.Number("1")},
			}},
		},
		{"a list that is not an object", SerializedVariable{Type: "list", Data: "garbage"}},
		{"a scalar that is not an object", SerializedVariable{Type: "scalar", Data: "garbage"}},
		{
			"a scalar type that is not a string",
			SerializedVariable{Type: "scalar", Data: map[string]any{"type": json.Number("1"), "value": json.Number("1")}},
		},
		{"a slice that is not an object", SerializedVariable{Type: "slice", Data: "garbage"}},
		{
			"a slice with no element type",
			SerializedVariable{Type: "slice", Data: map[string]any{"values": []any{}}},
		},
		{
			"an element type no cell can be restored from",
			SerializedVariable{Type: "slice", Data: map[string]any{"elem": "struct { X int }", "values": []any{}}},
		},
		{
			"slice values that are not an array",
			SerializedVariable{Type: "slice", Data: map[string]any{"elem": "int", "values": "none"}},
		},
		// A null element has no type to restore it as, and the slice the
		// caller asked for holds no nil to put there.
		{
			"a null element in a slice",
			SerializedVariable{Type: "slice", Data: map[string]any{
				"elem": "int", "values": []any{json.Number("1"), nil},
			}},
		},
		{
			"a slice element that does not match its type",
			SerializedVariable{Type: "slice", Data: map[string]any{
				"elem": "int", "values": []any{"1"},
			}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok, err := decodeVariable(c.sv)
			if !ok {
				t.Errorf("decodeVariable(%#v) reported the type %q as one it does not read", c.sv, c.sv.Type)
			}
			if err == nil {
				t.Errorf("decodeVariable(%#v) = (%#v, %v), want an error", c.sv, got, ok)
			}
		})
	}
}

// A fitted scaler is worth keeping only if the scaler that comes back scales
// exactly like the one that went in, so the check is on the numbers Transform
// produces and not only on the parameters it carries.
func TestVariableCodec_RoundTripsAFittedScaler(t *testing.T) {
	// A nil cell has to survive the round trip too: the columns are fitted from
	// what is there, and Transform passes a nil through rather than filling it
	// in, so a restored scaler that zeroed it would not scale alike.
	fitted := insyra.NewDataTable(
		insyra.NewDataList(1.5, nil, 3.0, 10.0).SetName("a"),
		insyra.NewDataList(4, 8, 15, 16).SetName("b"),
	)
	apply := insyra.NewDataTable(
		insyra.NewDataList(2.0, 7.25, nil).SetName("a"),
		insyra.NewDataList(0, 3, 99).SetName("b"),
	)
	cases := []struct {
		kind string
		new  func() insyra.Scaler
	}{
		{"standard", func() insyra.Scaler { return insyra.NewStandardScaler() }},
		{"minmax", func() insyra.Scaler { return insyra.NewMinMaxScaler(-1, 1) }},
		{"robust", func() insyra.Scaler { return insyra.NewRobustScaler() }},
		{"maxabs", func() insyra.Scaler { return insyra.NewMaxAbsScaler() }},
	}
	for _, c := range cases {
		t.Run(c.kind, func(t *testing.T) {
			orig := c.new()
			// One column by name and one by its Excel index, because the ref is
			// part of what a restored scaler has to resolve the same way.
			if err := orig.Fit(fitted, "a", "B"); err != nil {
				t.Fatalf("Fit: %v", err)
			}
			sv, got := roundTripVariable(t, orig)
			if sv.Type != "scaler" {
				t.Errorf("stored type = %q, want %q", sv.Type, "scaler")
			}
			restored, isScaler := got.(insyra.Scaler)
			if !isScaler {
				t.Fatalf("restored value is %T, want a %T", got, orig)
			}
			if gotType, wantType := reflect.TypeOf(restored), reflect.TypeOf(orig); gotType != wantType {
				t.Fatalf("restored type = %s, want %s", gotType, wantType)
			}
			if !reflect.DeepEqual(restored.Params(), orig.Params()) {
				t.Errorf("restored params = %#v, want %#v", restored.Params(), orig.Params())
			}
			want, err := orig.Transform(apply)
			if err != nil {
				t.Fatalf("Transform of the original: %v", err)
			}
			have, err := restored.Transform(apply)
			if err != nil {
				t.Fatalf("Transform of the restored scaler: %v", err)
			}
			sameBits(t, have, want)
		})
	}
}

// The tree a hierarchical fit produced is a fitted result too: it has to come
// back as a *stats.HierarchicalResult, because that is what CutTreeByK takes.
func TestVariableCodec_RoundTripsAHierarchicalTree(t *testing.T) {
	orig, err := stats.HierarchicalAgglomerative(
		insyra.NewDataTable(
			insyra.NewDataList(1.0, 2.0, 8.0, 9.0).SetName("x"),
			insyra.NewDataList(2.0, 3.0, 9.0, 9.0).SetName("y"),
		),
		stats.AgglomerativeMethod("complete"),
	)
	if err != nil {
		t.Fatalf("HierarchicalAgglomerative: %v", err)
	}

	sv, got := roundTripVariable(t, orig)
	if sv.Type != "hclust" {
		t.Errorf("stored type = %q, want %q", sv.Type, "hclust")
	}
	restored, isTree := got.(*stats.HierarchicalResult)
	if !isTree {
		t.Fatalf("restored value is %T, want *stats.HierarchicalResult", got)
	}
	if !reflect.DeepEqual(restored, orig) {
		t.Errorf("restored tree = %#v, want %#v", restored, orig)
	}
	want, err := stats.CutTreeByK(orig, 2)
	if err != nil {
		t.Fatalf("CutTreeByK of the original: %v", err)
	}
	have, err := stats.CutTreeByK(restored, 2)
	if err != nil {
		t.Fatalf("CutTreeByK of the restored tree: %v", err)
	}
	if !reflect.DeepEqual(have, want) {
		t.Errorf("CutTreeByK(restored, 2) = %#v, want %#v", have, want)
	}
}

// A nil scaler or tree is not a fitted one with no columns: storing it as a
// scaler would hand back an empty scaler where the caller had nothing at all.
func TestVariableCodec_TypedNilScalerAndTreeAreStoredAsScalars(t *testing.T) {
	for _, v := range []any{(*insyra.StandardScaler)(nil), (*stats.HierarchicalResult)(nil)} {
		sv, got := roundTripVariable(t, v)
		if sv.Type != "scalar" {
			t.Errorf("stored type of %T = %q, want %q", v, sv.Type, "scalar")
		}
		if got != nil {
			t.Errorf("%T round trip = %#v (%T), want nil", v, got, got)
		}
	}
}

// A kind tag the file carries for a different library is unreadable rather than
// absent: reporting ok == false would hand the variable to a reader that cannot
// read it either, and it would come back as the raw JSON.
func TestVariableCodec_DecodeRejectsAScalerKindItDoesNotKnow(t *testing.T) {
	got, ok, err := decodeVariable(SerializedVariable{Type: "scaler", Data: map[string]any{"kind": "bogus"}})
	if !ok {
		t.Errorf("decodeVariable of a scaler reported the type %q as one it does not read", "scaler")
	}
	if err == nil {
		t.Fatalf("decodeVariable of a scaler of kind %q = (%#v, %v), want an error", "bogus", got, ok)
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("decodeVariable error = %q, want it to name the kind it refused", err.Error())
	}
}

// celsiusSlice is a named slice type. The file names a slice by its element
// type, so a named one would come back as []float64 and the type the caller
// holds would be gone.
type celsiusSlice []float64

func TestVariableCodec_RejectsASliceOfANamedType(t *testing.T) {
	_, err := encodeVariable(celsiusSlice{1.5})
	if err == nil {
		t.Fatal("encodeVariable(celsiusSlice{1.5}) returned no error")
	}
	if !strings.Contains(err.Error(), "celsiusSlice") {
		t.Errorf("encodeVariable error = %q, want it to name the type it refused", err.Error())
	}
	// The unnamed slice of the same element type is one the environment holds.
	sv, err := encodeVariable([]float64{1.5})
	if err != nil {
		t.Fatalf("encodeVariable([]float64{1.5}) returned %v", err)
	}
	if sv.Type != "slice" {
		t.Errorf("stored type of []float64 = %q, want %q", sv.Type, "slice")
	}
}

func TestVariableCodec_RoundTripsADurationSlice(t *testing.T) {
	want := []time.Duration{time.Second, -time.Hour}
	sv, got := roundTripVariable(t, want)
	if sv.Type != "slice" {
		t.Errorf("stored type = %q, want %q", sv.Type, "slice")
	}
	wantValue, gotValue := reflect.ValueOf(want), reflect.ValueOf(got)
	if gotValue.Type() != wantValue.Type() {
		t.Errorf("duration slice round trip type = %s, want %s", gotValue.Type(), wantValue.Type())
		return
	}
	if gotValue.Len() != wantValue.Len() {
		t.Errorf("duration slice round trip has %d elements, want %d", gotValue.Len(), wantValue.Len())
		return
	}
	for i := 0; i < wantValue.Len(); i++ {
		if gotValue.Index(i).Interface() != wantValue.Index(i).Interface() {
			t.Errorf("duration slice element %d = %v, want %v", i, gotValue.Index(i).Interface(), wantValue.Index(i).Interface())
		}
	}
}
