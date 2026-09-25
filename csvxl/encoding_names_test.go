package csvxl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/encoding/traditionalchinese"
)

// csvxl read "" as UTF-8 taken as-is and accepted only the lowercase "auto",
// while the core CSV readers detect the encoding for "" and for "auto" in any
// case. The same argument now means the same thing in both.
func TestEncodingNamesMeanWhatTheCoreReadersMean(t *testing.T) {
	// Detection needs enough text to tell Big5 from other double-byte
	// encodings; a two-line sample reads as Shift-JIS.
	var text strings.Builder
	text.WriteString("城市,人口,說明\n")
	for i := 0; i < 20; i++ {
		text.WriteString("台北,250,臺灣北部的直轄市，人口眾多，交通便利\n")
	}
	big5, err := traditionalchinese.Big5.NewEncoder().String(text.String())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "big5.csv")
	if err := os.WriteFile(path, []byte(big5), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "AUTO", "Auto"} {
		got, err := ReadCsvToString(path, name)
		if err != nil {
			t.Fatalf("ReadCsvToString(%q): %v", name, err)
		}
		if !strings.Contains(got, "台北") {
			t.Fatalf("ReadCsvToString(%q) did not detect Big5: %q", name, got)
		}
		out := filepath.Join(t.TempDir(), "out.xlsx")
		if err := CsvToExcel([]string{path}, []string{"s"}, out, name); err != nil {
			t.Fatalf("CsvToExcel(%q): %v", name, err)
		}
	}
}
