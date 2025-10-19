# GPArotation_vgQ_oblimin.go — 差異報告

檔案: GPArotation_vgQ_oblimin.go
對應 R 檔案: GPArotation_vgQ.oblimin.R

## 摘要

- 目的：比較 oblimin family 在 Gq/f 計算與 delta/gamma 處理上的差異，並指出 Go 實作需要注意的數值穩定性與測試要點。

## 逐段差異（重點）

1. 計算流程：Go 與 R 主要流程等價 —— 先計算 L^2，再乘上非對角矩陣 (notDiag)。若 gamma != 0，則套用 (I - gamma/p * J) 變換。Gq 與 f 的計算公式在兩邊一致。
2. 類型處理：Go 以數值矩陣代替 R 的 boolean/logic matrix（以 0/1 表示），實作上等價但需注意 dtype 轉換時的數值穩定性。
3. 錯誤處理：Go 版本為函式返回 error（取代直接 panic），增加了運算錯誤與邊界情況的可控性。

## 相容性風險

- 在極端 gamma 值或非常接近奇異的輸入矩陣下，Gq/f 的中間變數可能出現數值不穩定；建議加入 SVD-based fallback 或 tiny-regularization 作為容錯機制。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/oblimin`
- 必要測資：
  1. 一組正常情況（隨機正定矩陣，gamma=0）
  2. gamma 非零的情境（例如 gamma=0.5）
  3. 病態矩陣（接近奇異）以測試 fallback 與容差行為

## 優先次序（建議）

1. 數值穩健性驗證（高）
2. 建立 fixtures 與單元測試（中）
3. 文件化與 API 清晰化（低）

## 下一步

- 在 `tests/fa/fixtures/oblimin` 建立由 R 產生的幾組測資，並新增 Go 單元測試確認 Gq 與 f 在允許的容差內吻合。
