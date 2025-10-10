# fa.go — 差異報告（審閱提案）

注意：此檔為審閱提案，原始 `fa.差異報告.md` 保持不變。

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
