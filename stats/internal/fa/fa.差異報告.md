# fa.go — 差異報告

檔案: fa.go
對應 R 檔案: fa.R / psych::fa

## 摘要（高階差異）

- 目的：比較 Go `fa` 主流程與 R `psych::fa` 在估計步驟（initial communalities、factorextraction、rotation dispatch）與參數預設、錯誤處理、回傳結構上的差異。
- 建議：統一返回 diagnostics（converged/iterations/fits）、避免 panic、並以 fixtures 驗證各步驟一致性。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/fa`
- 建議建立 5 組代表性資料（不同 p/k、含缺值與病態案例），由 R 產生金標並比對 Go 輸出（loadings、communalities、Phi、diagnostics）。

## 下一步

- 我可以產生 PoC fixtures 並新增 minimal Go test harness 以自動比較 R 與 Go 的結果。

## 修正記錄

### 2024-12-XX

- ✅ 添加了 `RotateWithDiagnostics` 函數：提供統一的 diagnostics 返回，包含 method、converged、restarts、objective 和 iterations 資訊
- ✅ 保持向後相容性：保留了原有的 `Rotate` 函數作為包裝函數
- ✅ 統一診斷資訊：diagnostics map 包含了關鍵的收斂和效能指標，便於與 R 結果比較
