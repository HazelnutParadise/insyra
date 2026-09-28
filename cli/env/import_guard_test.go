package env

import (
	"os"
	"path/filepath"
	"testing"
)

// `env import` without --force must not overwrite an environment that holds
// something. A file that exists but cannot be read is not evidence that the
// environment is empty, so the guard in front of the overwrite fails closed.

// exportOne writes an export of a one-variable environment and returns its path.
func exportOne(t *testing.T) string {
	t.Helper()
	if err := Create("source"); err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := SaveState("source", map[string]any{"new": 1}); err != nil {
		t.Fatalf("save source: %v", err)
	}
	out := filepath.Join(t.TempDir(), "source-export.json")
	if err := Export("source", out); err != nil {
		t.Fatalf("export source: %v", err)
	}
	return out
}

func TestImportRefusesATargetItCannotRead(t *testing.T) {
	// A directory where the file should be cannot be read as a file on any
	// platform, which a permission bit cannot promise on Windows.
	for _, name := range []string{"state.json", "history.txt", "config.json"} {
		t.Run(name, func(t *testing.T) {
			setupTempHome(t)
			exportFile := exportOne(t)
			if err := Create("target"); err != nil {
				t.Fatalf("create target: %v", err)
			}
			envPath, err := ResolveEnvPath("target")
			if err != nil {
				t.Fatal(err)
			}
			blocker := filepath.Join(envPath, name)
			_ = os.RemoveAll(blocker)
			if err := os.Mkdir(blocker, 0o755); err != nil {
				t.Fatal(err)
			}

			// Ask the guard itself. Asking Import is not enough: with a
			// directory in the way the import also fails later, when it tries
			// to write, so an error from Import proves nothing about the guard.
			empty, err := defaultManager.isEnvironmentEmpty("target")
			if err == nil {
				t.Fatalf("the guard answered empty=%v with no error for a target whose %s could not be read", empty, name)
			}
			if _, err := Import(exportFile, "target", false); err == nil {
				t.Fatalf("import without --force went ahead although %s could not be read", name)
			}
		})
	}
}

func TestImportFillsATargetWithNothingInIt(t *testing.T) {
	setupTempHome(t)
	exportFile := exportOne(t)
	if err := Create("target"); err != nil {
		t.Fatalf("create target: %v", err)
	}
	if _, err := Import(exportFile, "target", false); err != nil {
		t.Fatalf("import into an empty target without --force: %v", err)
	}
}
