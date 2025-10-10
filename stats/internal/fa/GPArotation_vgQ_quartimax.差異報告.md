檔案: GPArotation_vgQ_quartimax.go
對應 R 檔案: GPArotation_vgQ.quartimax.R

## 摘要（高階差異）

- 目的：此檔案在 Go 端負責計算「quartimax」類型的 vgQ 目標函數與其梯度（Gq）——用於 GPA rotation 演算法中的變換評估。
- 高階結論：Go 的實作風格與 repo 中其他 `vgQ` 家族函式一致，數學表達（以 L 的元素平方、行/列彙總與逐元素運算為主）應可與 R 的 `GPArotation_vgQ.quartimax.R` 對齊；目前已實作的主要差異點集中在數值穩定性檢查、錯誤處理與邊界情況的 fallback 行為。

## 逐段重要差異（對應行為與實作細節）

1) 輸入與前置處理

- Go: 預期接受矩陣 `L`（p x k），會在函式內逐元素計算 `L^2` 並進行後續運算。其他 `vgQ` 已見模式是先處理 `L` 的正規化選項。
- R: `GPArotation_vgQ.quartimax.R` 在實作上傾向向量化處理（`L^2 <- L^2`、`M <- crossprod(L2)` 等），並在必要時處理 `delta` 或 `normalize` 參數。

差異重點：兩者數學等價，但需確認 Go 是否在開始前處理了相同的 `delta`/`normalize` 參數與命名行為。

2) 目標函數 f 與梯度 Gq 的表達

- Go（應遵循的一般模式）：
  - 計算 L 的元素平方 L2。
  - 依 quartimax 的定義，f 通常與變異/複雜度相關（或與 row/col 的某種加權總和有關），Gq 則是依據 f 對 L 的導數，常呈逐元素形式（L * 某矩陣或逐元素乘法）。
- R: 具體實作會使用向量化運算得到 f（標量）與 Gq（p x k 矩陣）。

差異重點：確認 Go 是否在計算 Gq 時使用了與 R 相同的 intermediate（例如是否用 `diag`/`rowSums`/`crossprod` 等相同矩陣操作），以及在 f 的縮放/常數因子（例如是否除以 4、或乘上 2/k）上的一致性。

3) 數值穩健性與錯誤處理

- Go: 在 repo 中的其他 `vgQ` 實作（如 geomin、bentler、oblimin）可見對零值/奇異情況以明確檢查（加 delta、檢查 inverse/svd 成功與否），但也常在無法處理時採用 `panic`。
- R: `GPArotation` 與 `psych` 通常在錯誤時以 `try`/`warning`/fallback（改用其他方法）處理，而非中止整個流程。

差異重點：建議在 Go 實作中加入錯誤回傳（非 panic），並在數值上使用小常數 (delta) 或 SVD-based 穩健反演以避免 det/solve 的崩潰。

4) 回傳格式與 metadata

- Go: 與其他 `vgQ` 函式一致地回傳 `(Gq *mat.Dense, f float64, method string)` 三元，直接供上層 `GPForth/GPFoblq` 調用。
- R: 同樣回傳結構化物件，並在 caller（GPArotation）層處理 method 字串與 Table/Th。

差異重點：一致性高，需確認 method 字串命名完全匹配以利上層比對。

## 相容性風險與優先度建議

- 高優先（應盡快處理）：

 1. 錯誤處理：將可能的 panic 改為返回 error（或在最上層捕捉），以便在輸入病態矩陣時可以 fallback（R 的行為模式）。
 2. 數值穩健性：在計算包含 log / det / 除法 的步驟前加入小常數（delta）或改用更穩健的演算法（SVD/log-det）。

- 中優先：

 3. 驗證常數與縮放：確認 f 與 Gq 的常數係數（例如 2/k、1/4 等）與 R 端一致。
 4. 測試：建立可重複的單元測試（含邊界情況、隨機種子）來比對 R 的輸出。

- 低優先：

 5. 文件化：在函式註解中加入對應 R 檔案與數學式註解，便於未來維護。

## 建議的修正與測試向量

1) 錯誤處理修正（範例）：

- 若目前函式使用 `panic`，改為 `func ... (...) (*mat.Dense, float64, string, error)` 並在失敗時返回錯誤。

2) 數值穩健性測試向量（最小集合）：

- 正常矩陣：隨機 p=50, k=5 的 loadings 矩陣，比對 R 與 Go 的 f 與 Gq（相對誤差 < 1e-8）。
- 接近零元素：在某些元素加入 0，測試 log / 除法 的處理（若使用 delta，驗證 delta 的合理範圍）。
- 病態矩陣：構造接近奇異的 L，驗證 Go 不會 panic，且在返回 error 時上層能夠 fallback。

3) 驗證常數：

- 針對 10 組固定小矩陣（由 R 產生並保存結果），在 Go 端比對 f 與 Gq（逐元素）以確保常數與方向一致。
