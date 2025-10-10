# 檔案: `psych_smc.go` — 差異報告

對應 R 檔案: `psych_smc.R`

## 摘要（高階差異）

- 目的：比較 Go `Smc` 與 R `psych::smc` 在 SMC（squared multiple correlation）計算中對缺失值（NA）、pairwise correlation 與 covariance 選項（covar）的處理差異，並提出修復與測試建議。
- 高階結論：在完整無缺失的相關矩陣下，兩者的數學等價；主要差異在於 R 提供了更細緻的 NA/pairwise 處理、covar 回填邏輯與多重 fallback。Go 的實作目前較簡化，可能在含缺失或 covar=true 的情況下給出不同結果或採用不同 fallback 策略。

## 逐段重點差異

1. covar 參數與對角變異處理

   - R：若使用 covariance matrix（covar=true），會保留對角的變異資訊，並在必要時將計算結果乘以 vari（對角變異）回填。
   - Go：目前偏向以 correlation matrix 為主要輸入，covar 選項的對應行為不完整或未實作。

2. NA / pairwise correlation 處理

   - R：實作中有建立 `tempR`、選取可行子矩陣並用 maxr 或其它方式填補缺失，且會在過程中發出 warning 或使用 fallback 方法。
   - Go：缺少完整的 pairwise/NA 路徑；遇到 predictors 不可逆時，現有行為是採用簡化策略（例如將 SMC 設為 1）而非用子矩陣或 SVD-based fallback。

3. 數值穩健性與 fallback

   - 建議在 Go 中加入 SVD/pinv fallback、以及在 idx 為空或 predictors 奇異時返回可解釋的 error 或 diagnostics，而非直接返回極端預設值。

4. 回傳資訊的豐富度

   - 建議在返回值中加入 diagnostics 欄位（如 `covarUsed`、`wasImputed`、`imputationMethod`、`converged`）以便與 R 的行為比對。

## 相容性風險與優先建議

- 高：在含 NA 的實際資料上，現有 Go 實作可能與 R 給出不同 SMC 值，需優先補齊 pairwise/NA 處理邏輯。
- 中：在 predictors 奇異或接近奇異時，需以 SVD/pinv 取代 current fallback 並回傳 error/diagnostics。

## 測試建議（最小集合）

1. 與 R 的金標比較：使用多組資料（含 NA 與無 NA），在 R（psych::smc）與 Go 上執行比較 SMC 值與填補差異。
2. Pairwise 測試：建立包含 NA 的樣本，檢驗 Go 在 pairwise 處理上的行為與效能。
3. 奇異矩陣測試：模擬 predictors 為奇異或接近奇異的情況，測試 SVD/pinv fallback 是否正常工作與 error 的可解釋性。
