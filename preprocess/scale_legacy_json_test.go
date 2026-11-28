package preprocess

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

// legacyStandardJSON is what insyra.StandardScaler wrote in 0.3.4 and 0.3.5
// after Fit(dt, "a", "B") on a table whose first column is named "a" and whose
// second is unnamed: each "ref" is a plain string.
const legacyStandardJSON = `{"kind":"standard","featureMin":0,"featureMax":0,"fitted":true,"columns":[` +
	`{"ref":"a","name":"a","mean":4.833333333333333,"std":4.536885862938733,"min":0,"max":0,"median":0,"q1":0,"q3":0,"iqr":0,"maxAbs":0,"outputMin":0,"outputMax":0},` +
	`{"ref":"B","name":"B","mean":9.333333333333334,"std":6.110100926607786,"min":0,"max":0,"median":0,"q1":0,"q3":0,"iqr":0,"maxAbs":0,"outputMin":0,"outputMax":0}]}`

func legacyFitTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(1.5, 3.0, 10.0).SetName("a"),
		insyra.NewDataList(4, 8, 16),
	)
}

// A scaler saved by the 0.3.x core still loads, and transforms as the core
// scaler did: the output below is what insyra.StandardScaler.Transform gave.
func TestScalerJSONWithPlainStringRefsLoads(t *testing.T) {
	var s StandardScaler
	if err := json.Unmarshal([]byte(legacyStandardJSON), &s); err != nil {
		t.Fatal(err)
	}
	out, err := s.Transform(legacyFitTable())
	if err != nil {
		t.Fatal(err)
	}
	assertScaleJSONSameCells(t, "column a", []any{-0.734718358370645, -0.4040950971038547, 1.1388134554745}, out.GetColByNumber(0).Data())
	assertScaleJSONSameCells(t, "column B", []any{-0.8728715609439697, -0.2182178902359925, 1.0910894511799618}, out.GetColByNumber(1).Data())

	again, err := json.Marshal(&s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(again), `"ref":"a"`) || !strings.Contains(string(again), `"ref":"B"`) {
		t.Fatalf("a plain-string ref should be written back as the same string: %s", again)
	}
}

// The core looked a plain-string ref up as a name first, so "B" picks a column
// named B even when another column sits at index B.
func TestScalerJSONPlainStringRefIsANameFirst(t *testing.T) {
	var s StandardScaler
	if err := json.Unmarshal([]byte(legacyStandardJSON), &s); err != nil {
		t.Fatal(err)
	}
	dt := insyra.NewDataTable(
		insyra.NewDataList(1.5, 3.0, 10.0).SetName("a"),
		insyra.NewDataList(100, 200, 300).SetName("x"),
		insyra.NewDataList(4, 8, 16).SetName("B"),
	)
	out, err := s.Transform(dt)
	if err != nil {
		t.Fatal(err)
	}
	assertScaleJSONSameCells(t, "column x", []any{int64(100), int64(200), int64(300)}, out.GetColByNumber(1).Data())
	assertScaleJSONSameCells(t, "column named B", []any{-0.8728715609439697, -0.2182178902359925, 1.0910894511799618}, out.GetColByNumber(2).Data())
}

func TestScalerJSONRefusesAnEmptyPlainStringRef(t *testing.T) {
	var s StandardScaler
	err := json.Unmarshal([]byte(strings.Replace(legacyStandardJSON, `"ref":"a"`, `"ref":""`, 1)), &s)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("err = %v, want the empty reference refused", err)
	}
}
