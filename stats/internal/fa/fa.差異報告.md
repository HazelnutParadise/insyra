# fa.go — 差異報告

檔案: fa.go
對應 R 檔案: psych_fa.R (wrapper)、psych_faRotations.R (rotation wrapper)

## 摘要（高階差異）

- 目的：說明 `fa.go` 在 API（SMC、Rotate）層面的行為與 R 的 `psych::fa` / `psych::faRotations` 在功能與邏輯上可能的差異，特別是在缺值處理、錯誤處理與參數預設方面。
- 結論：Go 在核心演算法呼叫（Smc、FaRotations）上與 R 大致一致；但在缺值（NA）處理、錯誤處理（panic vs warning/fallback）與預設參數上存在差異，建議優先統一 NA 處理與降低 panic。

## 逐段差異（重點）

1. `SMC`（與 `Smc` 關聯）

- R：`psych::smc` 支援以 correlation matrix 或原始資料輸入，並在存在 NA 時提供複雜替代 / 回填策略（如 tempR、maxr 等），也支援 covar 選項以回填對角變異。
- Go：`SMC` 為 `Smc` 的 wrapper；若輸入非 correlation matrix，會以簡化方式計算 correlation matrix。`Smc` 在子矩陣不可逆時可能以預設值回應（如設為 1），未提供完整 pairwise/NA 處理。

1. `Rotate`（對應 `faRotations`）

- R：`faRotations` 在選擇 rotation method 時會檢查外部套件（如 GPArotation）是否可用，並在失敗時採取 fallback 或發出 warning，且回傳豐富的診斷資訊（fit、complexity 等）。
- Go：`Rotate` 會直接呼叫內部 FaRotations/GPArotation 實作並回傳 loadings、rotmat、Phi 等，但在方法失敗或數值問題時容易以 panic 結束流程。

1. 錯誤處理與回傳結構

- 建議 Go 在 library 層避免直接 panic，改為回傳 error，並補充回傳結構以包含 diagnostics（如 `converged`、`iterations`、`fit`、`complexity`）。

## 相容性風險與建議

- 高風險：NA / pairwise 處理（SMC、Rotate）導致的輸出差異。
- 建議：優先補強 NA 處理、加入 diagnostics、並將 panic 行為改為 error 返回。

## 測試建議（具體）

1. 建立整合測試：使用 R 作為金標（psych::smc + psych::faRotations）生成代表性 fixtures（包含 NA、病態矩陣、不同 methods 與 seeds），放在 `tests/fa/fixtures`，並新增 Go harness 進行比較（loadings、Phi、SMC、converged）。
2. 單元測試：對 SMC、Pinv、GPForth/GPFoblq 等核心函式建立單元測試以驗證穩健性。

## 下一步

- 在 `tests/fa/fixtures` 中建立代表性 fixtures（包含 NA、病態、正常），並新增 minimal Go test harness 用以自動化比較 R 與 Go 的輸出。
