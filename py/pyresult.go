package py

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/py/internal/ipc"
	json "github.com/goccy/go-json"
)

var (
	resultStore sync.Map // map[string][2]any

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
	for {
		select {
		case err := <-execErr:
			// A result Python delivered before the process failed still
			// counts, for instance when its context ended between the two.
			if result, exists := resultStore.LoadAndDelete(executionID); exists {
				return result.([2]any)
			}
			return [2]any{nil, err.Error()}
		case <-processDone:
			// handleIPCConnection stores a result before it acknowledges it,
			// and Python exits only after the acknowledgement, so a result
			// that was delivered is in the store by now.
			if result, exists := resultStore.LoadAndDelete(executionID); exists {
				return result.([2]any)
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

	// Parse request
	var requestData struct {
		ExecutionID string `json:"execution_id"`
		Data        [2]any `json:"data"`
	}
	if err := json.Unmarshal(msg, &requestData); err != nil {
		insyra.LogWarning("py", "server", "Unmarshal error: %v", err)
		return
	}

	// Store result
	resultStore.Store(requestData.ExecutionID, requestData.Data)

	// Send response
	resp, merr := json.Marshal(map[string]string{"status": "ok"})
	if merr != nil {
		insyra.LogWarning("py", "server", "json marshal response failed: %v", merr)
		return
	}
	if werr := ipc.WriteMessage(conn, resp); werr != nil {
		insyra.LogWarning("py", "server", "WriteMessage error: %v", werr)
	}
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
