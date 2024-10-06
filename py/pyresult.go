// py/pyresult.go

package py

import (
	"encoding/json"
	"net/http"

	"github.com/HazelnutParadise/insyra"
)

var pyResult map[string]interface{}

// 啟動 HTTP 伺服器來接收 Python 回傳的複雜資料結構
func startServer() {
	http.HandleFunc("/pyresult", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// 使用 map 接收任意類型的資料
		var result map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&result)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		pyResult = result
	})

	insyra.LogInfo("py.init: Insyra listening on http://localhost:" + port + "...")
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		insyra.LogWarning("py.init: Failed to start server on port %s, trying backup port %s...", port, backupPort)
		err = http.ListenAndServe(":"+backupPort, nil)
		if err != nil {
			insyra.LogFatal("py.init: Failed to start backup server on port %s: %v", backupPort, err)
		}
	}

}
