package py

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HazelnutParadise/insyra/py/internal/ipc"
)

// fakePythonEnv, when set, makes the test binary act as the environment's
// Python. It reads the execution ID and the IPC address from the script it is
// given and answers the way the value says: "result:<JSON>" returns the JSON
// value, "pyjson:<text>" returns the text as it is, the way Python's
// json.dumps writes NaN and the infinities, "sleep" never answers, and "exit:<code>" exits with that
// status without returning anything, as Python does for a script it cannot
// compile. It fails, as insyra.Return raises, unless the Go side acknowledges
// the result.
const fakePythonEnv = "INSYRA_PY_TEST_FAKE_PYTHON"

var (
	scriptExecutionID = regexp.MustCompile(`execution_id = "([0-9a-f]+)"`)
	scriptIPCAddress  = regexp.MustCompile(`ipc_address = r"([^"]+)"`)
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakePythonEnv); mode != "" {
		os.Exit(fakePython(mode, os.Args[1:]))
	}
	os.Exit(m.Run())
}

// useFakePython marks the environment ready with the test binary as its
// interpreter, answering as fakePython does for mode, and restores the package
// when the test ends.
func useFakePython(t *testing.T, mode string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	pyInitMu.Lock()
	wasInit, wasPy := isPyEnvInit, pyPath
	isPyEnvInit, pyPath = true, self
	pyInitMu.Unlock()
	t.Cleanup(func() {
		pyInitMu.Lock()
		isPyEnvInit, pyPath = wasInit, wasPy
		pyInitMu.Unlock()
	})
	t.Setenv(fakePythonEnv, mode)
	if runtime.GOOS != "windows" {
		t.Setenv("TMPDIR", "") // os.TempDir() is then /tmp, short enough for the IPC socket
	}
}

// fakePython stands in for the environment's interpreter; see fakePythonEnv.
func fakePython(mode string, args []string) int {
	if code, ok := strings.CutPrefix(mode, "exit:"); ok {
		status, err := strconv.Atoi(code)
		if err != nil {
			fmt.Fprintln(os.Stderr, "fake python: bad exit status", code)
			return 3
		}
		fmt.Fprintln(os.Stderr, "fake python: exiting with status", status)
		return status
	}
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
	var result any
	var msg []byte
	switch {
	case strings.HasPrefix(mode, "pyjson:"):
		// Sent as it is: Python's json.dumps writes NaN, Infinity and
		// -Infinity, which encoding/json would refuse to marshal.
		msg = []byte(`{"execution_id": "` + string(id[1]) + `", "data": [` + strings.TrimPrefix(mode, "pyjson:") + `, null]}`)
	case strings.HasPrefix(mode, "result:"):
		result = json.RawMessage(strings.TrimPrefix(mode, "result:"))
	default:
		fmt.Fprintln(os.Stderr, "fake python: unknown mode", mode)
		return 3
	}
	if msg == nil {
		msg, err = json.Marshal(map[string]any{"execution_id": string(id[1]), "data": []any{result, nil}})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 3
		}
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
	reply, err := ipc.ReadMessage(conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}
	// insyra.Return raises unless the Go side says it read the result.
	var ack struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(reply, &ack); err != nil || ack.Status != "ok" {
		fmt.Fprintln(os.Stderr, "fake python: the Go side refused the result:", ack.Error)
		return 1
	}
	return 0
}
