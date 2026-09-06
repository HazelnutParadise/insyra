package csvxl

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCsvToExcelErrorUnwraps(t *testing.T) {
	err := ExcelToCsv("/definitely/not/here.xlsx", t.TempDir(), nil)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist in chain, got %v", err)
	}
}

func TestExcelToCsvDirPerm(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "a.csv")
	if err := os.WriteFile(csv, []byte("x\n1"), 0o644); err != nil {
		t.Fatal(err)
	}
	xlsx := filepath.Join(dir, "a.xlsx")
	if err := CsvToExcel([]string{csv}, nil, xlsx, UTF8); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")
	if err := ExcelToCsv(xlsx, outDir, nil); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm&0o002 != 0 {
		t.Fatalf("output dir is world-writable: %o", perm)
	}
}
