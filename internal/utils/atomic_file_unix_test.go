//go:build !windows

package utils

import (
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestWriteFileAtomicallyAppliesUmaskToNewFiles(t *testing.T) {
	old := syscall.Umask(0o027)
	defer syscall.Umask(old)

	path := filepath.Join(t.TempDir(), "new-file")
	if err := WriteFileAtomically(path, func(w io.Writer) error { _, err := w.Write([]byte("new")); return err }); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o666&^0o027); got != want {
		t.Fatalf("new file mode=%o, want 0666 &^ 0027 = %o", got, want)
	}
}

func TestWriteFileAtomicallyKeepsAnExistingFilesModeUnderUmask(t *testing.T) {
	old := syscall.Umask(0o022)
	defer syscall.Umask(old)

	path := filepath.Join(t.TempDir(), "existing")
	if err := os.WriteFile(path, []byte("old"), 0o664); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o664); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomically(path, func(w io.Writer) error { _, err := w.Write([]byte("new")); return err }); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o664 {
		t.Fatalf("existing file mode=%o, want 0664 kept through the umask", got)
	}
}
