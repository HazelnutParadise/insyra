package py

import (
	"crypto/rand"
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
			// Python進程已經正常結束，檢查是否有結果
			resultStore.Delete(executionID)
			// 給結果發送留一點時間
			time.Sleep(10 * time.Millisecond)
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
			insyra.LogFatal("py", "init", "Failed to start IPC server on %s: %v", ipcAddress, err)
		}
		// insyra.LogInfo("py", "init", "Insyra IPC server listening on %s", ipcAddress)

		// Signal that the server is ready
		close(serverReady)

		// Accept loop
		for {
			conn, err := ln.Accept()
			if err != nil {
				insyra.LogWarning("py", "server", "Accept error: %v", err)
				continue
			}
			go handleIPCConnection(conn)
		}
	})
}

func handleIPCConnection(conn net.Conn) {
	defer func() {
		if cerr := conn.Close(); cerr != nil {
			insyra.LogWarning("py", "server", "conn.Close error: %v", cerr)
		}
	}()

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
