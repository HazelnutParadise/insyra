package py

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// useTempEnvironment points the package at an environment directory under a
// fresh temporary directory, puts a fake uv with mode where the setup runs uv
// from (no fake when mode is empty), and restores the package when the test
// ends. It returns the environment directory and the fake's log.
func useTempEnvironment(t *testing.T, mode string) (envDir, log string) {
	t.Helper()
	envDir = filepath.Join(t.TempDir(), ".insyra_env", "py25c_test")
	pyInitMu.Lock()
	wasDir, wasInit, wasPy, wasUV := absInstallDir, isPyEnvInit, pyPath, uvPath
	absInstallDir, isPyEnvInit = envDir, false
	pyInitMu.Unlock()
	t.Cleanup(func() {
		pyInitMu.Lock()
		absInstallDir, isPyEnvInit, pyPath, uvPath = wasDir, wasInit, wasPy, wasUV
		pyInitMu.Unlock()
	})
	if mode == "" {
		return envDir, ""
	}
	version, err := pinnedUVVersion()
	if err != nil {
		t.Fatal(err)
	}
	return envDir, fakeUVAt(t, uvBinaryPath(envDir, version), mode)
}

// markNotReady makes the next pyEnvInit behave like the first one in a new
// process.
func markNotReady() {
	pyInitMu.Lock()
	isPyEnvInit = false
	pyInitMu.Unlock()
}

func TestSetupRunsUVOncePerPinSet(t *testing.T) {
	envDir, log := useTempEnvironment(t, "ok")
	if err := pyEnvInit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runs := uvRuns(t, log); len(runs) != 1 {
		t.Fatalf("the first setup ran uv %d times, want once: %q", len(runs), runs)
	}
	if want := venvPython(filepath.Join(envDir, ".venv")); pyPath != want {
		t.Errorf("pyPath is %s, want %s", pyPath, want)
	}
	version, err := pinnedUVVersion()
	if err != nil {
		t.Fatal(err)
	}
	if want := uvBinaryPath(envDir, version); uvPath != want {
		t.Errorf("uvPath is %s, want the pinned uv at %s", uvPath, want)
	}

	markNotReady()
	if err := pyEnvInit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runs := uvRuns(t, log); len(runs) != 1 {
		t.Errorf("an environment already in sync ran uv again: %q", runs)
	}
}

// An environment built by an older insyra has no marker, and is brought to
// the pins the first time it is used.
func TestSetupSyncsAnEnvironmentBuiltBefore(t *testing.T) {
	envDir, log := useTempEnvironment(t, "ok")
	python := venvPython(filepath.Join(envDir, ".venv"))
	if err := os.MkdirAll(filepath.Dir(python), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(python, nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := pyEnvInit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runs := uvRuns(t, log); len(runs) != 1 {
		t.Errorf("an unmarked environment ran uv %d times, want one sync", len(runs))
	}
}

// SEC-10 of #289: the old setup marked the environment ready before its
// first step, so a setup that failed part-way never ran again.
func TestAFailedSetupIsRunAgain(t *testing.T) {
	useTempEnvironment(t, "fail")
	if err := pyEnvInit(context.Background()); err == nil {
		t.Fatal("a failing sync made the setup succeed")
	}
	pyInitMu.Lock()
	ready := isPyEnvInit
	pyInitMu.Unlock()
	if ready {
		t.Fatal("a failed setup marked the environment ready")
	}
	t.Setenv(fakeUVEnv, "ok")
	if err := pyEnvInit(context.Background()); err != nil {
		t.Errorf("the next setup failed too: %v", err)
	}
}

func TestRunCodeContextBoundsTheSetup(t *testing.T) {
	useTempEnvironment(t, "sleep")
	if runtime.GOOS != "windows" {
		t.Setenv("TMPDIR", "") // os.TempDir() is then /tmp, short enough for the IPC socket
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	err := RunCodeContext(ctx, nil, "insyra.Return(1)")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a setup past the call's deadline gave %v, want context.DeadlineExceeded", err)
	}
}

// #254: ReinstallPyEnv deletes the whole environment directory, packages
// added with PipInstall included, and keeps the pinned uv.
func TestReinstallPyEnvDeletesTheEnvironmentDirectory(t *testing.T) {
	envDir, log := useTempEnvironment(t, "ok")
	if err := pyEnvInit(context.Background()); err != nil {
		t.Fatal(err)
	}
	added := filepath.Join(envDir, ".venv", "lib", "added-with-pipinstall")
	if err := os.MkdirAll(filepath.Dir(added), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(added, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ReinstallPyEnv(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(added); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s survived the reinstall", added)
	}
	if runs := uvRuns(t, log); len(runs) != 2 {
		t.Errorf("uv ran %d times, want a second sync for the reinstall", len(runs))
	}
	if !envInSync(envDir) {
		t.Error("the reinstalled environment is not in sync")
	}
	version, err := pinnedUVVersion()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(uvBinaryPath(envDir, version)); err != nil {
		t.Errorf("the reinstall removed the pinned uv: %v", err)
	}
}

// Waiting for another call's setup used to ignore the waiting call's context,
// so a call with a one-second deadline waited for a whole download.
func TestSetupStopsWaitingWhenItsContextEnds(t *testing.T) {
	useTempEnvironment(t, "ok")
	pyInitMu.Lock() // another call is preparing the environment
	defer pyInitMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := pyEnvInit(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("waiting on another setup past the deadline gave %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("pyEnvInit waited %v", elapsed)
	}
}

// A delete that failed part-way used to leave the environment marked ready,
// so the next call ran what was left of it.
func TestAReinstallThatCannotDeleteLeavesTheEnvironmentNotReady(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not stop a delete on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root deletes regardless of permissions")
	}
	envDir, _ := useTempEnvironment(t, "ok")
	if err := pyEnvInit(context.Background()); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(envDir, ".venv", "lib", "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	if err := ReinstallPyEnv(); err == nil {
		t.Fatal("ReinstallPyEnv succeeded although part of the environment could not be deleted")
	}
	pyInitMu.Lock()
	ready := isPyEnvInit
	pyInitMu.Unlock()
	if ready {
		t.Error("a reinstall that could not delete the environment left it marked ready")
	}
}

// TestPinnedEnvironmentEndToEnd builds the real environment from nothing in
// a temporary directory. It downloads uv, Python and every package, so it
// needs the network and minutes, and runs only with INSYRA_PY_E2E=1. Point
// UV_CACHE_DIR and UV_PYTHON_INSTALL_DIR at temporary directories too if the
// run should leave nothing in your own uv directories.
func TestPinnedEnvironmentEndToEnd(t *testing.T) {
	if os.Getenv("INSYRA_PY_E2E") != "1" {
		t.Skip("set INSYRA_PY_E2E=1 to build the real Python environment")
	}
	useTempEnvironment(t, "")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	var dt *insyra.DataTable
	if err := RunCodeContext(ctx, &dt, `insyra.Return(pd.DataFrame({"a": [1, 2], "b": [3.5, 4.5]}))`); err != nil {
		t.Fatal(err)
	}
	if rows, cols := dt.Size(); rows != 2 || cols != 2 {
		t.Errorf("the DataFrame came back %d x %d, want 2 x 2", rows, cols)
	}

	// A DataFrame inside a dict used to fail in json.dumps on the Python side.
	type scored struct {
		Table *insyra.DataTable `json:"table"`
		Score float64           `json:"score"`
	}
	got, err := Run[scored](ctx, `insyra.Return({"table": pd.DataFrame({"a": [1, 2], "b": [3.5, 4.5]}), "score": 7})`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Table == nil || got.Score != 7 {
		t.Fatalf("a dict holding a DataFrame came back as %+v", got)
	}
	if rows, cols := got.Table.Size(); rows != 2 || cols != 2 {
		t.Errorf("the nested DataFrame came back %d x %d, want 2 x 2", rows, cols)
	}

	empty, err := Run[*insyra.DataTable](ctx, `insyra.Return(pd.DataFrame(columns=["a", "b"]))`)
	if err != nil {
		t.Fatal(err)
	}
	if names := empty.ColNames(); len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("an empty DataFrame came back with columns %q, want a and b", names)
	}

	// A NaN or an infinity in a result came back as nil with no error.
	withNaN, err := Run[*insyra.DataTable](ctx, `insyra.Return(pd.DataFrame({"a": [1.0, float("nan")], "b": [float("inf"), -float("inf")]}))`)
	if err != nil {
		t.Fatal(err)
	}
	if withNaN == nil || !math.IsNaN(asFloat(withNaN.GetElementByNumberIndex(1, 0))) || !math.IsInf(asFloat(withNaN.GetElementByNumberIndex(0, 1)), 1) || !math.IsInf(asFloat(withNaN.GetElementByNumberIndex(1, 1)), -1) {
		t.Errorf("a DataFrame with a NaN and infinities came back as %v", withNaN)
	}
	if got, err := Run[any](ctx, `insyra.Return(10**400)`); err == nil || !strings.Contains(err.Error(), "float64") {
		t.Errorf("an integer too large for a float64 gave %v, %v; want an error", got, err)
	}

	var version string
	if err := RunCodeContext(ctx, &version, "import platform\ninsyra.Return(platform.python_version())"); err != nil {
		t.Fatal(err)
	}
	if want, _ := pinnedPythonVersion(); version != want {
		t.Errorf("Python is %s, want the pinned %s", version, want)
	}

	installed, err := PipList()
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range pinnedDependencies(t) {
		if installed[name] != want {
			t.Errorf("%s is %q, want the pinned %s", name, installed[name], want)
		}
	}
}
