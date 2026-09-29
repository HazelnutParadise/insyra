package py

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra"
)

// useFakePython marks the environment ready with the test binary as its
// interpreter, answering as fakePython does for mode, points the environment
// directory at a temporary one, and restores the package when the test ends.
func useFakePython(t *testing.T, mode string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pyInitMu.Lock()
	wasInit, wasPy, wasDir := isPyEnvInit, pyPath, absInstallDir
	isPyEnvInit, pyPath, absInstallDir = true, self, t.TempDir()
	pyInitMu.Unlock()
	t.Cleanup(func() {
		pyInitMu.Lock()
		isPyEnvInit, pyPath, absInstallDir = wasInit, wasPy, wasDir
		pyInitMu.Unlock()
	})
	t.Setenv(fakePythonEnv, mode)
	if runtime.GOOS != "windows" {
		t.Setenv("TMPDIR", "") // os.TempDir() is then /tmp, short enough for the IPC socket
	}
}

// PY-2 of #255: RunCode bound the result into an `any`, so its type was only
// checked at run time.
func TestRunDecodesIntoTheTypeAskedFor(t *testing.T) {
	useFakePython(t, `result:{"name":"insyra","count":3}`)
	type answer struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	got, err := Run[answer](context.Background(), `insyra.Return({"name": "insyra", "count": 3})`)
	if err != nil {
		t.Fatal(err)
	}
	if got != (answer{Name: "insyra", Count: 3}) {
		t.Errorf("Run returned %+v", got)
	}
}

func TestRunDecodesADataFrame(t *testing.T) {
	useFakePython(t, `result:{"_insyra_type":"datatable","data":[[1,2],[3,4]],"columns":["a","b"],"index":["r1","r2"]}`)
	dt, err := Run[*insyra.DataTable](context.Background(), "insyra.Return(df)")
	if err != nil {
		t.Fatal(err)
	}
	if rows, cols := dt.Size(); rows != 2 || cols != 2 {
		t.Errorf("the table is %d x %d, want 2 x 2", rows, cols)
	}
	if got := dt.ColNames(); !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("the columns are %q, want a and b", got)
	}
}

func TestRunReturnsTheZeroValueAndThePythonError(t *testing.T) {
	useFakePython(t, "error:name 'df' is not defined")
	got, err := Run[map[string]int](context.Background(), "insyra.Return(df)")
	if err == nil || !strings.Contains(err.Error(), "name 'df' is not defined") {
		t.Errorf("a Python error gave %v", err)
	}
	if got != nil {
		t.Errorf("a failed run returned %v, want the zero value", got)
	}
}

func TestRunFillsThePlaceholders(t *testing.T) {
	useFakePython(t, "script")
	script, err := Run[string](context.Background(), "x = $v1", []int{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "x = [1, 2]") {
		t.Errorf("the script Python ran does not contain x = [1, 2]:\n%s", script)
	}
}

func TestRunStopsWithItsContext(t *testing.T) {
	useFakePython(t, "sleep")
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := Run[int](ctx, "import time\ntime.sleep(60)")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a run past its deadline gave %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("the run took %v to stop", elapsed)
	}
}

// contextForms calls each function that takes a context with ctx.
func contextForms(t *testing.T, ctx context.Context) map[string]error {
	t.Helper()
	file := filepath.Join(t.TempDir(), "code.py")
	if err := os.WriteFile(file, []byte("insyra.Return(1)"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, runErr := Run[int](ctx, "insyra.Return(1)")
	return map[string]error{
		"Run":                 runErr,
		"RunCodeContext":      RunCodeContext(ctx, nil, "insyra.Return(1)"),
		"RunCodefContext":     RunCodefContext(ctx, nil, "insyra.Return($v1)", 1),
		"RunFileContext":      RunFileContext(ctx, nil, file),
		"RunFilefContext":     RunFilefContext(ctx, nil, file),
		"PipInstallContext":   PipInstallContext(ctx, "numpy"),
		"PipUninstallContext": PipUninstallContext(ctx, "numpy"),
	}
}

// A nil context made exec.CommandContext panic in a goroutine the runner had
// started, which ended the program.
func TestContextFormsRefuseANilContext(t *testing.T) {
	useFakePython(t, "sleep")
	var nilCtx context.Context
	for name, err := range contextForms(t, nilCtx) {
		if !errors.Is(err, errNilContext) {
			t.Errorf("%s with a nil context gave %v, want errNilContext", name, err)
		}
	}
}

func TestContextFormsStartNothingWhenTheContextIsDone(t *testing.T) {
	useFakePython(t, "sleep") // a run that started would take a minute
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	for name, err := range contextForms(t, ctx) {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("%s with a cancelled context gave %v, want context.Canceled", name, err)
		}
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("the calls took %v; something started", elapsed)
	}
}

func TestPipInstallContextStopsWithItsContext(t *testing.T) {
	useFakePython(t, "sleep")
	uv := filepath.Join(t.TempDir(), uvExecutableName())
	fakeUVAt(t, uv, "sleep")
	pyInitMu.Lock()
	wasUV := uvPath
	uvPath = uv
	pyInitMu.Unlock()
	t.Cleanup(func() {
		pyInitMu.Lock()
		uvPath = wasUV
		pyInitMu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := PipInstallContext(ctx, "numpy"); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("an install past its deadline gave %v, want context.DeadlineExceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("the install took %v to stop", elapsed)
	}
}

func TestRunCodeWithTimeoutKeepsItsMeaning(t *testing.T) {
	useFakePython(t, "sleep")
	if err := RunCodeWithTimeout(300*time.Millisecond, nil, "import time\ntime.sleep(60)"); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("RunCodeWithTimeout gave %v, want context.DeadlineExceeded", err)
	}

	f, err := parser.ParseFile(token.NewFileSet(), "py.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var doc string
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == "RunCodeWithTimeout" {
			doc = strings.Join(strings.Fields(fn.Doc.Text()), " ")
		}
	}
	for _, want := range []string{"Deprecated: use RunCodeContext", "Removed in the release after the one that deprecated it."} {
		if !strings.Contains(doc, want) {
			t.Errorf("RunCodeWithTimeout's doc %q does not say %q", doc, want)
		}
	}
}

func TestThePlainFormsStillBindIntoOut(t *testing.T) {
	useFakePython(t, `result:{"v":1}`)
	var out map[string]int
	if err := RunCode(&out, `insyra.Return({"v": 1})`); err != nil {
		t.Fatal(err)
	}
	if out["v"] != 1 {
		t.Errorf("RunCode bound %v", out)
	}
}

// A process that fails reports its error and then closes processDone, so both
// are ready together. The runner used to pick either, and half the time
// returned no error for a Python that was killed or crashed.
func TestWaitForResultPrefersTheProcessError(t *testing.T) {
	for i := 0; i < 100; i++ {
		execErr := make(chan error, 1)
		processDone := make(chan struct{})
		execErr <- errors.New("signal: killed")
		close(processDone)
		got := waitForResult(fmt.Sprintf("no-result-%d", i), processDone, execErr)
		if got[1] == nil {
			t.Fatalf("run %d: a failed process gave %v, want its error", i, got)
		}
	}
}

// Run[insyra.DataTable] used to compile, run Python, and only then fail with
// advice about **insyra.DataTable that a Run caller cannot follow.
func TestRunRefusesAValueTableOrListBeforeStarting(t *testing.T) {
	useFakePython(t, "sleep") // a run that started would take a minute
	start := time.Now()
	if _, err := Run[insyra.DataTable](context.Background(), "insyra.Return(df)"); err == nil || !strings.Contains(err.Error(), "Run[*insyra.DataTable]") {
		t.Errorf("Run[insyra.DataTable] gave %v, want an error pointing to Run[*insyra.DataTable]", err)
	}
	if _, err := Run[insyra.DataList](context.Background(), "insyra.Return(s)"); err == nil || !strings.Contains(err.Error(), "Run[*insyra.DataList]") {
		t.Errorf("Run[insyra.DataList] gave %v, want an error pointing to Run[*insyra.DataList]", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("Run started Python before refusing the type (%v)", elapsed)
	}
}

// A result Python delivered still counts when the process then failed, for
// instance because its context ended between the two.
func TestWaitForResultKeepsADeliveredResult(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("delivered-%d", i)
		resultStore.Store(id, [2]any{float64(42), nil})
		execErr := make(chan error, 1)
		processDone := make(chan struct{})
		execErr <- errors.New("signal: killed")
		close(processDone)
		got := waitForResult(id, processDone, execErr)
		if got[0] != float64(42) || got[1] != nil {
			t.Fatalf("run %d: a delivered result came back as %v", i, got)
		}
		if _, left := resultStore.Load(id); left {
			t.Fatalf("run %d: the result was left in the store", i)
		}
	}
}

// With the environment not ready, a call whose context is already done must
// not start the setup, which would write the pinned project into the
// environment directory before uv refused to start.
func TestContextFormsDoNotStartTheSetupWhenTheContextIsDone(t *testing.T) {
	envDir, log := useTempEnvironment(t, "ok")
	if runtime.GOOS != "windows" {
		t.Setenv("TMPDIR", "") // os.TempDir() is then /tmp, short enough for the IPC socket
	}
	if err := pyEnvInit(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{envMarker, "pyproject.toml"} {
		if err := os.Remove(filepath.Join(envDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	markNotReady()
	runs := len(uvRuns(t, log))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for name, err := range contextForms(t, ctx) {
		if !errors.Is(err, context.Canceled) {
			t.Errorf("%s with a cancelled context gave %v, want context.Canceled", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(envDir, "pyproject.toml")); !errors.Is(err, os.ErrNotExist) {
		t.Error("a call with a cancelled context started the environment setup")
	}
	if got := len(uvRuns(t, log)); got != runs {
		t.Errorf("uv ran %d more times", got-runs)
	}
}
