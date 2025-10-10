
檔案: `GPArotation_vgQ_geomin.go`

對應 R 檔案: `GPArotation_vgQ.geomin.R`

## 摘要（高階差異）

- 目的：比較 Go 實作與 R 的 geomin 目標函數在數學公式、數值穩健性與實作細節上的一致性。
- 高階結論：Go 在數學上重現 R 的 geomin 公式（L^2+delta、rowSums(log(...))、pro 與梯度），但在數值保護（delta 的使用、division 的邊界）與錯誤處理方面需特別注意以符合 R 的行為。

## 逐段重點差異

1. L2 與 delta 處理

- Go：明確以元素方式計算 L2 = L^2 + delta（delta 預設值可見於呼叫端），並使用 math.Log 與 math.Exp 等逐元素函式。
- R：以向量化函數完成相同運算。行為等價，但實作差異在於邊界數值處理（R 可能有內建 NA/Inf 處理機制）。

2. 梯度計算

- Go：以 (2/k) *(L/L2)* proMat 的逐元素運算計算梯度。
- R：向量化表達相同公式。

## 相容性風險與建議

- 若 L 的某些元素接近 0，log(L2) 與 L/L2 可能導致數值不穩。建議在程式碼中保證 delta 的合理下限（例如 1e-8），或在文件中強調 delta 的重要性。
- 在邊界情況下（含 NA/Inf），應返回 error 或以可預期的方式處理，而非 panic。

## 測試建議

1. 對比 R 的 `vgQ.geomin` 在多組 delta 參數下的 f 與 Gq，確認數值收斂性與敏感度。
2. 構造含 0 值、接近 0 值與極大值的矩陣，驗證無 NaN/Inf，並在必要時加入 delta。

## 優先次序

1. 數值保護（高），2. 錯誤處理（中），3. 測試（中）。
