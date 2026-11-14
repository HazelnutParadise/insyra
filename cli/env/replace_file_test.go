package env

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// replaceFile replaces an existing target. The retry on a Windows sharing
// refusal cannot be provoked off Windows; the Windows CI job running
// TestConcurrentSavesNeverLoseAVariable is the check for it.
func TestReplaceFileReplacesTheTarget(t *testing.T) {
	oldOS, oldFor := renameRetryOS, renameRetryFor
	renameRetryOS, renameRetryFor = "windows", 2*time.Second
	defer func() { renameRetryOS, renameRetryFor = oldOS, oldFor }()

	dir := t.TempDir()
	final, tmp := filepath.Join(dir, "state.json"), filepath.Join(dir, "state.json.1.tmp")
	if err := os.WriteFile(final, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmp, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(tmp, final); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(final); string(got) != "new" {
		t.Errorf("state.json = %q, want new", got)
	}
}

func TestReplaceFileReportsAnErrorThatIsNotAShare(t *testing.T) {
	dir := t.TempDir()
	if err := replaceFile(filepath.Join(dir, "missing.tmp"), filepath.Join(dir, "state.json")); err == nil {
		t.Error("want an error for a missing source")
	}
}

// readStateRetrying reads an existing file and reports a missing one at once. The
// retry on a Windows sharing violation cannot be provoked off Windows; the
// Windows CI job running TestConcurrentSavesNeverLoseAVariable is the check.
func TestReadStateRetryingReadsAndReportsAMissingFile(t *testing.T) {
	oldOS := renameRetryOS
	renameRetryOS = "windows"
	defer func() { renameRetryOS = oldOS }()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := readStateRetrying(path); err != nil || string(got) != "{}" {
		t.Fatalf("readStateRetrying = %q, %v", got, err)
	}
	start := time.Now()
	if _, err := readStateRetrying(filepath.Join(dir, "missing.json")); !os.IsNotExist(err) {
		t.Fatalf("a missing file: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Error("a missing file was retried")
	}
}
