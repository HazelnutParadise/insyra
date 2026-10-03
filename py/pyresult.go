package py

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/py/internal/ipc"
	json "github.com/goccy/go-json"
)

var (
	resultStore sync.Map // map[string][2]any

	// refusedStore keeps why the result a run sent could not be read. It is
	// the run's error only if the run ends without delivering another one:
	// Python can catch the error insyra.Return raises and return something
	// else.
	refusedStore sync.Map // map[string]string

	// The IPC server is shared by the runs in flight and closed when the last
	// of them finishes. Closing a Unix listener removes its socket file.
	serverMu    sync.Mutex
	serverLn    net.Listener
	serverAddr  string
	serverUsers int
)

// 生成唯一的執行ID
func generateExecutionID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to timestamp-based ID if crypto rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", bytes)
}

// 等待並獲取指定ID的結果，當Python進程結束時自動返回nil
func waitForResult(executionID string, processDone <-chan struct{}, execErr <-chan error) [2]any {
	defer refusedStore.Delete(executionID)
	for {
		select {
		case err := <-execErr:
			// A result Python delivered before the process failed still
			// counts, for instance when its context ended between the two.
			if result, exists := resultStore.LoadAndDelete(executionID); exists {
				return result.([2]any)
			}
			// insyra.Return raises when the Go side refuses its result, which
			// fails the process; the refusal says why.
			if reason, refused := refusedStore.Load(executionID); refused {
				return [2]any{nil, reason}
			}
			return [2]any{nil, err.Error()}
		case <-processDone:
			// handleIPCConnection stores a result before it acknowledges it,
			// and Python exits only after the acknowledgement, so a result
			// that was delivered is in the store by now.
			if result, exists := resultStore.LoadAndDelete(executionID); exists {
				return result.([2]any)
			}
			if reason, refused := refusedStore.Load(executionID); refused {
				return [2]any{nil, reason}
			}
			// A process that failed reported its error before processDone
			// closed, so the error is already waiting. select may still pick
			// this case, and the failure would pass for a run that returned
			// nothing.
			select {
			case err := <-execErr:
				return [2]any{nil, err.Error()}
			default:
			}
			return [2]any{nil, nil}
		default:
			// 檢查是否有結果
			if result, exists := resultStore.Load(executionID); exists {
				// 找到結果，清理並返回
				resultStore.Delete(executionID)
				return result.([2]any)
			}
			// 短暫等待後重試
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// ipcConnDeadline bounds one request/response exchange on an IPC connection.
// A Python call that legitimately takes longer than this is the caller's
// timeout to set, not the framing layer's.
const ipcConnDeadline = 10 * time.Minute

func handleIPCConnection(conn net.Conn) {
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			insyra.LogWarning("py", "server", "conn.Close error: %v", cerr)
		}
	}()

	// A peer that connects and then says nothing would hold the goroutine and
	// the connection open for good.
	if derr := conn.SetDeadline(time.Now().Add(ipcConnDeadline)); derr != nil {
		insyra.LogWarning("py", "server", "failed to set a deadline on the IPC connection: %v", derr)
	}

	// Read message
	msg, err := ipc.ReadMessage(conn)
	if err != nil {
		// insyra.LogWarning("py", "server", "ReadMessage error: %v", err)
		return
	}

	m, err := decodeResultMessage(msg)
	if err != nil {
		reason := fmt.Sprintf("py: the result Python sent could not be read: %v", err)
		if id := runIDOf(msg); id != "" {
			// The run returns this error if nothing else arrives.
			refusedStore.Store(id, reason)
		} else {
			insyra.LogWarning("py", "server", "%s", reason)
		}
		writeReply(conn, map[string]string{"status": "error", "error": reason})
		return
	}
	resultStore.Store(m.ExecutionID, m.Data)
	writeReply(conn, map[string]string{"status": "ok"})
}

// newIPCAddress returns a fresh random address: a named pipe on Windows, a
// socket file in os.TempDir() elsewhere.
func newIPCAddress() string {
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		insyra.LogWarning("py", "newIPCAddress", "rand.Read failed: %v", err)
	}
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`\\.\pipe\insyra_ipc_%x`, randBytes)
	}
	addr := filepath.Join(os.TempDir(), fmt.Sprintf("insyra_ipc_%x.sock", randBytes))
	if rerr := os.Remove(addr); rerr != nil && !os.IsNotExist(rerr) {
		insyra.LogWarning("py", "newIPCAddress", "failed to remove leftover ipc socket: %v", rerr)
	}
	return addr
}

// acquireIPCServer returns the address of the IPC server that carries results
// back from Python, opening the server when no run is using it. A server that
// cannot open is an error, and the next call tries again. Every successful
// call must be matched by one releaseIPCServer.
func acquireIPCServer() (string, error) {
	serverMu.Lock()
	defer serverMu.Unlock()
	if serverUsers == 0 {
		addr := newIPCAddress()
		ln, err := ipc.Listen(addr)
		if err != nil {
			return "", fmt.Errorf("py: failed to start the IPC server on %s: %w", addr, err)
		}
		serverLn, serverAddr = ln, addr
		go acceptIPC(ln)
	}
	serverUsers++
	return serverAddr, nil
}

// releaseIPCServer ends one run's use of the IPC server and closes the server
// when no run is left.
func releaseIPCServer() {
	serverMu.Lock()
	defer serverMu.Unlock()
	if serverUsers == 0 {
		return
	}
	serverUsers--
	if serverUsers > 0 {
		return
	}
	if err := serverLn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		insyra.LogWarning("py", "server", "failed to close the IPC server: %v", err)
	}
	serverLn, serverAddr = nil, ""
}

// acceptIPC hands each connection on ln to handleIPCConnection until ln is
// closed or fails.
func acceptIPC(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			// A listener that is closed, or permanently broken, makes
			// Accept fail every time. `continue` then spun a warning
			// per iteration for the life of the process.
			if errors.Is(err, net.ErrClosed) {
				return
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			insyra.LogError("py", "server", "IPC accept failed, stopping the listener: %v", err)
			// Close it too: a listener left open still queues connections,
			// and the Python process behind one would wait for an
			// acknowledgement nobody sends. Closed, its connect fails and
			// the run returns an error.
			_ = ln.Close()
			return
		}
		go handleIPCConnection(conn)
	}
}

// writeReply answers insyra.Return, which raises unless the status is ok.
func writeReply(conn net.Conn, reply map[string]string) {
	resp, err := json.Marshal(reply)
	if err != nil {
		insyra.LogWarning("py", "server", "json marshal response failed: %v", err)
		return
	}
	if err := ipc.WriteMessage(conn, resp); err != nil {
		insyra.LogWarning("py", "server", "WriteMessage error: %v", err)
	}
}

// resultMessage is the message insyra.Return sends: the run's ID and its
// [result, error] pair.
type resultMessage struct {
	ExecutionID string `json:"execution_id"`
	Data        [2]any `json:"data"`
}

// Python writes a float's exponent with a lowercase e, so these numbers never
// appear in what it sends and can stand in for the names it writes for a NaN
// and an infinity.
const (
	nanMark = "0E0"
	infMark = "1E0"
)

var (
	nanName = []byte("NaN")
	infName = []byte("Infinity")
)

// decodeResultMessage decodes a message from insyra.Return. Python's
// json.dumps writes a NaN or an infinity as NaN, Infinity or -Infinity, which
// JSON does not have, and writes an integer above 2^53 in full, which a
// float64 would round. So a message the JSON decoder refuses, or one with
// digits enough for such an integer, is read again with those names marked
// and its numbers kept as text, and restoreNumbers turns the numbers back. A
// message with neither decodes as it always has.
func decodeResultMessage(msg []byte) (resultMessage, error) {
	var m resultMessage
	if !hasLongInteger(msg) {
		if err := json.Unmarshal(msg, &m); err == nil {
			return m, nil
		}
	}
	marked, err := markNonFinite(msg)
	if err != nil {
		return resultMessage{}, err
	}
	dec := json.NewDecoder(bytes.NewReader(marked))
	dec.UseNumber()
	m = resultMessage{}
	if err := dec.Decode(&m); err != nil {
		return resultMessage{}, err
	}
	for i, v := range m.Data {
		restored, err := restoreNumbers(v)
		if err != nil {
			return resultMessage{}, err
		}
		m.Data[i] = restored
	}
	return m, nil
}

// markNonFinite replaces the names Python writes outside strings for a NaN
// and an infinity with nanMark and infMark. The minus sign of -Infinity stays
// in front of its mark. A number too large for a float64, which go-json would
// refuse with a message quoting all of it, is an error saying so.
func markNonFinite(msg []byte) ([]byte, error) {
	out := make([]byte, 0, len(msg))
	inString := false
	for i := 0; i < len(msg); i++ {
		c := msg[i]
		switch {
		case inString:
			out = append(out, c)
			if c == '\\' && i+1 < len(msg) {
				i++
				out = append(out, msg[i])
			} else if c == '"' {
				inString = false
			}
		case c == '"':
			inString = true
			out = append(out, c)
		case bytes.HasPrefix(msg[i:], nanName):
			out = append(out, nanMark...)
			i += len(nanName) - 1
		case bytes.HasPrefix(msg[i:], infName):
			out = append(out, infMark...)
			i += len(infName) - 1
		case c == '-' || (c >= '0' && c <= '9'):
			end := i + 1
			for end < len(msg) && strings.IndexByte("0123456789.eE+-", msg[end]) >= 0 {
				end++
			}
			number := string(msg[i:end])
			if _, err := strconv.ParseFloat(number, 64); errors.Is(err, strconv.ErrRange) {
				return nil, numberTooLarge(number)
			}
			out = append(out, number...)
			i = end - 1
		default:
			out = append(out, c)
		}
	}
	return out, nil
}

// numberTooLarge reports a number a float64 cannot hold, such as Python's
// 10**400, with its text shortened.
func numberTooLarge(text string) error {
	if len(text) > 30 {
		text = text[:30] + "…"
	}
	return fmt.Errorf("the number %s does not fit in a float64", text)
}

// restoreNumbers turns the json.Number values decodeResultMessage reads into
// the float64 values the JSON decoder gives, except an integer a float64
// would round, and the marks into NaN and the infinities.
func restoreNumbers(v any) (any, error) {
	switch x := v.(type) {
	case json.Number:
		switch x {
		case nanMark:
			return math.NaN(), nil
		case infMark:
			return math.Inf(1), nil
		case "-" + infMark:
			return math.Inf(-1), nil
		}
		if n, ok := exactInteger(string(x)); ok {
			return n, nil
		}
		f, err := strconv.ParseFloat(string(x), 64)
		if err != nil {
			return nil, numberTooLarge(string(x))
		}
		return f, nil
	case []any:
		for i, e := range x {
			restored, err := restoreNumbers(e)
			if err != nil {
				return nil, err
			}
			x[i] = restored
		}
	case map[string]any:
		for k, e := range x {
			restored, err := restoreNumbers(e)
			if err != nil {
				return nil, err
			}
			x[k] = restored
		}
	}
	return v, nil
}

// runIDOf finds the run's ID at the start of a message that could not be
// decoded, where insyra.Return writes it. It returns "" when it is not there.
func runIDOf(msg []byte) string {
	rest, ok := bytes.CutPrefix(msg, []byte(`{"execution_id": "`))
	if !ok {
		return ""
	}
	id, _, ok := bytes.Cut(rest, []byte(`"`))
	if !ok || len(id) == 0 {
		return ""
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return ""
		}
	}
	return string(id)
}

// maxExactInteger is 2^53, past which a float64 no longer holds every integer.
const maxExactInteger = 1 << 53

// hasLongInteger reports whether msg may hold an integer above 2^53: sixteen
// digits or more in a row, outside a string, that are neither the fraction of
// a decimal nor the part before its point or exponent, since Python writes an
// int with neither. Without one, every number in msg is exact as a float64.
func hasLongInteger(msg []byte) bool {
	inString := false
	for i := 0; i < len(msg); i++ {
		c := msg[i]
		switch {
		case inString:
			switch c {
			case '\\':
				i++
			case '"':
				inString = false
			}
		case c == '"':
			inString = true
		case c >= '0' && c <= '9':
			start := i
			for i < len(msg) && msg[i] >= '0' && msg[i] <= '9' {
				i++
			}
			fraction := start > 0 && msg[start-1] == '.'
			mantissa := i < len(msg) && (msg[i] == '.' || msg[i] == 'e' || msg[i] == 'E')
			if i-start >= 16 && !fraction && !mantissa {
				return true
			}
			i--
		}
	}
	return false
}

// exactInteger returns an integer Python wrote that a float64 would round: one
// above 2^53 in magnitude, as an int64, or as a uint64 above the int64 range.
// Python writes an int with no decimal point or exponent and a float always
// with one, so a float is never taken for an int. An integer beyond 64 bits is
// left to be read as a float64.
func exactInteger(text string) (any, bool) {
	if strings.ContainsAny(text, ".eE") {
		return nil, false
	}
	if n, err := strconv.ParseInt(text, 10, 64); err == nil {
		if n > maxExactInteger || n < -maxExactInteger {
			return n, true
		}
		return nil, false
	}
	if n, err := strconv.ParseUint(text, 10, 64); err == nil {
		return n, true
	}
	return nil, false
}
