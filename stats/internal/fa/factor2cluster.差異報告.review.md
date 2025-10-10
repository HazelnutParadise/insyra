# factor2cluster.go — 差異報告（審閱提案）

注意：此檔為審閱提案，原始 `factor2cluster.差異報告.md` 保持不變。

檔案: factor2cluster.go
對應 R 實作：無直接同名函式（可能為 varimax+target.rot pipeline）

## 摘要

- 目的：文件化 Go `Factor2Cluster` 的行為（最大絕對載荷分派）並比較 R 的 pipeline 結果。
- 建議：在文件中明確 tie-break 策略、提供 `UseRLikePipeline` 選項，並建立 fixtures 以驗證差異。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/factor2cluster`
- 建議建立 5 組 pipeline（金標由 R 生成：varimax -> target.rot -> cluster），並比對 Go 的 one-hot cluster 結果。

## 下一步

- 我可以產生 PoC fixtures，並草擬一個替代 pipeline 的選項實作建議。
