package py

import (
	stdjson "encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/isr"
	json "github.com/goccy/go-json"
)

// bindJSON decodes text the way the IPC handler does and binds it into out.
func bindJSON(t *testing.T, out any, text string) error {
	t.Helper()
	var result any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatal(err)
	}
	return bindPyResult(out, result)
}

// sameAsEncodingJSON binds text into a new T and checks that it equals what
// encoding/json gives for the same text. The text holds no table, so the two
// can be compared whole.
func sameAsEncodingJSON[T any](t *testing.T, what, text string) {
	t.Helper()
	var want, got T
	if err := stdjson.Unmarshal([]byte(text), &want); err != nil {
		t.Fatalf("%s: encoding/json: %v", what, err)
	}
	if err := bindJSON(t, &got, text); err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s: got %+v, encoding/json gives %+v", what, got, want)
	}
}

// The adversarial review of py-nested-table-results found that a struct
// embedding a pointer to itself, or two embedding each other, recursed until
// the stack ran out, which ends the program and no recover can catch.
type SelfNode struct {
	*SelfNode
	T *insyra.DataTable
}

type MutualA struct {
	*MutualB
	T *insyra.DataTable
}

type MutualB struct {
	*MutualA
	X int
}

func TestStructsThatEmbedThemselvesDecode(t *testing.T) {
	var n SelfNode
	if err := bindJSON(t, &n, `{"T":`+frame+`}`); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "a struct embedding itself", n.T)
	if n.SelfNode != nil {
		t.Error("an embedded pointer no key reached was allocated")
	}

	var a MutualA
	if err := bindJSON(t, &a, `{"X":1,"T":null}`); err != nil {
		t.Fatal(err)
	}
	if a.MutualB == nil || a.X != 1 {
		t.Errorf("the promoted field X came back as %+v", a.MutualB)
	}
}

// An isr wrapper takes the whole result. A result of the other kind used to
// leave it without its list or table and report nothing.
func TestAnISRWrapperGivenTheOtherKindOfResultIsAnError(t *testing.T) {
	l := isr.UseDL(insyra.NewDataList(9, 8))
	if err := bindJSON(t, &l, frame); err == nil {
		t.Error("a DataFrame bound into an isr list gave no error")
	}
	if l == nil || l.DataList == nil || l.Len() != 2 {
		t.Errorf("the failed bind changed the list to %v", l)
	}

	d := isr.UseDT(insyra.NewDataTable(insyra.NewDataList(1, 2)))
	if err := bindJSON(t, &d, series); err == nil {
		t.Error("a Series bound into an isr table gave no error")
	}
	if d == nil || d.DataTable == nil {
		t.Fatal("the failed bind dropped the table")
	}
	if rows, _ := d.Size(); rows != 2 {
		t.Errorf("the failed bind changed the table to %d rows", rows)
	}
}

// zeroOf returns the zero value of v's type, for a type this package cannot
// name.
func zeroOf[T any](T) T {
	var zero T
	return zero
}

func TestAnISRWrapperStillTakesTheWholeResult(t *testing.T) {
	dv := isr.DT
	if err := bindJSON(t, &dv, frame); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "isr.DT value", dv.DataTable)

	l := zeroOf(isr.UseDL(insyra.NewDataList()))
	if err := bindJSON(t, &l, series); err != nil {
		t.Fatal(err)
	}
	if l == nil || l.DataList == nil || l.Len() != 3 || l.GetName() != "s" {
		t.Errorf("a nil isr list pointer came back as %v", l)
	}
}

// A ",string" option used to be dropped once the type held a table anywhere,
// even in a field tagged "-" that is never decoded.
func TestQuotedFieldsBesideATable(t *testing.T) {
	type withID struct {
		ID    int64             `json:"id,string"`
		Table *insyra.DataTable `json:"table"`
	}
	var w withID
	if err := bindJSON(t, &w, `{"id":"12345678901234567","table":`+frame+`}`); err != nil {
		t.Fatal(err)
	}
	if w.ID != 12345678901234567 {
		t.Errorf("ID is %d, want 12345678901234567", w.ID)
	}
	checkFrame(t, "table beside a quoted field", w.Table)

	type skipped struct {
		ID    int64             `json:"id,string"`
		Cache *insyra.DataTable `json:"-"`
	}
	sameAsEncodingJSON[skipped](t, "a table tagged -", `{"id":"12345678901234567","Cache":null}`)
	sameAsEncodingJSON[withID](t, "a quoted null", `{"id":null}`)
}

// An embedded table follows encoding/json's rule for an embedded struct. With
// a tag it is a field under that name: it used to take the whole result, so a
// dict became a one-row table and the other fields stayed zero. Without a tag
// it is decoded from the whole value, as the isr types are, and the struct's
// other fields are decoded beside it.
func TestAnEmbeddedTableFollowsTheRuleForEmbeddedStructs(t *testing.T) {
	type tagged struct {
		*insyra.DataTable `json:"table"`
		Score             float64 `json:"score"`
	}
	var s tagged
	if err := bindJSON(t, &s, `{"table":`+frame+`,"score":7}`); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "an embedded table tagged table", s.DataTable)
	if s.Score != 7 {
		t.Errorf("Score is %v, want 7", s.Score)
	}

	type untagged struct {
		*insyra.DataTable
		Note string
	}
	var u untagged
	if err := bindJSON(t, &u, frame); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "an untagged embedded table beside a field", u.DataTable)

	type deeper struct {
		tableWrap
		Score float64 `json:"score"`
	}
	var d deeper
	if err := bindJSON(t, &d, frame); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "a table embedded one level down", d.DataTable)
}

// None decodes as JSON's null does: a pointer becomes nil and a struct is left
// as it is.
func TestNoneIntoAWrapper(t *testing.T) {
	l := isr.UseDL(insyra.NewDataList(1))
	if err := bindJSON(t, &l, `null`); err != nil {
		t.Fatal(err)
	}
	if l != nil {
		t.Errorf("None into an isr list pointer gave %v, want nil", l)
	}
	w := tableWrap{insyra.NewDataTable(insyra.NewDataList(1, 2))}
	if err := bindJSON(t, &w, `null`); err != nil {
		t.Fatal(err)
	}
	if w.DataTable == nil {
		t.Error("None into a wrapper value dropped its table")
	}
}

type tableWrap struct{ *insyra.DataTable }

// A wrapper type below the top level used to get no table.
func TestWrappersInsideASliceOrAMap(t *testing.T) {
	var list []tableWrap
	if err := bindJSON(t, &list, `[`+frame+`]`); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d wrappers, want 1", len(list))
	}
	checkFrame(t, "a wrapper in a slice", list[0].DataTable)

	var byName map[string]*tableWrap
	if err := bindJSON(t, &byName, `{"a":`+frame+`}`); err != nil {
		t.Fatal(err)
	}
	if byName["a"] == nil {
		t.Fatal("no wrapper under a")
	}
	checkFrame(t, "a wrapper in a map", byName["a"].DataTable)
}

type lowerKey string

func (k *lowerKey) UnmarshalText(text []byte) error {
	*k = lowerKey(strings.ToLower(string(text)))
	return nil
}

// Map keys follow encoding/json: a string, an integer, or a type with its own
// UnmarshalText. An integer key used to leave its table empty.
func TestTablesInAMapWithIntegerOrTextKeys(t *testing.T) {
	var byNumber map[int]*insyra.DataTable
	if err := bindJSON(t, &byNumber, `{"1":`+frame+`}`); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "under the key 1", byNumber[1])

	var byText map[lowerKey]*insyra.DataTable
	if err := bindJSON(t, &byText, `{"ABC":`+frame+`}`); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "under the key abc", byText["abc"])

	var bad map[int8]*insyra.DataTable
	if err := bindJSON(t, &bad, `{"300":`+frame+`}`); err == nil {
		t.Error("a key out of the int8 range gave no error")
	}
	if err := bindJSON(t, &byNumber, `{"x":`+frame+`}`); err == nil {
		t.Error("a key that is not a number gave no error for map[int]")
	}
}

// Decoding into a value that already holds something keeps what the result
// leaves out, as encoding/json does.
func TestDecodingIntoAnExistingValueKeepsWhatTheResultLeavesOut(t *testing.T) {
	type kept struct {
		Keep  int
		Table *insyra.DataTable
	}
	p := &kept{Keep: 42}
	if err := bindJSON(t, &p, `{"Table":`+frame+`}`); err != nil {
		t.Fatal(err)
	}
	if p.Keep != 42 {
		t.Errorf("Keep is %d, want 42", p.Keep)
	}
	checkFrame(t, "a table in an existing struct", p.Table)

	m := map[string]*insyra.DataTable{"old": insyra.NewDataTable()}
	if err := bindJSON(t, &m, `{"new":`+frame+`}`); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["old"]; !ok {
		t.Error("the existing key was dropped")
	}
	checkFrame(t, "a table added to an existing map", m["new"])

	arr := [2]*insyra.DataTable{insyra.NewDataTable(), insyra.NewDataTable()}
	if err := bindJSON(t, &arr, `[`+frame+`]`); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "the first element", arr[0])
	if arr[1] != nil {
		t.Error("an element past the end of the result was kept; encoding/json sets it to its zero value")
	}
}

type sharedName struct{ Name string }
type sharedNameToo struct{ Name string }
type shadowedBase struct{ Score int }

// LazyInner is exported so an embedded pointer to it can be allocated.
type LazyInner struct{ Y int }

type lazyHidden struct{ Y int }

// Struct fields used to be matched by a loop of this package's own, which
// chose among case-insensitive keys at random, filled two fields from one key,
// filled hidden and conflicting fields, read "-," as "skip" and allocated an
// embedded pointer no key reached.
func TestFieldsAreMatchedTheWayEncodingJSONMatchesThem(t *testing.T) {
	type folded struct {
		Score int `json:"score"`
		Table *insyra.DataTable
	}
	for i := 0; i < 50; i++ {
		sameAsEncodingJSON[folded](t, "two keys matching without regard to case", `{"SCORE":1,"Score":2}`)
	}

	type twoCases struct {
		A     int `json:"x"`
		B     int `json:"X"`
		Table *insyra.DataTable
	}
	sameAsEncodingJSON[twoCases](t, "tags x and X", `{"x":1}`)

	type shadowing struct {
		shadowedBase
		Score int
		Table *insyra.DataTable
	}
	sameAsEncodingJSON[shadowing](t, "a field hiding an embedded one", `{"Score":7}`)

	type conflicting struct {
		sharedName
		sharedNameToo
		Table *insyra.DataTable
	}
	sameAsEncodingJSON[conflicting](t, "two fields of one name at one depth", `{"Name":"x"}`)

	type dash struct {
		D     int `json:"-,"` //nolint:staticcheck // "-," naming a field "-" is the rule under test
		Table *insyra.DataTable
	}
	sameAsEncodingJSON[dash](t, "a field named -", `{"-":3}`)

	type lazy struct {
		*LazyInner
		Table *insyra.DataTable
	}
	sameAsEncodingJSON[lazy](t, "an embedded pointer no key reaches", `{"Table":null}`)
	sameAsEncodingJSON[lazy](t, "an embedded pointer a key reaches", `{"Y":1}`)

	type hidden struct {
		*lazyHidden
		Table *insyra.DataTable
	}
	if err := stdjson.Unmarshal([]byte(`{"Y":1}`), &hidden{}); err == nil {
		t.Fatal("encoding/json now sets a nil embedded pointer to an unexported struct")
	}
	var h hidden
	if err := bindJSON(t, &h, `{"Y":1}`); err == nil {
		t.Error("a key for a nil embedded pointer to an unexported struct gave no error; encoding/json cannot set one either")
	}
}

type tableWrapToo struct{ *insyra.DataTable }

// The second review found that an untagged embedded table beside other fields
// read the struct's own object as a one-row table, and that an embedded
// IDataTable, a named field for encoding/json, did the same.
func TestAnEmbeddedTableBesideFieldsTakesOnlyATable(t *testing.T) {
	type beside struct {
		*insyra.DataTable
		Score float64
	}
	var b beside
	if err := bindJSON(t, &b, `{"a":1,"b":2,"Score":7}`); err != nil {
		t.Fatal(err)
	}
	if b.DataTable != nil {
		t.Errorf("the struct's own object became a %v table", b.ColNames())
	}
	if b.Score != 7 {
		t.Errorf("Score is %v, want 7", b.Score)
	}

	b = beside{}
	if err := bindJSON(t, &b, frame); err != nil {
		t.Fatal(err)
	}
	checkFrame(t, "a DataFrame for an embedded table beside a field", b.DataTable)

	type viaInterface struct {
		insyra.IDataTable
		N int
	}
	var v viaInterface
	if err := bindJSON(t, &v, `{"IDataTable":`+frame+`,"N":2}`); err != nil {
		t.Fatal(err)
	}
	dt, ok := v.IDataTable.(*insyra.DataTable)
	if !ok {
		t.Fatalf("the embedded IDataTable holds %T", v.IDataTable)
	}
	checkFrame(t, "an embedded IDataTable under its name", dt)
	if v.N != 2 {
		t.Errorf("N is %d, want 2", v.N)
	}

	// A wrapper alone still reads a dict as a one-row table, as a
	// *insyra.DataTable does.
	var w tableWrap
	if err := bindJSON(t, &w, `{"a":1,"b":2}`); err != nil {
		t.Fatal(err)
	}
	if w.DataTable == nil {
		t.Fatal("a dict for a wrapper gave no table")
	}
	if rows, cols := w.Size(); rows != 1 || cols != 2 {
		t.Errorf("a dict for a wrapper gave a %d x %d table, want 1 x 2", rows, cols)
	}
}

// Two tables embedded at one depth hide each other, as two fields of one name
// at one depth do.
func TestTwoEmbeddedTablesAtOneDepthHideEachOther(t *testing.T) {
	type both struct {
		tableWrap
		tableWrapToo
	}
	var b both
	if err := bindJSON(t, &b, frame); err != nil {
		t.Fatal(err)
	}
	if b.tableWrap.DataTable != nil || b.tableWrapToo.DataTable != nil {
		t.Error("a table hidden by another at the same depth was filled")
	}
}

// A ",string" value that does not decode leaves the field as it was, as
// encoding/json does, rather than half written.
func TestAQuotedValueThatFailsLeavesTheField(t *testing.T) {
	type quoted struct {
		N     int `json:",string"`
		Table *insyra.DataTable
	}
	q := quoted{N: 3}
	if err := bindJSON(t, &q, `{"N":"5.0"}`); err == nil {
		t.Error(`"5.0" for an int gave no error`)
	}
	if q.N != 3 {
		t.Errorf("N is %d after a failed decode, want 3", q.N)
	}
}

// A slice decodes into the elements it already has, as encoding/json does.
func TestASliceHoldingTablesDecodesIntoItsElements(t *testing.T) {
	type elem struct {
		Keep  int
		X     int
		Table *insyra.DataTable
	}
	list := []elem{{Keep: 1}}
	if err := bindJSON(t, &list, `[{"X":2},{"X":3}]`); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Keep != 1 || list[0].X != 2 || list[1].X != 3 {
		t.Errorf("got %+v, want the first element to keep Keep 1", list)
	}
}
