package py

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/py/internal/ipc"
)

// fakeUVEnv, when set in a child's environment, makes the test binary act as
// uv. The tests copy the binary to where the setup runs uv from, so the setup
// is exercised without the network. The value says what the fake does.
const fakeUVEnv = "INSYRA_PY_TEST_FAKE_UV"

// fakeUVLogEnv names a file the fake uv appends its arguments to, one line
// per run.
const fakeUVLogEnv = "INSYRA_PY_TEST_FAKE_UV_LOG"

// fakePythonEnv, when set, makes the test binary act as the environment's
// Python. It reads the execution ID and the IPC address from the script it is
// given and answers the way the value says: "result:<JSON>" returns the JSON
// value, "error:<text>" returns an error, "script" returns the script itself,
// and "sleep" waits to be killed.
const fakePythonEnv = "INSYRA_PY_TEST_FAKE_PYTHON"

var (
	scriptExecutionID = regexp.MustCompile(`execution_id = "([0-9a-f]+)"`)
	scriptIPCAddress  = regexp.MustCompile(`ipc_address = r"([^"]+)"`)
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeUVEnv); mode != "" {
		os.Exit(fakeUV(mode, os.Args[1:]))
	}
	if mode := os.Getenv(fakePythonEnv); mode != "" {
		os.Exit(fakePython(mode, os.Args[1:]))
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

// fakePython stands in for the environment's interpreter; see fakePythonEnv.
func fakePython(mode string, args []string) int {
	if mode == "sleep" {
		time.Sleep(time.Minute)
		return 0
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "fake python: want one script, got", args)
		return 3
	}
	script, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	id := scriptExecutionID.FindSubmatch(script)
	addr := scriptIPCAddress.FindSubmatch(script)
	if id == nil || addr == nil {
		fmt.Fprintln(os.Stderr, "fake python: the script has no execution ID or IPC address")
		return 3
	}
	var result, pyErr any
	switch {
	case strings.HasPrefix(mode, "result:"):
		result = json.RawMessage(strings.TrimPrefix(mode, "result:"))
	case strings.HasPrefix(mode, "error:"):
		pyErr = strings.TrimPrefix(mode, "error:")
	case mode == "script":
		result = string(script)
	default:
		fmt.Fprintln(os.Stderr, "fake python: unknown mode", mode)
		return 3
	}
	msg, err := json.Marshal(map[string]any{"execution_id": string(id[1]), "data": []any{result, pyErr}})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	conn, err := ipc.Dial(string(addr[1]))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	defer func() { _ = conn.Close() }()
	if err := ipc.WriteMessage(conn, msg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	if _, err := ipc.ReadMessage(conn); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	return 0
}
