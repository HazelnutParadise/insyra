檔案: GPArotation_vgQ_simplimax.go
對應 R 檔案: GPArotation_vgQ.simplimax.R

## 摘要（高階差異）

- 目的：比較 Go 的 `vgQSimplimax` 與 R `vgQ.simplimax` 在目標函數設計（sparsity/complexity）的實作與梯度計算上是否等價。
- 高階結論：Go 的實作採用相同的數學架構，但缺少逐行比對與具體測試向量；需要檢視對稀疏性懲罰（thresholding）與 normalization 的處理是否一致。

## 逐行重要差異摘錄

- 稀疏化/懲罰項：確認 Go 與 R 在懲罰函數（例如 penalize top-n 或 threshold）的實作細節。
- 梯度連續性：simplimax 在極端稀疏情況可能有非微分點，需確認 Go 的處理（是否做平滑近似）。

## 相容性風險與建議

- 若懲罰/threshold 策略差異，最終 loadings 的稀疏結果會不同。建議把 threshold 與懲罰選項暴露並預設與 R 一致。
- 建議對非微分點採用穩健化處理（小的 epsilon）並在文件中註明。

## 測試建議（最小集合）

1. 稀疏性驗證：建立包含明顯稀疏與非稀疏的 loadings 矩陣，比較 R 與 Go 的輸出模式（哪幾個元素被「壓為 0」）。
2. 敏感度測試：測試不同 threshold 與 penalty 參數對結果的影響。

## 優先次序

1. 驗證懲罰策略與 threshold（中高），2. 測試（中），3. 文件（低）。

## 下一步

- 我會提取 R 的 `vgQ.simplimax` 函數原型並產生 5 組具代表性的測資，放到 fixtures 目錄以便 Go 測試使用（需你確認是否要我建立 fixtures）。
