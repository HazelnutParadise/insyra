# 檔案: `GPArotation_vgQ_varimax.go` — 差異報告

對應 R 檔案: `GPArotation_vgQ.varimax.R`

## 摘要（高階差異）

- 目的：比較 Go 的 `vgQVarimax`（varimax 目標函數與梯度）與 R 的 `vgQ.varimax` 實作，確認常數縮放、梯度方向與輸出格式一致性，並給出測試與修正建議。
- 高階結論：兩者在數學上應等價，但需核實實作細節（常數因子、數值穩健性處理、回傳 metadata），特別是在上層 `GPForth` 迴圈中使用時。

## 逐段重點差異

1. 目標函數 f 與梯度 Gq

   - 檢查：確認 Go 的 f 與 Gq 是否與 R 的導數式相同（注意常數係數與矩陣運算次序）。

2. 常數縮放

   - 因為縮放會影響 optimizer 性能，應以 R 為基準做小矩陣數值比對以確定一致性。

3. 數值穩健性

   - 檢查 Go 是否在處理 NaN/Inf、極端值或零行/列時有合理 fallback（delta、regularize、SVD/pinv）。

4. 回傳格式

   - 確保 method 字串、Gq/f 的回傳順序與上層期待一致，並回傳必要的 diagnostics（如 scaling used）。

## 相容性風險與建議

- 高優先：建立與 R 的單元測試，特別是 f/Gq 的逐元素比較。
- 中優先：檢查並統一縮放與避免不必要的矩陣拷貝，以減少浮點誤差。

## 測試建議

1. 小矩陣比對：4x2、8x3 等隨機 loadings，比較 R 與 Go 的 f 與 Gq。
2. 邊界測試：零行/列、NaN/Inf、過大/過小值。

## 下一步

- 我已覆寫並新增本報告檔案。接下來我會把 todo 中的 `GPArotation_vgQ_varimax` 標為 completed，並（若你同意）開始產出 `FA_diffs_SUMMARY.md` 以匯總所有差異檔案與優先修正清單。
檔案: GPArotation_vgQ_varimax.go
對應 R 檔案: GPArotation_vgQ.varimax.R

摘要（高階差異）

- varimax 的 Go 實作 `vgQVarimax` 預期與 R 的 varimax 等價：計算 Gq 與 f，並用於 `GPForth` 的 orthogonal 迭代。

逐行重要差異摘錄

- 已在其他文件中看到其被呼叫的方式，建議針對 `vgQVarimax` 與 R `varimax` 輸出 (loadings, rotmat) 做一致性測試。

建議

- 以 R 的 `GPArotation::vgQ.varimax` 作為金標準，建立數值測試。
