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
