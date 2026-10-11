//go:build !windows

package utils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicallyLongName(t *testing.T) {
	path := filepath.Join(t.TempDir(), strings.Repeat("x", 230))
	if err := WriteFileAtomically(path, func(w io.Writer) error { _, err := w.Write([]byte("ok")); return err }); err != nil {
		t.Fatal(err)
	}
}

func TestWriteFileAtomicallyDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "not-created")
	link := filepath.Join(dir, "link")
	if err := os.Symlink(filepath.Base(target), link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomically(link, func(w io.Writer) error { _, err := w.Write([]byte("ok")); return err }); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil || string(got) != "ok" {
		t.Fatalf("target=%q err=%v", got, err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link was not preserved: info=%v err=%v", info, err)
	}
}

func TestWriteFileAtomicallyPreservesExistingPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.txt")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomically(path, func(w io.Writer) error { _, err := w.Write([]byte("new")); return err }); err != nil {
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

// Writing through a symbolic link replaces the file it points to and leaves
// the link a link.
func TestWriteFileAtomicallyKeepsASymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.bin")
	link := filepath.Join(dir, "link.bin")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := WriteFileAtomically(link, func(w io.Writer) error {
		_, err := w.Write([]byte("new"))
		return err
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link became a regular file")
	}
	if got, _ := os.ReadFile(target); string(got) != "new" {
		t.Errorf("target holds %q, want %q", got, "new")
	}
}
