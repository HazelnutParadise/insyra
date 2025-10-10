# psych_faRotations.go — 差異報告

檔案: psych_faRotations.go
對應 R 檔案: psych_faRotations.R

## 摘要（高階差異）

- 目的：說明 `fa.go` / `psych_faRotations.go` 在 rotation dispatch 與各 rotation method 的對應行為，並比較 Go 與 R 在參數預設、錯誤處理與回傳診斷資訊的差異。
- 結論：Go 的 dispatcher 功能與 R 的 `psych::faRotations` 對應，但在錯誤處理（panic vs warning/fallback）、參數預設以及回傳診斷欄位上存在差異，建議優先降低 panic、補齊診斷欄位並加入 fixtures 進行比對。

## 逐段差異（重點）

1. Dispatch 與 method mapping

- Go：使用 typed `RotOpts` 與內建 dispatch（switch/map）直接呼叫對應 rotation 函式，偏向內建所有子演算法以移除外部依賴。
- R：`psych::faRotations` 在需要時可依賴 `GPArotation` 套件，缺套件時採取 fallback 或 warning。

1. 參數預設與型別

- Go：`RotOpts` 為 struct，型別嚴格，但部分預設值以 zero value 表示，建議文件明確標注哪些欄位為必填。
- R：參數預設在函式中定義，並使用 `NULL`/missing 進行檢查。

1. 錯誤處理與 fallback

- Go：某些子函式遇數值錯誤（奇異矩陣、SVD 失敗）會 `panic`，若上層未捕捉則終止流程。
- R：傾向使用 `try`/`warning`/fallback 保持流程不中斷。

1. 回傳內容與診斷資訊

- Go：回傳以核心資料（loadings、method 等）為主，缺少 `converged`、`iterations`、`restarts`、`fit` 等細項欄位。
- R：回傳物件含豐富診斷資料，利於使用者評估 rotation 結果可靠性。

## 相容性風險與建議

- 高優先：將 panic 行為調整為返回 error 或允許上層捕捉（以 match R 的非致命錯誤處理），並在文件中列出 `RotOpts` 的預設值與必填欄位。
- 中優先：補齊回傳診斷欄位（converged/iterations/restarts/fit/complexity）。

## 測試建議（具體）

1. 功能驗證：挑選 5 組代表性 loadings（不同 p/k、含 sparse/dense），在 R 與 Go 上使用相同 method 做 rotation，比較 loadings、Phi、converged 與 fit 指標。
2. 邊界情形：模擬 NaN/Inf、奇異矩陣、及缺少外部套件情形，驗證 Go 是否會 panic，並改為返回 error 或採 fallback。

## 優先次序（建議）

1. panic->error（高）
2. 補齊診斷欄位（中）
3. 加入 fixtures（中）

## 下一步

- 在 `tests/fa/fixtures/faRotations` 建立 5 組代表性 fixtures（不同 method 與 seed），並新增整合測試 harness 來比較 R 與 Go 的輸出（loadings、Phi、converged）。
