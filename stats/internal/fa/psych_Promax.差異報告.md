# psych_Promax.go — 差異報告

檔案: psych_Promax.go
對應 R 檔案: psych_Promax.R

## 摘要（高階差異）

- 目的：比較 Go 與 R 在 Promax（oblique）旋轉的差異，包含 power transform、參數對應、錯誤處理與回傳資訊。
- 結論：Go 的核心流程與 R 相符（先 orthogonal rotate、再做 power transform 並最小化目標函數），但需驗證 `power` / `normalize` 的預設值與數值穩健性。

## 逐段差異（重點）

1. 演算法步驟

- 共通：先套用 orthogonal rotation（例如 varimax），接著對絕對載荷值做 power transform（通常為 3 或 4），再估計 oblique 轉換矩陣與 phi。
- 差異檢核：確認 Go 在 power transform 後的正規化 / 標準化步驟，以及求解轉換矩陣使用的方法是否與 R 相同。

1. 參數對應（power、normalize）

- R：`Promax` 允許設定 `power` 與 `normalize`，不同版本的預設值可能不同。
- Go：請檢查 `psych_Promax.go` 是否暴露相同參數與預設值，避免結果差異。

1. 數值穩健性與錯誤處理

- R：若某步驟失敗會 warning / fallback。
- Go：若現有實作有 panic 或未妥善處理的 error，建議改為回傳 error 並加入 diagnostics。

1. 回傳結構

- 建議 Go 回傳 `loadings`、`Phi`、`rotMat`、`converged`、`iterations` 等欄位以便比對。

## 優先次序（建議）

1. 確認 `power` / `normalize` 的預設與 API（高）
2. panic->error（中）
3. 加入 fixtures（中）

## 測試建議

1. 選取 5 組代表性 loadings，在 R 與 Go 上以不同 `power` 值（2,3,4）與 `normalize` true/false 做比較。
2. 邊界測試：輸入含 NaN/Inf 或接近奇異的載荷矩陣，觀察行為（error vs fallback）。

## 下一步

- 生成 fixtures 並放入 `tests/fa/fixtures/promax`，新增 minimal harness 以比對結果。
