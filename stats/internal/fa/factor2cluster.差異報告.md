檔案: factor2cluster.go
對應 R 檔案: (R 端未直接找到 `factor2cluster` 名稱函式；`objects_index.R` 未列出對應，R 實作可能是 `cluster` 或 `target.rot` 的組合/變體)

摘要（高階差異）

- Go 實作 `Factor2Cluster` 直接從 loadings 找出每個變項最大絕對載荷所屬因子，並回傳一個 p x q 的二元矩陣 (one-hot cluster matrix)。
- 在 R 的 `r_source_code_fa` 目錄中沒有直接名為 `factor2cluster` 的函式。R 端常見做法是先做 Varimax 或其他正交轉換，再用 `target.rot` 或手工搜尋來得到 cluster 描述。`psych_faRotations.R` 中在 `cluster` 分支會先做 `varimax` 再呼叫 `target.rot`，與 Go `Factor2Cluster` 的輸出概念類似，但不是逐行等價實作。

逐行重要差異摘錄

- Go: 明確的最大值絕對值比較，若存在相同絕對值（tie），目前實作會選擇第一個遇到的因子索引（index），未做平手處理。
- R: 若使用 `target.rot` 或其他 cluster 機制，會考量目標矩陣與旋轉策略，可能會有不同的分群結果（非僅僅最大載荷）。

相容性風險與建議

- 若使用者期望 R `cluster` 分群（由複雜的 target.rot 流程產生）與 Go 返回相同結果，可能會不一致。建議明確文件化 `Factor2Cluster` 的行為（最大絕對載荷決定），或在必要時提供一個更接近 R 的替代—例如先執行 Varimax + target.rot 再從結果產生 cluster 矩陣。

## 優先次序（建議）

1. 文件化行為（高），2. 提供可選 pipeline（中），3. 加入測試 fixtures（中）。

## 下一步

- 在 `tests/fa/fixtures/factor2cluster` 放入 5 組 R pipeline（varimax+target.rot）產生的對照資料，並在 Go 中新增選項 `UseRLikePipeline` 以比較差異（我可以幫忙草擬 PR）。
