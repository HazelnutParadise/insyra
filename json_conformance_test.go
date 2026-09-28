// These tests pin insyra's JSON paths against encoding/json. go-json v0.11
// made them agree; v0.10.6 read a leading zero as a number, failed a whole read
// on 1e400, kept invalid UTF-8 as raw bytes and wrote 1e-07.

package insyra

import (
	"bytes"
	stdjson "encoding/json"
	"math"
	"testing"
	"time"
)

func conformanceColOf(t *testing.T, dt *DataTable, name string) int {
	t.Helper()
	for i, n := range dt.ColNames() {
		if n == name {
			return i
		}
	}
	t.Fatalf("no column named %q among %v", name, dt.ColNames())
	return -1
}

func TestReadJSONRefusesALeadingZero(t *testing.T) {
	dt, err := ReadJSON([]byte(`[{"a":01}]`))
	if err == nil {
		t.Errorf("ReadJSON accepted a leading zero, table %v", dt)
	}
	if dt != nil {
		t.Errorf("expected a nil table alongside the error, got %v", dt)
	}
}

func TestReadJSONRefusesATrailingComma(t *testing.T) {
	dt, err := ReadJSON([]byte(`[{"a":1,}]`))
	if err == nil {
		t.Errorf("ReadJSON accepted a trailing comma, table %v", dt)
	}
	if dt != nil {
		t.Errorf("expected a nil table alongside the error, got %v", dt)
	}
}

func TestReadJSONKeepsAnOutOfRangeNumberAsText(t *testing.T) {
	dt, err := ReadJSON([]byte(`[{"a":1e400,"b":1}]`))
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if dt == nil {
		t.Fatal("ReadJSON returned a nil table with no error")
	}
	aCol := conformanceColOf(t, dt, "a")
	bCol := conformanceColOf(t, dt, "b")

	gotA := dt.GetElementByNumberIndex(0, aCol)
	if s, ok := gotA.(string); !ok || s != "1e400" {
		t.Errorf(`column a: expected string("1e400"), got %v (%T)`, gotA, gotA)
	}
	gotB := dt.GetElementByNumberIndex(0, bCol)
	if i, ok := gotB.(int64); !ok || i != 1 {
		t.Errorf("column b: expected int64(1), got %v (%T)", gotB, gotB)
	}
}

func TestReadJSONReplacesInvalidUTF8LikeEncodingJSON(t *testing.T) {
	input := []byte("[{\"a\":\"x\xffy\"}]")

	var stdRows []map[string]any
	if err := stdjson.Unmarshal(input, &stdRows); err != nil {
		t.Fatalf("encoding/json could not decode the fixture: %v", err)
	}
	if len(stdRows) != 1 {
		t.Fatalf("encoding/json decoded %d rows, expected 1", len(stdRows))
	}
	want, ok := stdRows[0]["a"].(string)
	if !ok {
		t.Fatalf("encoding/json decoded a as %T, not a string", stdRows[0]["a"])
	}
	if want != "x\ufffdy" {
		t.Fatalf("encoding/json decoded a as %q, expected %q", want, "x\ufffdy")
	}

	dt, err := ReadJSON(input)
	if err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if dt == nil {
		t.Fatal("ReadJSON returned a nil table with no error")
	}
	got := dt.GetElementByNumberIndex(0, conformanceColOf(t, dt, "a"))
	gotStr, ok := got.(string)
	if !ok {
		t.Fatalf("expected a string cell, got %v (%T)", got, got)
	}
	if gotStr != want {
		t.Errorf("ReadJSON read %q, encoding/json read %q", gotStr, want)
	}
	if gotStr != "x\ufffdy" {
		t.Errorf("expected the 0xFF byte to become U+FFFD, got %q", gotStr)
	}
}

func TestToJSONWritesWhatEncodingJSONWrites(t *testing.T) {
	dt := NewDataTable(
		NewDataList(1e-7, 2.5e-8, 1e21, int64(math.MaxInt64), int64(math.MinInt64), float32(0.1)).SetName("num"),
		NewDataList("<a&b>", "\u2028", "é中文", nil, true, false).SetName("str"),
		NewDataList(time.Date(2024, 1, 2, 3, 4, 5, 600, time.UTC), nil, "x", 1.5, int64(7), uint8(3)).SetName("mixed"),
	)

	for _, useColNames := range []bool{true, false} {
		want, err := stdjson.MarshalIndent(dt.buildJSONRows(useColNames), "", "  ")
		if err != nil {
			t.Fatalf("useColNames=%v: encoding/json.MarshalIndent: %v", useColNames, err)
		}
		got := dt.ToJSON_Bytes(useColNames)
		if !bytes.Equal(got, want) {
			t.Errorf("useColNames=%v:\n got %s\nwant %s", useColNames, got, want)
		}
		if !bytes.Contains(got, []byte("1e-7")) {
			t.Errorf("useColNames=%v: expected 1e-7 in the output, got %s", useColNames, got)
		}
		if bytes.Contains(got, []byte("1e-07")) {
			t.Errorf("useColNames=%v: 1e-07 is not what encoding/json writes, got %s", useColNames, got)
		}
	}
}
