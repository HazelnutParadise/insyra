package insyra

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Each name below was replaced by the one beside it. The old name stays until
// 0.4 as a Deprecated wrapper, says what replaced it and when it goes, and
// gives exactly the old name's answer.

func TestDeprecatedNamesSayWhatReplacedThem(t *testing.T) {
	const removal = "Removed in 0.4."
	deprecated := []struct{ file, old, current string }{
		{"read.go", "Slice2DToDataTable", "ReadSlice2D"},
		{"read.go", "ReadJSON_File", "ReadJSONFile"},
		{"datatable_json.go", "ToJSON_Bytes", "ToJSONBytes"},
		{"datatable_json.go", "ToJSON_String", "ToJSONString"},
	}
	for _, d := range deprecated {
		doc, ok := rootDocComments(t, d.file)[d.old]
		if !ok {
			t.Errorf("%s has no doc comment in %s", d.old, d.file)
			continue
		}
		notice := "Deprecated: use " + d.current + ", which is the same "
		if !strings.Contains(doc, notice) || !strings.Contains(doc, removal) {
			t.Errorf("%s doc is %q; want it to contain %q and %q", d.old, doc, notice, removal)
		}
	}
}

func TestDeprecatedReadersAndWritersKeepTheirResults(t *testing.T) {
	data := [][]any{{1, "a"}, {2, "b"}}
	oldDT, oldErr := Slice2DToDataTable(data) //nolint:staticcheck // the deprecated name is what is under test
	newDT, newErr := ReadSlice2D(data)
	if oldErr != nil || newErr != nil || !oldDT.IsEqualTo(newDT) {
		t.Fatalf("Slice2DToDataTable and ReadSlice2D disagree: %v, %v", oldErr, newErr)
	}

	path := filepath.Join(t.TempDir(), "t.json")
	if err := os.WriteFile(path, []byte(`[{"a":1,"b":"x"},{"a":2,"b":"y"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	oldJSON, oldErr := ReadJSON_File(path) //nolint:staticcheck // the deprecated name is what is under test
	newJSON, newErr := ReadJSONFile(path)
	if oldErr != nil || newErr != nil || !oldJSON.IsEqualTo(newJSON) {
		t.Fatalf("ReadJSON_File and ReadJSONFile disagree: %v, %v", oldErr, newErr)
	}

	for _, useColNames := range []bool{true, false} {
		if string(newJSON.ToJSON_Bytes(useColNames)) != string(newJSON.ToJSONBytes(useColNames)) { //nolint:staticcheck // under test
			t.Errorf("ToJSON_Bytes(%v) and ToJSONBytes differ", useColNames)
		}
		if newJSON.ToJSON_String(useColNames) != newJSON.ToJSONString(useColNames) { //nolint:staticcheck // under test
			t.Errorf("ToJSON_String(%v) and ToJSONString differ", useColNames)
		}
	}
}
