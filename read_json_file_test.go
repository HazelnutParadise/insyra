package insyra_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/HazelnutParadise/insyra"
)

func TestReadJSONFileKeepsLargeInt(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.json")
	if err := os.WriteFile(p, []byte(`[{"id": 9007199254740993, "v": 1.5}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	dt, err := insyra.ReadJSON_File(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByName("id").Get(0); got != int64(9007199254740993) {
		t.Fatalf("expected int64(9007199254740993), got %v (%T)", got, got)
	}
	if got := dt.GetColByName("v").Get(0); got != 1.5 {
		t.Fatalf("expected 1.5, got %v (%T)", got, got)
	}
}

func TestReadJSONFileMatchesReadJSON(t *testing.T) {
	body := []byte(`[{"a": 1, "b": "x"}, {"a": 2, "b": "y"}]`)
	p := filepath.Join(t.TempDir(), "d.json")
	if err := os.WriteFile(p, body, 0o644); err != nil {
		t.Fatal(err)
	}
	fromFile, err := insyra.ReadJSON_File(p)
	if err != nil {
		t.Fatal(err)
	}
	fromBytes, err := insyra.ReadJSON(body)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromFile.To2DSlice(), fromBytes.To2DSlice()) {
		t.Fatalf("file %v != bytes %v", fromFile.To2DSlice(), fromBytes.To2DSlice())
	}
}

func TestReadJSONFileSingleObject(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.json")
	if err := os.WriteFile(p, []byte(`{"a": 1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dt, err := insyra.ReadJSON_File(p)
	if err != nil {
		t.Fatal(err)
	}
	if r, c := dt.Size(); r != 1 || c != 1 {
		t.Fatalf("expected 1x1, got %dx%d", r, c)
	}
}
