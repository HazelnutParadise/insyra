package insyra

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func scaleJSONFitTable() *DataTable {
	return NewDataTable(
		NewDataList(1.5, nil, 3.0, 10.0).SetName("a"),
		NewDataList(4, 8, 15, 16).SetName("b"),
	)
}

func scaleJSONOtherTable() *DataTable {
	return NewDataTable(
		NewDataList(2.0, 7.25, nil).SetName("a"),
		NewDataList(0, 3, 99).SetName("b"),
	)
}

// scaleJSONPair pairs a constructor with the zero value of the same concrete
// type, which is what json.Unmarshal writes into.
type scaleJSONPair struct {
	name    string
	orig    func() Scaler
	restore func() Scaler
}

func scaleJSONPairs() []scaleJSONPair {
	return []scaleJSONPair{
		{"standard", func() Scaler { return NewStandardScaler() }, func() Scaler { return new(StandardScaler) }},
		{"minmax", func() Scaler { return NewMinMaxScaler(-1, 1) }, func() Scaler { return new(MinMaxScaler) }},
		{"robust", func() Scaler { return NewRobustScaler() }, func() Scaler { return new(RobustScaler) }},
		{"maxabs", func() Scaler { return NewMaxAbsScaler() }, func() Scaler { return new(MaxAbsScaler) }},
	}
}

// assertScaleJSONSameCells compares cell by cell, requiring identical float64
// bit patterns. A nil cell on either side must be nil on both.
func assertScaleJSONSameCells(t *testing.T, where string, want, got []any) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d", where, len(got), len(want))
	}
	for i := range want {
		if want[i] == nil || got[i] == nil {
			if want[i] != got[i] {
				t.Fatalf("%s cell %d = %#v, want nil", where, i, got[i])
			}
			continue
		}
		wf, ok := ToFloat64Safe(want[i])
		if !ok {
			t.Fatalf("%s cell %d: want %v is not numeric", where, i, want[i])
		}
		gf, ok := ToFloat64Safe(got[i])
		if !ok {
			t.Fatalf("%s cell %d: got %v is not numeric", where, i, got[i])
		}
		if math.Float64bits(gf) != math.Float64bits(wf) {
			t.Fatalf("%s cell %d = %v (%#016x), want %v (%#016x)",
				where, i, gf, math.Float64bits(gf), wf, math.Float64bits(wf))
		}
	}
}

func assertScaleJSONSameTable(t *testing.T, want, got *DataTable) {
	t.Helper()
	if want == nil || got == nil {
		t.Fatalf("table is nil (want nil=%v, got nil=%v)", want == nil, got == nil)
	}
	wantNames, gotNames := want.ColNames(), got.ColNames()
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("column names = %v, want %v", gotNames, wantNames)
	}
	for _, name := range wantNames {
		assertScaleJSONSameCells(t, "column "+name,
			want.GetColByName(name).Data(), got.GetColByName(name).Data())
	}
}

func assertScaleJSONSameList(t *testing.T, want, got *DataList) {
	t.Helper()
	if want == nil || got == nil {
		t.Fatalf("list is nil (want nil=%v, got nil=%v)", want == nil, got == nil)
	}
	if want.GetName() != got.GetName() {
		t.Fatalf("list name = %q, want %q", got.GetName(), want.GetName())
	}
	assertScaleJSONSameCells(t, "list "+want.GetName(), want.Data(), got.Data())
}

func TestScalerJSONRoundTrip(t *testing.T) {
	for _, p := range scaleJSONPairs() {
		t.Run(p.name, func(t *testing.T) {
			orig := p.orig()
			if err := orig.Fit(scaleJSONFitTable(), Name("a"), "B"); err != nil {
				t.Fatalf("Fit: %v", err)
			}

			data, err := json.Marshal(orig)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			got := p.restore()
			if err := json.Unmarshal(data, got); err != nil {
				t.Fatalf("Unmarshal: %v\n%s", err, data)
			}

			if got.Kind() != orig.Kind() {
				t.Fatalf("Kind = %q, want %q", got.Kind(), orig.Kind())
			}
			if !reflect.DeepEqual(got.Params(), orig.Params()) {
				t.Fatalf("Params = %+v, want %+v", got.Params(), orig.Params())
			}

			other := scaleJSONOtherTable()
			wantOut, err := orig.Transform(other)
			if err != nil {
				t.Fatalf("orig Transform: %v", err)
			}
			gotOut, err := got.Transform(other)
			if err != nil {
				t.Fatalf("restored Transform: %v", err)
			}
			assertScaleJSONSameTable(t, wantOut, gotOut)

			wantBack, err := orig.InverseTransform(other)
			if err != nil {
				t.Fatalf("orig InverseTransform: %v", err)
			}
			gotBack, err := got.InverseTransform(other)
			if err != nil {
				t.Fatalf("restored InverseTransform: %v", err)
			}
			assertScaleJSONSameTable(t, wantBack, gotBack)
		})
	}
}

func TestScalerJSONDataList(t *testing.T) {
	dl := NewDataList(1.0, 2.0, 4.0).SetName("x")
	orig := NewStandardScaler()
	if err := orig.FitDataList(dl); err != nil {
		t.Fatalf("FitDataList: %v", err)
	}
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := new(StandardScaler)
	if err := json.Unmarshal(data, got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, data)
	}
	wantOut, err := orig.TransformDataList(dl)
	if err != nil {
		t.Fatalf("orig TransformDataList: %v", err)
	}
	gotOut, err := got.TransformDataList(dl)
	if err != nil {
		t.Fatalf("restored TransformDataList: %v", err)
	}
	assertScaleJSONSameList(t, wantOut, gotOut)
}

func TestScalerJSONKeepsNaNParams(t *testing.T) {
	dt := NewDataTable(NewDataList(nil, nil).SetName("x"))
	orig := NewStandardScaler()
	if err := orig.Fit(dt, Name("x")); err != nil {
		t.Fatalf("Fit: %v", err)
	}
	if !math.IsNaN(orig.Params()["x"].Mean) {
		t.Fatalf("fitted mean = %v, want NaN", orig.Params()["x"].Mean)
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal with NaN params: %v", err)
	}
	got := new(StandardScaler)
	if err := json.Unmarshal(data, got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, data)
	}
	if !math.IsNaN(got.Params()["x"].Mean) {
		t.Fatalf("restored mean = %v, want NaN", got.Params()["x"].Mean)
	}
}

func TestScalerJSONRefusesOtherKind(t *testing.T) {
	fit := NewDataTable(NewDataList(1.0, 2.0, 3.0).SetName("x"))

	target := NewRobustScaler()
	if err := target.Fit(fit, Name("x")); err != nil {
		t.Fatalf("Fit robust: %v", err)
	}
	before := target.Params()
	probe := NewDataTable(NewDataList(2.0, 4.0).SetName("x"))
	beforeOut, err := target.Transform(probe)
	if err != nil {
		t.Fatalf("orig Transform: %v", err)
	}

	src := NewStandardScaler()
	if err := src.Fit(fit, Name("x")); err != nil {
		t.Fatalf("Fit standard: %v", err)
	}
	data, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := json.Unmarshal(data, target); err == nil {
		t.Fatalf("expected an error reading a standard scaler into a robust scaler")
	}

	if !reflect.DeepEqual(target.Params(), before) {
		t.Fatalf("Params = %+v, want unchanged %+v", target.Params(), before)
	}
	afterOut, err := target.Transform(probe)
	if err != nil {
		t.Fatalf("Transform after failed Unmarshal: %v", err)
	}
	assertScaleJSONSameTable(t, beforeOut, afterOut)
}

func TestScalerJSONUnfitted(t *testing.T) {
	orig := NewMaxAbsScaler()
	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !bytes.Contains(data, []byte(`"columns":[]`)) {
		t.Fatalf("unfitted JSON = %s, want an empty columns array", data)
	}
	got := new(MaxAbsScaler)
	if err := json.Unmarshal(data, got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, data)
	}
	if got.Kind() != orig.Kind() {
		t.Fatalf("Kind = %q, want %q", got.Kind(), orig.Kind())
	}
	if len(got.Params()) != 0 {
		t.Fatalf("Params = %+v, want none", got.Params())
	}
	if _, err := got.Transform(scaleJSONOtherTable()); err == nil {
		t.Fatalf("expected an error transforming with a restored unfitted scaler")
	}
}

func TestScalerJSONKeepsTheFittedColumnReference(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1.0, 2.0, 3.0),
		NewDataList(4.0, 5.0, 6.0).SetName("b"),
		NewDataList(7.0, 8.0, 9.0).SetName("q"),
	)
	orig := NewStandardScaler()
	if err := orig.Fit(dt, "A", Name("b"), 2); err != nil {
		t.Fatalf("Fit: %v", err)
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"position":0`) {
		t.Fatalf("JSON missing position:0 for first unnamed column: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"name":"b"`) {
		t.Fatalf("JSON missing name selector for second column: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"name":"q"`) {
		t.Fatalf("JSON missing name selector for third column: %s", jsonStr)
	}

	got := new(StandardScaler)
	if err := json.Unmarshal(data, got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, data)
	}

	wantOut, err := orig.Transform(dt)
	if err != nil {
		t.Fatalf("orig Transform: %v", err)
	}
	gotOut, err := got.Transform(dt)
	if err != nil {
		t.Fatalf("restored Transform: %v", err)
	}
	assertScaleJSONSameTable(t, wantOut, gotOut)
}

func TestScalerRefRoundTrips(t *testing.T) {
	for _, v := range []any{"A", Name("b"), 2} {
		r, err := encodeScalerRef(v)
		if err != nil {
			t.Fatalf("encodeScalerRef(%v): %v", v, err)
		}
		got, err := decodeScalerRef(r)
		if err != nil {
			t.Fatalf("decodeScalerRef(%v): %v", r, err)
		}
		if !reflect.DeepEqual(got, v) {
			t.Fatalf("round-trip %v: got %v, want %v", v, got, v)
		}
	}

	if _, err := encodeScalerRef(3.5); err == nil {
		t.Fatalf("encodeScalerRef(3.5) expected error, got none")
	}
	if _, err := decodeScalerRef(scalerRefJSON{}); err == nil {
		t.Fatalf("decodeScalerRef(empty) expected error, got none")
	}
}

func TestScalerJSONRefusesAnEmptyReference(t *testing.T) {
	dt := NewDataTable(NewDataList(1.0, 2.0, 3.0).SetName("x"))
	orig := NewStandardScaler()
	if err := orig.Fit(dt, Name("x")); err != nil {
		t.Fatalf("Fit: %v", err)
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	jsonStr := string(data)
	badJSON := regexp.MustCompile(`"ref":\{[^}]*\}`).ReplaceAllString(jsonStr, `"ref":{}`)
	if badJSON == jsonStr {
		t.Fatalf("could not replace ref object in JSON: %s", jsonStr)
	}

	got := new(StandardScaler)
	err = json.Unmarshal([]byte(badJSON), got)
	if err == nil {
		t.Fatalf("expected an error for empty ref, got none")
	}
	if !strings.Contains(err.Error(), "a column reference needs exactly one of index, name or position") {
		t.Fatalf("wrong error: %v", err)
	}

	// Receiver should be unchanged from its zero state (unfitted, no params)
	if got.Kind() != "" {
		t.Fatalf("receiver kind changed: %q, want empty", got.Kind())
	}
	if len(got.Params()) != 0 {
		t.Fatalf("receiver params changed: %+v, want empty", got.Params())
	}
}
