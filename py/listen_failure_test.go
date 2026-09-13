package py

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const listenFailureChildEnv = "INSYRA_PY_LISTEN_FAILURE_CHILD"

// A failed IPC listen used to end the whole program through LogFatal. It now
// logs a warning, leaves the server down and still releases anyone waiting
// for the address. The server state is process-wide (sync.Once), so the check
// runs in a child copy of the test binary whose TMPDIR points at a missing
// directory, which makes the Unix-socket listen fail.
func TestIPCListenFailureDoesNotEndProgram(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the IPC address is a named pipe on Windows; TMPDIR does not steer it")
	}
	if os.Getenv(listenFailureChildEnv) == "1" {
		go startServer()
		done := make(chan string, 1)
		go func() { done <- getIPCAddress() }()
		select {
		case <-done:
			fmt.Println("CHILD_SURVIVED")
		case <-time.After(10 * time.Second):
			fmt.Println("CHILD_BLOCKED")
			os.Exit(3)
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestIPCListenFailureDoesNotEndProgram$", "-test.v")
	cmd.Env = append(os.Environ(), listenFailureChildEnv+"=1", "TMPDIR="+filepath.Join(t.TempDir(), "missing"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child process ended with %v:\n%s", err, out)
	}
	if !strings.Contains(string(out), "CHILD_SURVIVED") {
		t.Fatalf("child did not get past the failed listen:\n%s", out)
	}
	if !strings.Contains(string(out), "Failed to start IPC server") {
		t.Fatalf("the failed listen was not logged:\n%s", out)
	}
}
