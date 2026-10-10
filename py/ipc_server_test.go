package py

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/py/internal/ipc"
)

// tooLongTempDir is a TMPDIR under which no Unix socket can be created: the
// directory name alone is longer than sun_path (104 bytes on macOS, 108 on
// Linux), so net.Listen fails with EINVAL.
var tooLongTempDir = "/" + strings.Repeat("x", 200)

// environmentMarkedReady makes pyEnvInit report the environment as prepared,
// with an interpreter path that does not exist, so a test that reaches it
// fails at once instead of downloading anything.
func environmentMarkedReady(t *testing.T) {
	t.Helper()
	pyInitMu.Lock()
	wasInit, wasPath := isPyEnvInit, pyPath
	isPyEnvInit, pyPath = true, filepath.Join(t.TempDir(), "no-such-python")
	pyInitMu.Unlock()
	t.Cleanup(func() {
		pyInitMu.Lock()
		isPyEnvInit, pyPath = wasInit, wasPath
		pyInitMu.Unlock()
	})
}

// PY-1 of #254: when the IPC server could not listen, the Python code was
// still generated with its address, Python failed to connect, and RunCode
// returned "exit status 1" while the reason was only in the log.
//
// The runners take the IPC server before they prepare the environment, so a
// server that cannot open costs no download. The test holds the setup lock: a
// runner that went to the environment first would wait on it.
func TestRunCodeReturnsTheIPCStartFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("named pipe names do not depend on TMPDIR")
	}
	environmentMarkedReady(t)
	t.Setenv("TMPDIR", tooLongTempDir)

	pyInitMu.Lock()
	results := make(chan error, 2)
	go func() { results <- RunCodeContext(context.Background(), nil, "insyra.Return(1)") }()
	go func() { results <- RunCode(nil, "insyra.Return(1)") }()
	var errs []error
	timeout := time.After(10 * time.Second)
	for len(errs) < 2 {
		select {
		case err := <-results:
			errs = append(errs, err)
		case <-timeout:
			t.Error("a runner went to the environment setup before taking the IPC server")
			pyInitMu.Unlock()
			// The environment is still marked ready, so the waiting runners
			// finish now; wait for them before the test's cleanups run.
			for len(errs) < 2 {
				errs = append(errs, <-results)
			}
			return
		}
	}
	pyInitMu.Unlock()

	for _, err := range errs {
		if err == nil {
			t.Fatal("a runner succeeded without an IPC server")
		}
		if !errors.Is(err, syscall.EINVAL) {
			t.Errorf("the error %q does not carry the failed listen", err)
		}
		if !strings.Contains(err.Error(), "IPC server") {
			t.Errorf("the error %q does not say what failed", err)
		}
	}
}

// The start used to run once per process, so one failure failed every later
// call too.
func TestIPCServerStartIsTriedAgain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("named pipe names do not depend on TMPDIR")
	}
	t.Setenv("TMPDIR", tooLongTempDir)
	if _, err := acquireIPCServer(); err == nil {
		releaseIPCServer()
		t.Fatal("the IPC server opened under a TMPDIR too long for a socket path")
	}

	t.Setenv("TMPDIR", "") // os.TempDir() is then /tmp, short enough for a socket
	addr, err := acquireIPCServer()
	if err != nil {
		t.Fatalf("the next start failed too: %v", err)
	}
	defer releaseIPCServer()
	if addr == "" {
		t.Error("the IPC server opened with an empty address")
	}
}

// The socket file used to stay in the temp directory after the process ended.
func TestIPCSocketExistsOnlyWhileARunNeedsIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a named pipe leaves no file behind")
	}
	t.Setenv("TMPDIR", "")

	first, err := acquireIPCServer()
	if err != nil {
		t.Fatal(err)
	}
	second, err := acquireIPCServer()
	if err != nil {
		releaseIPCServer()
		t.Fatal(err)
	}
	if first != second {
		t.Errorf("two runs at once got different servers: %q and %q", first, second)
	}

	releaseIPCServer()
	if _, err := os.Stat(first); err != nil {
		t.Errorf("the socket went away while a run still needed it: %v", err)
	}
	releaseIPCServer()
	if _, err := os.Stat(first); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the socket %s is still there after the last run: %v", first, err)
	}
}

func TestIPCServerStoresAResult(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Setenv("TMPDIR", "")
	}
	addr, err := acquireIPCServer()
	if err != nil {
		t.Fatal(err)
	}
	defer releaseIPCServer()

	conn, err := ipc.Dial(addr)
	if err != nil {
		t.Fatalf("dialling the IPC server: %v", err)
	}
	defer func() { _ = conn.Close() }()

	const id = "ipc-server-test"
	if err := ipc.WriteMessage(conn, []byte(`{"execution_id":"`+id+`","data":[42,null]}`)); err != nil {
		t.Fatalf("sending the result: %v", err)
	}
	if _, err := ipc.ReadMessage(conn); err != nil {
		t.Fatalf("reading the acknowledgement: %v", err)
	}
	got, ok := resultStore.LoadAndDelete(id)
	if !ok {
		t.Fatal("the result was not stored")
	}
	if r := got.([2]any); r[0] != float64(42) || r[1] != nil {
		t.Errorf("stored %v, want [42 <nil>]", r)
	}
}

// failingListener is a listener whose Accept always fails with an error that
// is not a timeout, the way a listener out of file descriptors fails.
type failingListener struct {
	closed atomic.Bool
}

func (l *failingListener) Accept() (net.Conn, error) {
	return nil, errors.New("accept failed for the test")
}

func (l *failingListener) Close() error {
	l.closed.Store(true)
	return nil
}

func (l *failingListener) Addr() net.Addr { return &net.UnixAddr{Name: "test", Net: "unix"} }

// An accept loop that stopped used to leave its listener open, so a Python
// process could still connect, send its result and wait for an
// acknowledgement that never came.
func TestAnAcceptLoopThatStopsClosesItsListener(t *testing.T) {
	ln := &failingListener{}
	done := make(chan struct{})
	go func() {
		acceptIPC(ln)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the accept loop kept going after an error that is not a timeout")
	}
	if !ln.closed.Load() {
		t.Error("the accept loop stopped without closing its listener")
	}
}
