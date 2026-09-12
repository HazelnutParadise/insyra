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
	ipcAddress  string
	serverReady = make(chan struct{})
	serverOnce  sync.Once
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
			// Python執行失敗（非正常退出），使用系統執行錯誤
			resultStore.Delete(executionID)
			return [2]any{nil, err.Error()}
		case <-processDone:
			// Python 進程已正常結束。結果由 handleIPCConnection 在 Python 送出
			// ack 前就已 Store（Store happens-before 進程結束、也就在 processDone
			// 關閉之前），因此這裡做最後一次讀取即可拿到剛送達的結果，避免把它刪掉後
			// 永久空轉。沒有結果則回傳空值而非繼續迴圈。
			if result, exists := resultStore.LoadAndDelete(executionID); exists {
				return result.([2]any)
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

// 啟動 IPC 伺服器來接收 Python 回傳的複雜資料結構
func startServer() {
	serverOnce.Do(func() {
		// Generate IPC address
		if runtime.GOOS == "windows" {
			// Use a random suffix for the pipe name
			randBytes := make([]byte, 8)
			if _, err := rand.Read(randBytes); err != nil {
				insyra.LogWarning("py", "startServer", "rand.Read failed: %v", err)
			}
			ipcAddress = fmt.Sprintf(`\\.\pipe\insyra_ipc_%x`, randBytes)
		} else {
			// Use a temp file for unix socket
			randBytes := make([]byte, 8)
			if _, err := rand.Read(randBytes); err != nil {
				insyra.LogWarning("py", "startServer", "rand.Read failed: %v", err)
			}
			ipcAddress = filepath.Join(os.TempDir(), fmt.Sprintf("insyra_ipc_%x.sock", randBytes))
			// Ensure it doesn't exist
			if rerr := os.Remove(ipcAddress); rerr != nil && !os.IsNotExist(rerr) {
				insyra.LogWarning("py", "startServer", "failed to remove leftover ipc socket: %v", rerr)
			}
		}

		ln, err := ipc.Listen(ipcAddress)
		if err != nil {
			// A library must not end the caller's process: record the failure
			// and leave the server down. Calls that need it then fail with a
			// recorded error instead of dereferencing a nil listener.
			insyra.LogError("py", "startServer", "Failed to start IPC server on %s: %v", ipcAddress, err)
			close(serverReady)
			return
		}
		// insyra.LogInfo("py", "init", "Insyra IPC server listening on %s", ipcAddress)

		// Signal that the server is ready
		close(serverReady)

		// Clean up the socket file when the process ends. Without this every
		// run leaves one behind in os.TempDir().
		if runtime.GOOS != "windows" {
			addr := ipcAddress
			runtime.AddCleanup(&serverOnce, func(path string) {
				_ = os.Remove(path)
			}, addr)
		}

		// Accept loop
		go func() {
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
					return
				}
				go handleIPCConnection(conn)
			}
		}()
	})
}

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

// getIPCAddress returns the IPC address, waiting for the server to start if necessary.
func getIPCAddress() string {
	<-serverReady
	return ipcAddress
}
