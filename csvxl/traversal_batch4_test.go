package csvxl

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// craftWorkbookWithSheetName writes a workbook whose first sheet is renamed
// to rawName directly inside workbook.xml, bypassing excelize's validation.
func craftWorkbookWithSheetName(t *testing.T, path, rawName string) {
	t.Helper()
	f := excelize.NewFile()
	if err := f.SetCellValue("Sheet1", "A1", "x"); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if zf.Name == "xl/workbook.xml" {
			data = bytes.Replace(data, []byte(`name="Sheet1"`), []byte(`name="`+rawName+`"`), 1)
		}
		w, err := zw.Create(zf.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// SEC-1: a sheet named "../important" must not reach outside outputDir, and
// must not truncate anything before the sheet is known to be readable.
func TestExcelToCsvRejectsTraversingSheetName(t *testing.T) {
	root := t.TempDir()
	victim := filepath.Join(root, "important.csv")
	if err := os.WriteFile(victim, []byte("precious data\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(root, "out")
	xlsx := filepath.Join(root, "evil.xlsx")
	craftWorkbookWithSheetName(t, xlsx, "../important")

	err := ExcelToCsv(xlsx, outDir, nil)
	if err == nil {
		t.Fatal("ExcelToCsv accepted a sheet name that escapes the output directory")
	}
	if !strings.Contains(err.Error(), "sheet name") {
		t.Fatalf("error should name the sheet-name check, got %v", err)
	}
	b, _ := os.ReadFile(victim)
	if string(b) != "precious data\n" {
		t.Fatalf("victim file was modified: %q", b)
	}
	if err := EachExcelToCsv(root, outDir); err == nil {
		t.Fatal("EachExcelToCsv accepted a sheet name that escapes the output directory")
	}
	b, _ = os.ReadFile(victim)
	if string(b) != "precious data\n" {
		t.Fatalf("victim file was modified by EachExcelToCsv: %q", b)
	}
}

// A sheet named "." or ".." is an ordinary name: v0.3.2 wrote it to "..csv"
// and "...csv" inside the output directory. Only a name taken from the workbook
// is checked: a caller's csvNames entry is used as given, as on v0.3.2, so it
// can place the CSV in a subdirectory.
func TestExcelToCsvChecksTheFileNameActuallyUsed(t *testing.T) {
	root := t.TempDir()
	outDir := filepath.Join(root, "out")

	for sheet, file := range map[string]string{".": "..csv", "..": "...csv"} {
		xlsx := filepath.Join(root, "dots.xlsx")
		craftWorkbookWithSheetName(t, xlsx, sheet)
		if err := ExcelToCsv(xlsx, outDir, nil); err != nil {
			t.Fatalf("ExcelToCsv refused a sheet named %q: %v", sheet, err)
		}
		if _, err := os.Stat(filepath.Join(outDir, file)); err != nil {
			t.Fatalf("a sheet named %q was not written to %s: %v", sheet, file, err)
		}
		if err := EachExcelToCsv(root, outDir); err != nil {
			t.Fatalf("EachExcelToCsv refused a sheet named %q: %v", sheet, err)
		}
		if _, err := os.Stat(filepath.Join(outDir, "dots_"+file)); err != nil {
			t.Fatalf("EachExcelToCsv did not write dots_%s: %v", file, err)
		}
	}

	plain := filepath.Join(root, "plain", "plain.xlsx")
	if err := os.MkdirAll(filepath.Dir(plain), 0o755); err != nil {
		t.Fatal(err)
	}
	craftWorkbookWithSheetName(t, plain, "sheet")
	if err := os.MkdirAll(filepath.Join(outDir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ExcelToCsv(plain, outDir, []string{"sub/renamed"}); err != nil {
		t.Fatalf("a csvNames entry naming a subdirectory was refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "sub", "renamed.csv")); err != nil {
		t.Fatalf("the csvNames entry was not used as given: %v", err)
	}

	slashed := filepath.Join(root, "slashed", "slashed.xlsx")
	if err := os.MkdirAll(filepath.Dir(slashed), 0o755); err != nil {
		t.Fatal(err)
	}
	craftWorkbookWithSheetName(t, slashed, "a/b")
	err := ExcelToCsv(slashed, outDir, nil)
	if err == nil || !strings.Contains(err.Error(), "sheet name") {
		t.Fatalf("a sheet name holding a separator was not refused by the name check: %v", err)
	}
}
