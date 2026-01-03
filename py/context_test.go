package py

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRunCodeContextCancel verifies that a short timeout cancels the Python execution
// and the Python code does not complete its final side-effect (file write).
func TestRunCodeContextCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires Python runtime in short mode")
	}

	tmp := t.TempDir()
	fname := filepath.Join(tmp, "test_ctx_cancel.txt")
	code := fmt.Sprintf(`
import time
for i in range(10):
    print("hi")
    time.sleep(1)
with open(%q, "w") as f:
    f.write("Hello, world!")
`, fname)

	err := RunCodeWithTimeout(1*time.Second, nil, code)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}

	if _, err := os.Stat(fname); err == nil {
		t.Fatalf("expected file %s not to be created due to cancellation", fname)
	}
}

// TestRunCodeContextManualCancel ensures that manual cancellation returns context.Canceled
func TestRunCodeContextManualCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires Python runtime in short mode")
	}

	tmp := t.TempDir()
	fname := filepath.Join(tmp, "test_ctx_cancel_manual.txt")
	code := fmt.Sprintf(`
import time
for i in range(10):
    print("hi")
    time.sleep(1)
with open(%q, "w") as f:
    f.write("Hello, world!")
`, fname)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()
	err := RunCodeContext(ctx, nil, code)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	if _, err := os.Stat(fname); err == nil {
		t.Fatalf("expected file %s not to be created due to cancellation", fname)
	}
}

// TestRunCodeContextSuccess ensures that when the timeout is sufficient,
// the Python execution completes and writes the expected file.
func TestRunCodeContextSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test that requires Python runtime in short mode")
	}

	tmp := t.TempDir()
	fname := filepath.Join(tmp, "test_ctx_ok.txt")
	code := fmt.Sprintf(`
import time
time.sleep(1)
with open(%q, "w") as f:
    f.write("OK")
`, fname)

	err := RunCodeWithTimeout(5*time.Second, nil, code)
	if err != nil {
		t.Fatalf("expected RunCodeWithTimeout to succeed, got error: %v", err)
	}

	if _, err := os.Stat(fname); err != nil {
		t.Fatalf("expected file %s to exist after successful run, got error: %v", fname, err)
	}
}
