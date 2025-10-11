# psych_target_rot.go — 差異報告

檔案: psych_target_rot.go
對應 R 檔案: psych_target_rot.R（包含 `target.rot` 的實作）

## 摘要

- 目的：比較 Go 與 R 在 target rotation（以目標矩陣對齊 loadings）上的行為差異，重點在於目標矩陣的 mask/缺值處理、符號與欄位排列不確定性、以及 oblique（Phi）矩陣的更新與回傳。
- 本報告給出可執行的下一步（fixtures、測試 harness、API 改動建議）。

## 逐段差異（重點）

1. 目標矩陣輸入與 mask

- R 行為：`target.rot` 使用 `NA` 作為 mask（不參與對齊的元素）。
- 建議（Go）：新增 `mask` 參數或接受 `NaN` 作為 mask，並在文件／返回 diagnostics 註明哪些元素被 mask（放到 `results.diagnostics.maskApplied`）。

1. 符號（sign）與欄位排列（column permutation）不確定性

- 建議：提供 deterministic 的 post-processing：
  - column matching（利用最大絕對相關或 Hungarian algorithm）
  - sign normalization（例如使第一個絕對值最大元素為正）
  - 或在 diagnostics 中提供 `matchingMap` 與 `signFlips` 供上層比對

1. Oblique（Phi）矩陣的處理

- 建議：回傳 `Phi`（若 oblique），並提供 `PhiSymmetric` 的驗證與 `isPositiveDefinite` 標誌；在必要時套用 tiny-regularization（例如 `Phi + 1e-10*I`）並在 diagnostics 註明。

1. 錯誤處理與診斷資訊

- 建議：不要直接 panic；改為 `return (result, error)`。同時在 `result.diagnostics` 中包含：
  - `maskApplied`（bool 或 mask matrix）
  - `numAligned`（成功對齊的元素數）
  - `residualNorm`（最終殘差）
  - `converged`（bool）
  - `iterations`（int）
  - `warnings`（string list）

## 相容性風險與建議

- 高風險：若不支援 mask/NA，實務資料上會與 R 產生不同結果 → 優先實作。
- 中風險：若不提供 deterministic matching，上下游比對困難 → 建議文件說明或提供 matching helper。

## 測試建議（具體、可執行）

- fixtures 路徑：`tests/fa/fixtures/target`

1. 基本案例（`tests/fa/fixtures/target/basic`）：

- 3 組小型 loadings 與 target（含部分 NA），每組包含 R 輸出 CSV（loadings_aligned.csv, rotation_matrix.csv, diagnostics.json）。

1. Phi 行為（`tests/fa/fixtures/target/phi`）：

- 2 組（orthogonal / oblique），檢查 `Phi` 與 residual norm。

1. Mask 邊界（`tests/fa/fixtures/target/mask-boundary`）：

- 全部 NA、無 NA、部分 NA，檢查程式是否返回 error 或正常完成與 diagnostics。

> 每組 fixture 建議包含：原始 loadings（loadings.csv）、target（target.csv）、R 產生的對齊後 loadings（aligned.csv）、rotation matrix（rot.csv）、diagnostics（diagnostics.json）。

## 優先次序（建議）

1. 優先支援 mask/NA 與 diagnostics（高）
1. 提供 deterministic column matching / sign normalization（中高）
1. 建立 fixtures 與 minimal Go test harness（中）

## 下一步（我可以替你執行）

1. 若你同意，我會：

- 從 `local/r_source_code_fa` 執行對應 R 程式產生上述三個 fixture 類別（CSV / JSON），將檔案放到 `tests/fa/fixtures/target/`。
- 產生一個 minimal Go test（`stats/internal/fa/target_rotation_test.go`）來載入 fixtures，執行 Go 的 target rotation，並比對 R 與 Go 的 aligned loadings（允許例如 1e-6 的絕對或相對容差，視情況放寬）。

## 修正記錄

### 2024-12-XX

- ✅ 添加了mask參數支持：新增了`TargetRotWithMask`函數，支持使用NaN值作為mask來排除目標矩陣中的特定元素
- ✅ 實現了diagnostics返回：返回包含`maskApplied`和`maskedElements`的diagnostics map
- ✅ 保持向後相容性：保留了原有的`TargetRot`函數作為包裝函數
- ✅ NA處理：NaN值被正確識別並從對齊過程中排除（設置為0）

### 剩餘工作

- 更完整的測試覆蓋率
- 符號標準化和欄位匹配的確定性處理
