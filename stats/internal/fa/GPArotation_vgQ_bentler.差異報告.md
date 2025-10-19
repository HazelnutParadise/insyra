# GPArotation_vgQ_bentler.go — 差異報告

檔案: GPArotation_vgQ_bentler.go
對應 R 檔案: GPArotation_vgQ.bentler.R

## 摘要

- 目的：驗證 Bentler criterion 的 log-det / inverse 數值穩健性處理。

## 測試建議

- fixtures 路徑：`tests/fa/fixtures/bentler`

## 修正記錄

### 2024-12-XX

- ✅ 修改了`vgQBentler`函數：將panic改為返回error，處理矩陣奇異情況
- ✅ 更新了函數簽名：從`(Gq *mat.Dense, f float64, method string)`改為`(Gq *mat.Dense, f float64, method string, err error)`
- ✅ 更新了調用者：修改了`GPArotation_GPForth.go`和`GPArotation_GPFoblq.go`中的調用以處理error
