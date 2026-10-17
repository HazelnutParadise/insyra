package utils

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicallyLongNameAndPanicCleanup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, strings.Repeat("a", 230))
	if err := WriteFileAtomically(path, func(w io.Writer) error { _, err := w.Write([]byte("ok")); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		_ = WriteFileAtomically(path, func(_ io.Writer) error { panic("boom") })
	}()
	if !panicked {
		t.Fatal("callback panic was swallowed")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".insyra-") {
			t.Fatalf("temporary file remains: %s", entry.Name())
		}
	}
}

func TestWriteFileAtomicallyFollowsSymlinkAndKeepsMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomically(link, func(w io.Writer) error { _, err := w.Write([]byte("new")); return err }); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(target); string(got) != "new" {
		t.Fatalf("target=%q", got)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link replaced: %v", err)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
	dangling := filepath.Join(dir, "dangling")
	if err := os.Symlink("created", dangling); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomically(dangling, func(w io.Writer) error { _, err := w.Write([]byte("created")); return err }); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "created")); string(got) != "created" {
		t.Fatalf("dangling target=%q", got)
	}
}
