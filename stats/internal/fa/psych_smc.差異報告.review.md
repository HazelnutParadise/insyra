# psych_smc.go — 差異報告（審閱提案）

注意：此檔為自動產生的審閱提案，原始檔 `psych_smc.差異報告.md` 保持不變。

檔案: psych_smc.go
對應 R 檔案: psych_smc.R

## 摘要（高階差異）

- 目的：比較 Go `Smc` 與 R `psych::smc` 在 SMC（squared multiple correlation）計算時，對缺失值（NA）、pairwise correlation 與 covariance (covar) 選項的處理差異。
- 結論（建議）：在完整資料下數值接近；建議優先補強 pairwise/NA 路徑與 SVD/pinv fallback，並回傳豐富 diagnostics 以便比對。

## 逐段差異（重點）

1. covar 參數與對角變異處理

- R 行為：covar=true 時保留對角變異並在必要時做回填。
- 建議（Go）：補齊 covar 路徑使其行為與 R 等價，或明確文件說明差異。

1. NA / pairwise correlation 處理

- R 行為：使用 tempR 與子矩陣策略處理 NA，並提供 warning / fallback。
- 建議（Go）：實作 pairwise 路徑並在不可逆情況下採 SVD/pinv fallback，回傳 diagnostics（wasImputed、imputationMethod）。

1. 數值穩健性與 fallback

- 建議：使用 SVD/pinv 來處理奇異 predictors，並避免以極端預設值回應。返回 error 或 diagnostics 供上層判斷。

## 相容性風險與建議

- 高風險：含 NA 的實務資料上會與 R 產生差異 → 優先補強。

## 測試建議（具體）

> fixtures 路徑：`tests/fa/fixtures/smc`

1. 準備 6 組測試矩陣：正常、含 NA（不同型態）、病態（高 condition number）。
1. 每組由 R 產出金標（aligned CSV / diagnostics JSON），在 Go 測試中比較逐元素差（容差可先設 1e-8/1e-6）。

## 優先次序（建議）

1. pairwise/NA 路徑（高）
2. SVD/pinv fallback（高）
3. 加入 diagnostics（中）

## 下一步

- 我可以自動產生 `tests/fa/fixtures/smc` 的 PoC fixtures（R 產生）並新增一個 minimal Go test harness。
