# 檔案: `psych_Promax.go` — 差異報告

對應 R 檔案: `psych_Promax.R`

## 摘要（高階差異）

- 目的：比較 Go 與 R 在 Promax（oblique）旋轉的實作差異，包括快速近似（Kaiser-style power transform）、參數對應、錯誤處理與回傳資訊的差別。
- 高階結論：Go 實作與 R 的核心思路一致（先 orthogonal rotate，再做 power transform 並最小化目標函數），但需確認參數（power、normalize）、數值穩健性與回傳診斷的一致性。

## 逐段重點差異

1. 演算法步驟

- 共通：通常先套用 orthogonal rotation（例如 varimax），接著對絕對負載值取 power（通常為 3 或 4），再估計 oblique 轉換矩陣並計算負載矩陣與 phi。
- 差異檢核：確認 Go 是否在 power transform 後使用與 R 相同的正則化 / 標準化步驟，以及是否在求解轉換矩陣時使用相同最小二乘或投影方法。

2. 參數對應（power、normalize）

- R：`Promax` 允許設定 power（p）與 normalize 選項，且在不同版本中預設值可能不同。
- Go：檢查 `psych_Promax.go` 是否暴露相同參數與相同預設值，否則會導致與 R 結果的可比性降低。

3. 數值穩健性與錯誤處理

- R：若某步驟失敗，會以 warning 或 fallback 機制處理。
- Go：若目前存在 panic 或直接錯誤中斷的情形，建議改為回傳 error，並記錄詳細診斷資訊（迭代次數、收斂標誌）。

4. 回傳資料結構

- 建議 Go 回傳結構應包含 `loadings`、`Phi`、`rotMat`、`converged` 與 `iterations` 等欄位，以便與 R 的回傳物件比對。

## 相容性風險與優先度建議

- 高優先：確認 `power` 與 `normalize` 的預設值是否與 R 相同；若不同，記錄差異並在文件中註明。
- 中優先：將可能發生的 panic 改為 error 回傳，並在錯誤情況下提供 fallback（例如減小 power 或改用不同的 initial rotation）。

## 測試建議（最小集合）

1. 與 R 比對：選取 5 組 loadings（含稀疏與非稀疏），在 R 與 Go 上執行 Promax，保存 `loadings`、`Phi` 與 `rotMat`，做逐元素比較。
2. 參數敏感度：測試 power=2,3,4 與 normalize=true/false 的組合，觀察差異。
3. 邊界情況：包含零向量行/列的矩陣，測試是否會導致奇異系統或 panic，並要求 error 返回。
