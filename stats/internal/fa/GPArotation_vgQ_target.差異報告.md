# GPArotation_vgQ_target.go — 差異報告

檔案: GPArotation_vgQ_target.go
對應 R 檔案: GPArotation_vgQ.target.R

## 摘要

- 目的：比較 target rotation 在 vgQ 系列的實作差異（mask/NA 處理、penalty 與收斂條件），並列出檢驗重點。

## 逐段差異（重點）

1. Mask 表示：R 端常用 NA 作為 mask，且可只傳 lower.tri；Go 需要明確支援 NA/NaN 或其他 sentinel，並在內部一致地回填為完整矩陣。
2. 勾稽規約：對稱目標時 R 常只提供 lower.tri；Go 應提供清楚的 API 文件說明如何處理 lower.tri 輸入。
3. 錯誤/邊界：遇到全部元素為 NA 的情形，應返回明確 error 而非 silent no-op 或 panic。

## 相容性風險

- mask/NA 處理不一致會直接導致對齊結果差異，建議在 API 與文件中明列行為。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/target`
-- 建議至少建立：
  1. 小型 target 範例（4x4，lower.tri 填值，上三角 NA）
  2. 全 NA 範例（確保返回 error）
  3. 無 NA 範例（正常流程）

## 優先次序

1. mask/NA 支援與文件（高）
2. lower.tri 行為一致性（中）
3. 建立測試 fixtures（中）

## 下一步

- 由 R 端產生至少 3 組 target fixtures 並放到 `tests/fa/fixtures/target`，接著新增 Go 單元測試以自動比對 aligned loadings 與 residual norms。
