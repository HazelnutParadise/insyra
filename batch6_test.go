package insyra

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// SEC-5 (a): a detected encoding that no decoder handles must be an error,
// not a silent fall-through that copies the raw bytes into the table.
func TestReadCSVRejectsUndecodableEncoding(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	// windows-1252 / latin-1 bytes: "café numéro" with 0xE9 for é.
	path := filepath.Join(t.TempDir(), "latin1.csv")
	body := append([]byte("name\n"), []byte{'c', 'a', 'f', 0xE9, ' ', 'n', 'u', 'm', 0xE9, 'r', 'o', '\n'}...)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}

	dt, err := ReadCSV_File(path, false, true)
	if err != nil {
		// Refusing is acceptable; silently producing broken text is not.
		return
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

// SEC-5 (e): when the detector cannot name a charset, fall back to UTF-8 with
// a warning rather than failing the whole read.
func TestDetectEncodingFallsBackToUTF8(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

	p := filepath.Join(t.TempDir(), "short.bin")
	if err := os.WriteFile(p, []byte{0xE9}, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := DetectEncoding(p)
	if err != nil {
		t.Fatalf("DetectEncoding failed instead of falling back: %v", err)
	}
	if got == "" {
		t.Fatal("DetectEncoding returned an empty charset")
	}
}

// SEC-3: a cell that a spreadsheet would execute as a formula can be escaped
// on request. The default is unchanged so a round trip keeps the exact value.
func TestToCSVFormulaSanitization(t *testing.T) {
	restoreConfig(t)
	Config.SetLogLevel(LogLevelFatal)

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
