package py

import (
	"context"
	"slices"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// frame is the payload insyra.Return sends for a DataFrame with columns a and
// b and the rows (1, 2) and (3, 4).
const frame = `{"_insyra_type":"datatable","data":[[1,2],[3,4]],"columns":["a","b"],"index":["0","1"]}`

// series is the payload insyra.Return sends for a Series named s.
const series = `{"_insyra_type":"datalist","data":[1,2,3],"name":"s"}`

func checkFrame(t *testing.T, what string, dt *insyra.DataTable) {
	t.Helper()
	if dt == nil {
		t.Fatalf("%s: no table", what)
	}
	if rows, cols := dt.Size(); rows != 2 || cols != 2 {
		t.Errorf("%s: the table is %d x %d, want 2 x 2", what, rows, cols)
	}
	if got := dt.ColNames(); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("%s: the columns are %q, want a and b", what, got)
	}
}

// The review of py-typed-run found that a field named DataTable was taken for
// an isr wrapper, so the whole answer went into it and the struct's other
// fields stayed zero; JSON decoding alone would have left the field an empty
// table.
func TestAStructHoldingATableDecodesEveryField(t *testing.T) {
	useFakePython(t, `result:{"table":`+frame+`,"score":7}`)
	type named struct {
		DataTable *insyra.DataTable `json:"table"`
		Score     float64           `json:"score"`
	}
	got, err := Run[named](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "DataTable field", got.DataTable)
	if got.Score != 7 {
		t.Errorf("Score is %v, want 7", got.Score)
	}

	type other struct {
		Table insyra.IDataTable `json:"table"`
		Score int
	}
	got2, err := Run[other](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	dt, ok := got2.Table.(*insyra.DataTable)
	if !ok {
		t.Fatalf("the IDataTable field holds %T", got2.Table)
	}
	checkFrame(t, "IDataTable field", dt)
	if got2.Score != 7 {
		t.Errorf("Score, matched without regard to case, is %v, want 7", got2.Score)
	}
}

func TestAStructHoldingAListDecodesEveryField(t *testing.T) {
	useFakePython(t, `result:{"DataList":`+series+`,"label":"x"}`)
	type withList struct {
		DataList *insyra.DataList
		Label    string `json:"label"`
	}
	got, err := Run[withList](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	if got.DataList == nil || got.DataList.Len() != 3 || got.DataList.GetName() != "s" {
		t.Errorf("the list field is %v", got.DataList)
	}
	if got.Label != "x" {
		t.Errorf("Label is %q, want x", got.Label)
	}
}

func TestTablesInAMapAndListsInASlice(t *testing.T) {
	useFakePython(t, `result:{"first":`+frame+`,"second":`+frame+`}`)
	tables, err := Run[map[string]*insyra.DataTable](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 2 {
		t.Fatalf("got %d tables, want 2", len(tables))
	}
	for name, dt := range tables {
		checkFrame(t, name, dt)
	}

	t.Setenv(fakePythonEnv, `result:[`+series+`,`+series+`]`)
	lists, err := Run[[]*insyra.DataList](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	if len(lists) != 2 || lists[0].Len() != 3 || lists[1].GetName() != "s" {
		t.Errorf("the lists are %v", lists)
	}
}

// Fields of an embedded struct are promoted, as encoding/json promotes them,
// and a field tagged "-" is left alone.
func TestAnEmbeddedStructsTableFieldIsPromoted(t *testing.T) {
	useFakePython(t, `result:{"table":`+frame+`,"score":7,"Skipped":1}`)
	type base struct {
		Table *insyra.DataTable `json:"table"`
	}
	type withBase struct {
		base
		Score   float64 `json:"score"`
		Skipped int     `json:"-"`
	}
	got, err := Run[withBase](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "promoted field", got.Table)
	if got.Score != 7 {
		t.Errorf("Score is %v, want 7", got.Score)
	}
	if got.Skipped != 0 {
		t.Errorf("a field tagged - was set to %d", got.Skipped)
	}
}

// isr's types embed the list or table they wrap, and that is the one case the
// whole result goes into the field.
func TestAnEmbeddedTableStillTakesTheWholeResult(t *testing.T) {
	useFakePython(t, `result:`+frame)
	type wrapper struct {
		*insyra.DataTable
	}
	var w wrapper
	if err := RunCode(&w, "insyra.Return(df)"); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "embedded table", w.DataTable)
}

func TestAStructWithoutTablesStillDecodesThroughJSON(t *testing.T) {
	useFakePython(t, `result:{"name":"insyra","tags":["a","b"],"nested":{"n":2}}`)
	type inner struct {
		N int `json:"n"`
	}
	type plain struct {
		Name   string   `json:"name"`
		Tags   []string `json:"tags"`
		Nested inner    `json:"nested"`
	}
	got, err := Run[plain](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "insyra" || !slices.Equal(got.Tags, []string{"a", "b"}) || got.Nested.N != 2 {
		t.Errorf("decoded %+v", got)
	}
}

// selfDecoding decodes itself, so its own UnmarshalJSON must see the JSON
// even though it holds a table.
type selfDecoding struct {
	Table *insyra.DataTable
	raw   string
}

func (s *selfDecoding) UnmarshalJSON(b []byte) error {
	s.raw = string(b)
	return nil
}

func TestATypeWithItsOwnUnmarshalJSONStillGetsTheJSON(t *testing.T) {
	useFakePython(t, `result:{"Table":`+frame+`}`)
	got, err := Run[selfDecoding](context.Background(), "insyra.Return(x)")
	if err != nil {
		t.Fatal(err)
	}
	if got.raw == "" {
		t.Error("the type's own UnmarshalJSON was not called")
	}
	if got.Table != nil {
		t.Error("the table was decoded around the type's own UnmarshalJSON")
	}
}

// A DataFrame a filter left empty came back with its columns renamed a_1 and
// b_1, because the empty path named them and SetColNames then named them
// again.
func TestAnEmptyDataFrameKeepsItsColumnNames(t *testing.T) {
	useFakePython(t, `result:{"_insyra_type":"datatable","data":[],"columns":["a","b"],"index":[]}`)
	dt, err := Run[*insyra.DataTable](context.Background(), "insyra.Return(df)")
	if err != nil {
		t.Fatal(err)
	}
	if rows, cols := dt.Size(); rows != 0 || cols != 2 {
		t.Errorf("the table is %d x %d, want 0 x 2", rows, cols)
	}
	if got := dt.ColNames(); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("the columns are %q, want a and b", got)
	}
}
