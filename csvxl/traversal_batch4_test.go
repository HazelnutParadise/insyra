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
