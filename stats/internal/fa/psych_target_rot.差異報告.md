檔案: psych_target_rot.go
對應 R 檔案: psych_target_rot.R (以及 target.rot)

摘要（高階差異）

# 檔案: `psych_target_rot.go` — 差異報告

對應 R 檔案: `psych_target_rot.R`（包含 `target.rot` 實作）

## 摘要（高階差異）

- 目的：比較 Go 與 R 在 target rotation（使用目標矩陣對齊 loadings）的處理細節，包括目標矩陣的缺失值處理、Phi（oblique）補值與 lower-tri/upper-tri 索引慣例。
- 結論：若 Go 已有對應實作，需確認與 R 在目標矩陣讀寫、mask（哪些元素允許對齊）、以及是否考慮符號不確定性（sign indeterminacy）等方面的一致性；若尚未實作，建議照 R 的接口與 fallback 行為來補齊。

## 逐段重點差異

1. 目標矩陣（target matrix）輸入與 mask

- R：`target.rot` 支援在目標矩陣中使用 `NA` 來表示不需對齊的元素，僅根據非 NA 部分計算最小二乘對齊。
- Go：需確認是否支援以 `NaN`/`Inf` 或其他 sentinel 值表示 mask；若未支援，建議加入相同的 mask 機制以達到與 R 的等價行為。

2. 符號與排列不確定性

- R：在對齊時通常會處理 sign / column permutation 的不確定性（視實作而定），並會返回對齊後的 loadings 與 rotation matrix。
- Go：應驗證是否在返回前標準化列的方向或做 column matching，確保與 R 的輸出可比較。

3. Phi（oblique）矩陣的補值與回傳

- R：若執行的是 oblique target rotation，會同時處理 Phi（factor correlation）矩陣的更新與回填。
- Go：應在回傳結果中包含 phi 並確認其計算方式與 R 一致（例如是否強制對稱、是否小幅正則化以避免數值問題）。

4. 錯誤處理與 diagnostics

- R：出錯時以 warning 或 stop 回報，並在某些情況下嘗試 fallback。
- Go：建議返回 error 與 diagnostics（例如 maskApplied、numAligned、residualNorm）以便上層進一步處理。

## 相容性風險與優先建議

- 高優先：加入 target mask 支援（接受 NA/NaN），以匹配 R 的語意。
- 中優先：確認符號/排列對齊策略，並在文件中描寫與 R 的差異。

## 測試建議（最小集合）

1. 簡單目標：建立一個小型 target matrix（含 NA）；在 R 與 Go 上做對齊並比較對齊後的 loadings 與 residual norm。
2. Phi 行為：測試 oblique 與 orthogonal 两種情形，驗證 Phi 在 Go 與 R 的一致性。
3. Mask 邊界：全部 NA 與無 NA 的情況，確認程式行為（error vs no-op）。

## 優先次序（建議）

1. 支援 mask/NA（高），2. 符號/排列對齊策略文件（中），3. 加入 fixtures 與自動化測試（中）。

## 下一步

- 實作或明確文件化 Go 支援的 mask 形式（NaN/Inf），並新增 `tests/fa/fixtures/target` 的三個範例（含 NA/全部 NA/無 NA）。
- 若你贊成，我將自動從 R 產生期望輸出並把 fixtures 與測試 harness 加入 repo。
