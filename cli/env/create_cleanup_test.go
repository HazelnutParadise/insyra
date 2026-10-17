package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A default file path that holds a directory is reported, not skipped.
func TestWriteDefaultFilesRejectsDirectoryAtDefaultFilePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "history.txt"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := writeDefaultFiles(dir)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("writeDefaultFiles = %v, want a not-a-regular-file error", err)
	}
}

// Creating an environment twice reports ErrAlreadyExists the second time and
// leaves the first intact.
func TestCreateTwiceReportsAlreadyExists(t *testing.T) {
	mgr := NewManager(t.TempDir(), "")
	if err := mgr.Create("once"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Create("once"); err == nil || !strings.Contains(err.Error(), ErrAlreadyExists.Error()) {
		t.Fatalf("second Create = %v, want ErrAlreadyExists", err)
	}
	if !mgr.Exists("once") {
		t.Fatal("the environment is gone after the second Create")
	}
}
