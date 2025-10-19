# psych_faRotations.go — 差異報告

檔案: psych_faRotations.go
對應 R 檔案: psych_faRotations.R

## 摘要

- 目的：說明 rotation dispatcher 與各 rotation method 在 Go 與 R 的對應與差異（參數預設、錯誤處理、診斷回傳）。
- 結論（建議）：優先將 panic 行為改為 error 返回、補齊診斷欄位（converged/iterations/restarts/fit）並加入 fixtures 以比對多個 method 的數值行為。

## 逐段差異（重點）

1. Dispatch 與 method mapping

- 建議文件化 `RotOpts`、method 對應表，並在缺少外部套件時提供明確 fallback。

1. 參數預設與型別

- 建議在文件及 unit tests 中明確列出 `RotOpts` 的預設值，並在必要時加入 input validation（必填欄位檢查）。

1. 錯誤處理與 fallback

- 建議：把子函式可能的 panic 轉為返回 error 或讓上層能夠安全捕捉與 fallback。

1. 回傳內容與診斷資訊

- 建議補齊常見診斷欄位（converged/iterations/restarts/fit/complexity），以便與 R 的結果做詳細比對。

## 測試建議（具體）

- fixtures 路徑：`tests/fa/fixtures/faRotations`

1. 建立 5 組代表性 loadings（不同 p/k 與稀疏／密集性），並對每組在 R 與 Go 以多個 method（varimax/promax/oblimin）執行比較。

## 優先次序（建議）

1. panic->error（高）
1. 補齊診斷欄位（中）
1. 建立 fixtures（中）

## 下一步

- 我可以建立 `tests/fa/fixtures/faRotations` 的 PoC fixtures 並新增整合測試 harness 來自動化比較 R 與 Go 的輸出。
