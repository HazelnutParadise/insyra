//go:build !windows

// The write is atomic only if the result is what the caller would have got
// from writing over the file or creating it: the replaced file's modes are
// kept and a new file follows the creating umask. The umask test changes the
// process umask, so it must not run in parallel with anything.
package insyra

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Replacing a file must not widen it: a workbook saved over a 0600 file stays
// 0600.
func TestToExcelKeepsTheModeOfAnExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	workbookWith(t, path, "only")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := excelTestTable(3).ToExcel(path, ExcelWriteOptions{Sheet: "only", IfSheetExists: SheetExistsReplace}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode %04o, want 0600", got)
	}
}

// A new file takes the mode the umask would give it. Not parallel: the umask
// is a process-wide value that this test changes and restores.
func TestToExcelNewFileHonoursUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	path := filepath.Join(t.TempDir(), "new.xlsx")
	if err := excelTestTable(1).ToExcel(path, ExcelWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode %04o, want 0600 under umask 077", got)
	}
}
