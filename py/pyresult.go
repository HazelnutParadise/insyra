package py

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

var (
	resultStore sync.Map // map[string][2]interface{}
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
func waitForResult(executionID string, processDone <-chan struct{}, execErr <-chan error) [2]interface{} {
	for {
		select {
		case err := <-execErr:
			// Python執行失敗（非正常退出），使用系統執行錯誤
			resultStore.Delete(executionID)
			return [2]interface{}{nil, err.Error()}
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
				return result.([2]interface{})
			}
			// 短暫等待後重試
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// 啟動 HTTP 伺服器來接收 Python 回傳的複雜資料結構
func startServer() {
	http.HandleFunc("/pyresult", func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()

		// 解析請求體
		var requestData struct {
			ExecutionID string         `json:"execution_id"`
			Data        [2]interface{} `json:"data"`
		}

		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// 直接存儲結果到sync.Map
		resultStore.Store(requestData.ExecutionID, requestData.Data)

		w.WriteHeader(http.StatusOK)
	})

	insyra.LogInfo("py", "init", "Insyra listening on http://localhost:"+port+".")
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		insyra.LogWarning("py", "init", "Failed to start server on port %s, trying backup port %s...", port, backupPort)
		err = http.ListenAndServe(":"+backupPort, nil)
		if err != nil {
			insyra.LogFatal("py", "init", "Failed to start backup server on port %s: %v", backupPort, err)
		}
	}
}
