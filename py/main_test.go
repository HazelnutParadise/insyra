package py

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeUVEnv, when set in a child's environment, makes the test binary act as
// uv. The tests copy the binary to where the setup runs uv from, so the setup
// is exercised without the network. The value says what the fake does.
const fakeUVEnv = "INSYRA_PY_TEST_FAKE_UV"

// fakeUVLogEnv names a file the fake uv appends its arguments to, one line
// per run.
const fakeUVLogEnv = "INSYRA_PY_TEST_FAKE_UV_LOG"

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeUVEnv); mode != "" {
		os.Exit(fakeUV(mode, os.Args[1:]))
	}
	os.Exit(m.Run())
}

// fakeUV stands in for `uv sync`: "ok" creates the interpreter the setup
// looks for, "fail" exits with an error, "sleep" waits to be killed.
func fakeUV(mode string, args []string) int {
	if log := os.Getenv(fakeUVLogEnv); log != "" {
		if f, err := os.OpenFile(log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			fmt.Fprintln(f, strings.Join(args, " "))
			_ = f.Close()
		}
	}
	switch mode {
	case "ok":
		// The setup must keep the user's own settings that would break the
		// sync away from uv.
		if v, set := os.LookupEnv("UV_PYTHON_PREFERENCE"); set {
			fmt.Fprintln(os.Stderr, "fake uv: UV_PYTHON_PREFERENCE reached uv:", v)
			return 4
		}
		if v := os.Getenv("UV_PYTHON_DOWNLOADS"); v != "automatic" {
			fmt.Fprintln(os.Stderr, "fake uv: UV_PYTHON_DOWNLOADS is", v, "not automatic")
			return 4
		}
		python := venvPython(os.Getenv("UV_PROJECT_ENVIRONMENT"))
		if err := os.MkdirAll(filepath.Dir(python), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := os.WriteFile(python, nil, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
	case "fail":
		fmt.Fprintln(os.Stderr, "error: fake uv was told to fail")
		return 2
	case "sleep":
		time.Sleep(time.Minute)
		return 0
	}
	fmt.Fprintln(os.Stderr, "fake uv: unknown mode", mode)
	return 3
}
