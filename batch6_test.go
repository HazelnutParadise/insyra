package insyra

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// SEC-5 (a): a CSV in a charset outside UTF-8/UTF-16/Big5/GB18030 used to be
// copied byte-for-byte into the table. Named explicitly, it must be decoded.
func TestReadCSVDecodesLatin1(t *testing.T) {
	quietLogs(t)

	// latin-1 bytes: "café numéro" with 0xE9 for é.
	path := filepath.Join(t.TempDir(), "latin1.csv")
	body := append([]byte("name\n"), []byte{'c', 'a', 'f', 0xE9, ' ', 'n', 'u', 'm', 0xE9, 'r', 'o', '\n'}...)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, enc := range []string{"iso-8859-1", "windows-1252", "latin1"} {
		dt, err := ReadCSV_File(path, false, true, enc)
		if err != nil {
			t.Fatalf("%s: %v", enc, err)
		}
		got, _ := dt.GetColByNumber(0).Get(0).(string)
		if !utf8.ValidString(got) {
			t.Fatalf("%s: cell is not valid UTF-8: %q — the bytes were copied without decoding", enc, got)
		}
		if got != "café numéro" {
			t.Fatalf("%s: cell = %q, want the decoded text", enc, got)
		}
	}
}

// With auto-detection, whatever charset the detector names must produce valid
// UTF-8 text rather than the raw bytes.
func TestReadCSVAutoDecodesLatin1(t *testing.T) {
	quietLogs(t)

	path := filepath.Join(t.TempDir(), "latin1.csv")
	body := append([]byte("name\n"), []byte{'c', 'a', 'f', 0xE9, ' ', 'n', 'u', 'm', 0xE9, 'r', 'o', '\n'}...)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}

	dt, err := ReadCSV_File(path, false, true)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := dt.GetColByNumber(0).Get(0).(string)
	if !utf8.ValidString(got) {
		t.Fatalf("cell is not valid UTF-8: %q — the bytes were copied without decoding", got)
	}
	if !strings.Contains(got, "café") {
		t.Fatalf("cell = %q, want the decoded text", got)
	}
}

// SEC-5 (d): a UTF-32LE BOM starts with the UTF-16LE BOM, so the prefix check
// must test the longer one first.
func TestDetectEncodingUTF32BOM(t *testing.T) {
	dir := t.TempDir()
	cases := map[string][]byte{
		"utf-32le": {0xFF, 0xFE, 0x00, 0x00, 'a', 0, 0, 0},
		"utf-32be": {0x00, 0x00, 0xFE, 0xFF, 0, 0, 0, 'a'},
		"utf-16le": {0xFF, 0xFE, 'a', 0},
		"utf-16be": {0xFE, 0xFF, 0, 'a'},
	}
	for want, body := range cases {
		p := filepath.Join(dir, want+".txt")
		if err := os.WriteFile(p, body, 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := DetectEncoding(p)
		if err != nil {
			t.Errorf("%s: %v", want, err)
			continue
		}
		if got != want {
			t.Errorf("DetectEncoding(%s) = %q, want %q", want, got, want)
		}
	}
}

// A UTF-32 file with a BOM reads back as its text end to end.
func TestReadCSVUTF32WithBOM(t *testing.T) {
	quietLogs(t)

	text := "name\nhé\n"
	body := []byte{0xFF, 0xFE, 0x00, 0x00}
	for _, r := range text {
		body = append(body, byte(r), byte(r>>8), byte(r>>16), byte(r>>24))
	}
	path := filepath.Join(t.TempDir(), "utf32.csv")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}

	dt, err := ReadCSV_File(path, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.GetColByNumber(0).GetName(); got != "name" {
		t.Fatalf("column name = %q, want %q", got, "name")
	}
	if got, _ := dt.GetColByNumber(0).Get(0).(string); got != "hé" {
		t.Fatalf("cell = %q, want %q", got, "hé")
	}
}

// SEC-3: a cell that a spreadsheet would execute as a formula can be escaped
// on request. The default is unchanged so a round trip keeps the exact value.
func TestToCSVFormulaSanitization(t *testing.T) {
	quietLogs(t)

	dt := NewDataTable(NewDataList("=cmd|' /C calc'!A0", "+1+1", "@SUM(A1)", "-2+3", "safe").SetName("v"))
	dir := t.TempDir()

	plain := filepath.Join(dir, "plain.csv")
	if err := dt.ToCSV(plain, false, true, false); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(plain)
	if !strings.Contains(string(raw), "=cmd") {
		t.Fatalf("ToCSV changed the value by default:\n%s", raw)
	}

	// The zero options write exactly what ToCSV writes.
	zero := filepath.Join(dir, "zero.csv")
	if err := dt.ToCSVWithOptions(zero, CSVWriteOptions{SetColNamesToFirstRow: true}); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, zero); string(got) != string(raw) {
		t.Fatalf("ToCSVWithOptions without SanitizeFormulas differs from ToCSV:\n%s\nvs\n%s", got, raw)
	}

	safe := filepath.Join(dir, "safe.csv")
	if err := dt.ToCSVWithOptions(safe, CSVWriteOptions{
		SetColNamesToFirstRow: true,
		SanitizeFormulas:      true,
	}); err != nil {
		t.Fatal(err)
	}
	out := string(mustRead(t, safe))
	for _, dangerous := range []string{"\n=cmd", "\n+1+1", "\n@SUM", "\n-2+3"} {
		if strings.Contains(out, dangerous) {
			t.Fatalf("cell starting with a formula character was not escaped: %q in\n%s", dangerous, out)
		}
	}
	if !strings.Contains(out, "safe") {
		t.Fatalf("ordinary cell was lost:\n%s", out)
	}
	if strings.Contains(out, "'safe") {
		t.Fatalf("ordinary cell was escaped:\n%s", out)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
