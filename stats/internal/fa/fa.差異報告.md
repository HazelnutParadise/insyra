檔案: fa.go
對應 R 檔案: psych_fa.R (wrapper/高階呼叫)，psych_faRotations.R (rotation wrapper)
檔案: `fa.go`

對應 R 檔案: `psych_fa.R`（外層 wrapper / 控制流程）、`psych_faRotations.R`（旋轉相關邏輯）

## 摘要（高階差異）

- 目的：說明 `fa.go` 暴露的 API（`SMC`、`Rotate`）與 R 的 `psych::fa`/`psych::faRotations` 在功能與實作細節上的差異。
- 高階結論：Go 實作在核心演算法上（呼叫 `Smc`、`FaRotations`）與 R 對應功能相符，但在輸入驗證、缺值（NA）處理、錯誤處理與預設參數上有可見差異，會導致在極端或有缺值的情境下結果不一致或行為不同（panic vs warning/fallback）。

## 逐段重點差異

1. `SMC`（與 `Smc` 關聯）

- R 層面：`psych::smc` 支援傳入相關矩陣或原始資料矩陣，並在存在 NA 時執行複雜的替代/回填策略（建立 `tempR`、剔除特定變數、用非 NA 的相關估計 SMC、以 `maxr` 補值等）。同時支援 `covar` 選項以回傳變異尺度下的 SMC。

- Go 層面：`SMC` 為 `Smc` 的 wrapper；若輸入非相關矩陣，會以簡化的 `CorrelationMatrix` 計算相關矩陣。`Smc` 在子矩陣不可逆時將 SMC 直接設定為 1；`CorrelationMatrix` 未實作 pairwise/NA 的完整處理。

- 影響：在含 NA 或需要 pairwise 的場合，Go 版可能輸出與 R 顯著不同結果。

2. `Rotate`（對應 `faRotations`）

- R 層面：`faRotations` 在選擇旋轉方法時會檢查外部套件（例如 `GPArotation`）是否可用，並在方法失敗時提供 warning 與 fallback（例如使用 `Promax`），同時回傳豐富的統計資訊（hyperplane, fit, complexity, indetermin, Table 等）。

- Go 層面：`Rotate` 會以 identity r 作為預設，直接呼叫內部 `FaRotations`/GPArotation 的 Go 實作並回傳 `loadings, rotmat, Phi, convergence`。Go 實作多數情況回傳欄位相容，但在方法失敗或數值問題上多以 panic 終止。

- 影響：若 rotation 演算法在數值上失敗，Go 端易中斷整個程序；R 則會較優雅地退回。

## 兼容性風險與建議

- 高風險：NA/pairwise 處理（`Smc`），會直接影響 SMC 與後續因子分析結果。建議優先補上 R 相同的處理流程或至少在 API 文件中明確告知限制。

- 中風險：錯誤處理策略（panic vs error/warning）。建議把 library 層的 panic 改為返回 error，由上層決定是否 fallback。

- 低風險：預設參數差異（例如 `RotOpts` 的預設值）——建議在文件中列出並與 R 的預設對照。

## 建議修正與測試

1. 在 `Smc` 補上 NA / pairwise 的處理：實作 R 的 `tempR`、`wcl`、`wcc` 邏輯，或在文件中明確註明不支援 NA。

2. 將 `Rotate` 及底層函式的 panic 改為 error 返回，並在呼叫端實作 fallback 選項（如改用 Promax 或回傳有意義的錯誤訊息）。

- 大部分數值演算法（Pinv、GPArotation 演算法、vgQ 族函式）在 Go 中有對應且實現細節與 R 相近或等價。主要差異集中在輸入資料的 NA 處理、錯誤處理策略（panic vs warning/fallback）與部分預設參數值。

3. 編寫單元測試：使用 R 的 `psych::smc` 與 `GPArotation` 的輸出作為金標準，對若干代表性矩陣（含 NA、病態矩陣、隨機矩陣）驗證 Go 的輸出（SMC 值、旋轉後 loadings、Phi、convergence）。

## 優先次序（建議）

1. 補 NA/pairwise 處理（高），2. panic->error（高），3. Pinv tol 暴露（中），4. 加入測試集（中）。
