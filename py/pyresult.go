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
	resultStore sync.Map // map[string]map[string]any
)

// 生成唯一的執行ID
func generateExecutionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// 等待並獲取指定ID的結果，當Python進程結束時自動返回nil
func waitForResult(executionID string, processDone <-chan struct{}) map[string]any {
	for {
		select {
		case <-processDone:
			// Python進程已經結束但沒有調用insyra_return，返回nil
			resultStore.Delete(executionID)
			return nil
		default:
			// 檢查是否有結果
			if result, exists := resultStore.Load(executionID); exists {
				// 找到結果，清理並返回
				resultStore.Delete(executionID)
				return result.(map[string]any)
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
			Data        map[string]any `json:"data"`
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
