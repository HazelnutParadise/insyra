# psych_target_rot.go — 差異報告

檔案: psych_target_rot.go
對應 R 檔案: psych_target_rot.R（包含 target.rot 的實作）

## 摘要（高階差異）

- 目的：比較 Go 與 R 在 target rotation（以目標矩陣對齊 loadings）上的行為差異，重點在於目標矩陣的 mask/缺值處理、符號/欄位排列不確定性、以及 oblique（Phi）矩陣的更新與回傳。
- 結論：目前 Go 版需確認或補齊與 R 相同的 mask 機制（R 使用 NA 作為 mask），並在返回值中包含足夠診斷資訊；遇到數值失敗時採取非致命的 fallback（或返回 error）以與 R 的 warning/fallback 行為對齊。

## 逐段差異（重點）

1. 目標矩陣輸入與 mask

- R：`target.rot` 允許在目標矩陣中使用 `NA` 表示該位置不參與對齊，計算僅基於非 NA 的元素。
- 建議（Go）：支援 `NaN`/`Inf` 或明確的 mask 參數，行為應等價於 R（以非缺值元素做最小平方對齊）。

1. 符號（sign）與欄位排列（column permutation）不確定性

- R：實作可能會處理 sign 矯正或 column matching，以便輸出可比較的 loadings。
- 建議（Go）：在回傳前做 column matching / sign normalization 或在報告中記錄使用者應如何比對結果。

1. Oblique（Phi）矩陣的處理

- R：若為 oblique target rotation，會更新並回傳 Phi（factor correlation），通常會確保對稱性並在必要時做小幅正則化。
- 建議（Go）：回傳 phi 並確保其計算與 R 版本一致（檢查對稱性、偶發的正定性問題與是否套用微量 regularization）。

1. 錯誤處理與診斷資訊

- R：遇到數值問題常以 warning 並嘗試 fallback（例如改用其他方法或略過失敗情況），不致中斷整個工作流程。
- 建議（Go）：回傳 `error`（不直接 panic）並在返回結果中包含 diagnostics 欄位（例如 `maskApplied`、`numAligned`、`residualNorm`、`converged`），以便上層決定 fallback 或警告行為。

## 相容性風險與建議

- 高風險項目：若不支援 mask/NA，Go 與 R 在實務資料（含缺值）上會產生不同結果，可能導致 downstream 不一致。建議優先實作 mask 支援。
- 中風險項目：符號與欄位匹配策略若未明確，會使元素層級比對困難，建議在文件中說明 matching 策略或提供一致化工具。

## 測試建議（具體）

1. 基本案例：一組小型 loadings 與 target（含部分 NA），在 R 與 Go 執行 target rotation，比較對齊後的 loadings、rotation matrix 與 residual norm。將輸出放在 `tests/fa/fixtures/target/basic`。
2. Phi 行為：測試 orthogonal 與 oblique 兩種情形，檢查 Phi 值是否一致，放在 `tests/fa/fixtures/target/phi`。
3. Mask 邊界：測試全部 NA、無 NA 與部分 NA 的情形，驗證程式行為（error vs no-op），放在 `tests/fa/fixtures/target/mask-boundary`。

## 優先次序（建議）

1. 支援 mask/NA（高）
2. 返回 diagnostics 並避免 panic（中高）
3. 加入 fixtures 與自動化測試（中）

## 下一步（可執行）

- 在 `tests/fa/fixtures/target/` 新增至少 3 組範例：
  - `basic`（小型示範矩陣，含部分 NA）
  - `phi`（oblique vs orthogonal 比較）
  - `mask-boundary`（全部 NA / 無 NA / 部分 NA）
- 若你同意，我可以：
  1) 從 `local/r_source_code_fa` 使用 R 產生上述 fixtures（CSV/JSON），
  2) 把 fixtures 放入 `tests/fa/fixtures/target/`，並新增 minimal Go 測試 harness 以比對元素差（允許小容差）。
