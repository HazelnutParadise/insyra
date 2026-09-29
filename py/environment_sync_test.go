package py

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeUVAt copies the test binary to path, so running path runs fakeUV with
// mode, and returns the file the fake logs its arguments to.
func fakeUVAt(t *testing.T, path, mode string) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeExecutable(path, data); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "uv.log")
	t.Setenv(fakeUVEnv, mode)
	t.Setenv(fakeUVLogEnv, log)
	return log
}

// uvRuns returns the arguments of each run the fake uv logged.
func uvRuns(t *testing.T, log string) []string {
	t.Helper()
	b, err := os.ReadFile(log)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func TestSyncEnvironmentRunsUVWithThePins(t *testing.T) {
	envDir := filepath.Join(t.TempDir(), "env")
	uv := filepath.Join(t.TempDir(), uvExecutableName())
	log := fakeUVAt(t, uv, "ok")

	if err := syncEnvironment(context.Background(), uv, envDir); err != nil {
		t.Fatal(err)
	}
	python, err := pinnedPythonVersion()
	if err != nil {
		t.Fatal(err)
	}
	want := "sync --frozen --inexact --managed-python --python " + python
	if runs := uvRuns(t, log); len(runs) != 1 || runs[0] != want {
		t.Errorf("uv ran %q, want one run of %q", runs, want)
	}
	for name, embedded := range map[string][]byte{"pyproject.toml": envPyproject, "uv.lock": envLock} {
		got, err := os.ReadFile(filepath.Join(envDir, name))
		if err != nil || !bytes.Equal(got, embedded) {
			t.Errorf("%s in the environment is not the embedded file (%v)", name, err)
		}
	}
	if !envInSync(envDir) {
		t.Error("the environment is not in sync after a successful sync")
	}
}

func TestEnvironmentIsOutOfSyncWhenItsMarkerOrInterpreterDiffers(t *testing.T) {
	envDir := filepath.Join(t.TempDir(), "env")
	uv := filepath.Join(t.TempDir(), uvExecutableName())
	fakeUVAt(t, uv, "ok")
	if err := syncEnvironment(context.Background(), uv, envDir); err != nil {
		t.Fatal(err)
	}

	marker := filepath.Join(envDir, envMarker)
	if err := os.WriteFile(marker, []byte("another pin set\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if envInSync(envDir) {
		t.Error("an environment synced to other pins counts as in sync")
	}

	if err := os.WriteFile(marker, []byte(pinFingerprint()+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(venvPython(filepath.Join(envDir, ".venv"))); err != nil {
		t.Fatal(err)
	}
	if envInSync(envDir) {
		t.Error("an environment without its interpreter counts as in sync")
	}
}

func TestAFailedSyncLeavesNoMarker(t *testing.T) {
	envDir := filepath.Join(t.TempDir(), "env")
	uv := filepath.Join(t.TempDir(), uvExecutableName())
	fakeUVAt(t, uv, "fail")

	err := syncEnvironment(context.Background(), uv, envDir)
	if err == nil || !strings.Contains(err.Error(), "fake uv was told to fail") {
		t.Fatalf("a failing uv gave %v, want its error output", err)
	}
	if _, serr := os.Stat(filepath.Join(envDir, envMarker)); !errors.Is(serr, os.ErrNotExist) {
		t.Error("a failed sync left a marker")
	}
	if envInSync(envDir) {
		t.Error("a failed sync counts as in sync")
	}
}

func TestSyncEnvironmentStopsWithItsContext(t *testing.T) {
	envDir := filepath.Join(t.TempDir(), "env")
	uv := filepath.Join(t.TempDir(), uvExecutableName())
	fakeUVAt(t, uv, "sleep")

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := syncEnvironment(ctx, uv, envDir)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a sync past its deadline gave %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("the sync took %v to stop", elapsed)
	}
}

// The user's own uv settings that would make the sync fail do not reach it:
// any UV_PYTHON_PREFERENCE conflicts with --managed-python, and
// UV_PYTHON_DOWNLOADS=never stops uv fetching the pinned Python.
func TestSyncEnvironmentKeepsConflictingUVSettingsAway(t *testing.T) {
	envDir := filepath.Join(t.TempDir(), "env")
	uv := filepath.Join(t.TempDir(), uvExecutableName())
	fakeUVAt(t, uv, "ok")
	t.Setenv("UV_PYTHON_PREFERENCE", "system")
	t.Setenv("UV_PYTHON_DOWNLOADS", "never")
	t.Setenv("UV_PROJECT_ENVIRONMENT", filepath.Join(t.TempDir(), "elsewhere"))

	if err := syncEnvironment(context.Background(), uv, envDir); err != nil {
		t.Fatalf("the sync failed under the user's uv settings: %v", err)
	}
	if !envInSync(envDir) {
		t.Error("UV_PROJECT_ENVIRONMENT moved the virtual environment away from the environment directory")
	}
}
