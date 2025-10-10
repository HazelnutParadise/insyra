# psych_target_rot.go — 差異報告（審閱提案）

注意：此檔為自動產生的「審閱提案」，我不會覆寫原始 `psych_target_rot.差異報告.md`，請在確認變更後告訴我是否要套回原檔或直接採用此檔並取代原檔。

檔案: psych_target_rot.go
對應 R 檔案: psych_target_rot.R（包含 `target.rot` 的實作）

## 摘要（高階差異）

- 目的：比較 Go 與 R 在 target rotation（以目標矩陣對齊 loadings）上的行為差異，重點在於目標矩陣的 mask/缺值處理、符號與欄位排列不確定性、以及 oblique（Phi）矩陣的更新與回傳。
- 我在原檔中保留了主要結論；下列為我整理成一致模板後的具體建議（含可執行的下一步與 fixtures 路徑）。

## 逐段差異（重點）

1. 目標矩陣輸入與 mask

- R 行為：`target.rot` 使用 `NA` 作為 mask（不參與對齊的元素）。
- 建議（Go）：新增 `mask` 參數或接受 `NaN` 作為 mask，並在文件／返回 diagnostics 註明哪些元素被 mask（放到 `results.diagnostics.maskApplied`）。

1. 符號（sign）與欄位排列（column permutation）不確定性

- 建議：提供一個 deterministic 的 post-processing：
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

1. 基本案例（`tests/fa/fixtures/target/basic`）：
   - 3 組小型 loadings 與 target（含部分 NA），每組包含 R 輸出 CSV（loadings_aligned.csv, rotation_matrix.csv, diagnostics.json）。
2. Phi 行為（`tests/fa/fixtures/target/phi`）：
   - 2 組（orthogonal / oblique），檢查 `Phi` 與 residual norm。
3. Mask 邊界（`tests/fa/fixtures/target/mask-boundary`）：
   - 全部 NA、無 NA、部分 NA，檢查程式是否返回 error 或正常完成與 diagnostics。

> 每組 fixture 建議包含：原始 loadings（loadings.csv）、target（target.csv）、R 產生的對齊後 loadings（aligned.csv）、rotation matrix（rot.csv）、diagnostics（diagnostics.json）。

## 優先次序（建議）

1. 優先支援 mask/NA 與 diagnostics（高）
2. 提供 deterministic column matching / sign normalization（中高）
3. 建立 fixtures 與 minimal Go test harness（中）

## 下一步（我可以替你執行）

1. 若你同意，我會：
   - 從 `local/r_source_code_fa` 執行對應 R 程式產生上述三個 fixture 類別（CSV / JSON），將檔案放到 `tests/fa/fixtures/target/`。
   - 產生一個 minimal Go test（`stats/internal/fa/target_rotation_test.go`）來載入 fixtures，執行 Go 的 target rotation，並比對 R 與 Go 的 aligned loadings（允許例如 1e-6 的絕對或相對容差，視情況放寬）。
2. 若你想我先只產生 proposal 的差異報告修改 patch（不覆寫原檔），我已把本檔（`psych_target_rot.差異報告.review.md`）放在相同資料夾供你審閱。

---

如果你接受我以 A（繼續下一批差異報告）進行，我會：

- 繼續下一批 4–6 個差異報告（從 `stats/internal/fa` 清單中依序處理），每批都遵循 read-before-write；

-- 在每批寫入前重新 read 以確保不覆蓋你手動修改的檔案；如發現已被你改動，改為建立 `.review.md` 提案檔。
