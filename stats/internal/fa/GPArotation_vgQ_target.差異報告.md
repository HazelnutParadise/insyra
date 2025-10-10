檔案: GPArotation_vgQ_target.go
對應 R 檔案: GPArotation_vgQ.target.R

## 摘要（高階差異）

- 目的：確認 Go 的 `vgQTarget` 在計算 target-based 目標函數與梯度時，行為與 R 的 `vgQ.target` 等價（包括 lower.tri/upper.tri 處理與 mask 行為）。
- 高階結論：需確認 target 矩陣索引規約（R 常用 lower.tri 與 NA 作為 mask）在 Go 中是否被正確模擬，並補足測試向量以驗證梯度計算。

## 逐行重要差異摘錄

- target mask 的表示：R 使用 `NA` 表示不考慮的元素；Go 應確認是否接受 `NaN` 或其他 sentinel，並在計算時只使用非 NA/非 NaN 的條目。
- lower.tri 對應：R 在處理對稱目標時常只傳 lower.tri；Go 必須在內部回填成完整矩陣或採取一致的索引方式。

## 相容性風險與建議

- 若 mask 行為不同，最終對齊結果會不同。建議在 API 中明確支援 NA/NaN 作為 mask 並在文檔中列出行為。
- 建議在遇到全部元素為 NA 的 target 時返回明確的 error（而非 panic 或 silent no-op）。

## 測試建議（最小集合）

1. 小型 target 範例：建立 4x4 target 含 NA（lower.tri 填入值，上三角 NA），在 R 與 Go 做對齊並比對 residual norm 與 aligned loadings。
2. 全 NA 與無 NA 情況測試：確保行為一致（error vs no-op）。

## 優先次序

1. mask/NA 支援（高），2. lower.tri 行為一致性（中），3. 測試與文件（中）。

## 下一步

- 我會提取 R 端的 target 範例並把至少 3 個 fixtures 加入 `repo/tests/fa/fixtures` 以便後續 Go 單元測試比對（需你同意我建立 fixtures）。
