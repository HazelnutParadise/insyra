package py

import (
	"net/http"
	"sync"

	"github.com/HazelnutParadise/insyra"
	json "github.com/goccy/go-json"
)

var (
	pyResult map[string]any
	mu       sync.Mutex
)

// 啟動 HTTP 伺服器來接收 Python 回傳的複雜資料結構
func startServer() {
	http.HandleFunc("/pyresult", func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()

		// 使用 sync.Pool 來緩存 map
		var result map[string]any
		pool := sync.Pool{
			New: func() interface{} {
				return make(map[string]any)
			},
		}

		result = pool.Get().(map[string]any)
		defer pool.Put(result)

		err := json.NewDecoder(r.Body).Decode(&result)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		mu.Lock()
		pyResult = result
		mu.Unlock()
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

// todo: pop機制與執行id
// func popPyResult() map[string]any {
// 	mu.Lock()
// 	defer mu.Unlock()
// 	result := pyResult
// 	pyResult = nil

// 	return result
// }
