# psych_smc.go — 差異報告

檔案: psych_smc.go
對應 R 檔案: psych_smc.R

## 摘要（高階差異）

- 目的：比較 Go `Smc` 與 R `psych::smc` 在 SMC（squared multiple correlation）計算時，對缺失值（NA）、pairwise correlation 與 covariance (covar) 選項的處理差異。
- 結論：在完整無缺值情境下兩者數值接近；但含 NA 或 covar=true 時，R 提供更完整的處理（`tempR`、子矩陣選取、maxr、imputation/fallback），而 Go 目前採簡化流程，可能導致結果差異或更脆弱的行為。

## 逐段差異（重點）

1. covar 參數與對角變異處理

- R：若使用 covariance matrix（covar=true），會保留對角變異資訊並在必要時乘以 vari 回填。
- Go：目前主要以 correlation matrix 為輸入，covar 選項的等價行為尚未完整實作。

1. NA / pairwise correlation 處理

- R：實作會建立 `tempR`，選取可行子矩陣並使用 maxr 或其他策略填補缺失，流程中會發出 warning 或採 fallback。
- Go：pairwise/NA 路徑不完整；遇 predictors 不可逆時，現有策略可能以極端預設值（如把 SMC 設為 1）回應，非使用 SVD/pinv fallback。

1. 數值穩健性與 fallback

- 建議在 Go 中加入 SVD/pinv fallback，並在 idx 為空或 predictors 異常時回傳可解釋的 error/diagnostics，而非直接返回極端預設值。

1. 回傳資料的豐富度

- 建議在回傳值中加入 diagnostics 欄位（如 `covarUsed`、`wasImputed`、`imputationMethod`、`converged`），以便與 R 的行為比對。

## 相容性風險與建議

- 高風險：含 NA 的實務資料上，Go 與 R 可能產生不同 SMC 值，建議優先補強 pairwise/NA 路徑與 SVD fallback。
- 中風險：增加 diagnostics 與文件化，幫助使用者理解差異。

## 測試建議（具體）

1. 與 R 的金標比較：準備 6 組測試矩陣（含 NA 與病態矩陣），比較 R 與 Go 的 SMC 逐元素差。
2. Pairwise 測試：建立包含 NA 的樣本，驗證 Go 在 pairwise 處理上的行為與效能。
3. 數值異常測試：模擬 predictors 為奇異或接近奇異，驗證 Go 的 fallback 與 error 回傳。

## 優先次序（建議）

1. 補強 pairwise/NA 處理（高）
2. SVD/pinv fallback 與 diagnostics（高）
3. 加入 fixtures（中）

## 下一步（可執行）

- 在 `tests/fa/fixtures/smc` 放入由 R 產生的代表性案例（含 NA、病態、正常），並新增 minimal Go harness 以對照輸出（我可以幫忙生成 fixtures）。
