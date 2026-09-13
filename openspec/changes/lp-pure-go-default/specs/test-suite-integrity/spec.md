## MODIFIED Requirements

### Requirement: Solver-free and bridge code is tested without its external dependency

不需要外部工具就能執行的程式碼 SHALL 有不依賴該工具的測試。`lp` 的 go-milp 引擎與 LP 讀取器 SHALL 在沒有 `glpsol` 的機器上完整測試；解析 GLPK 輸出的函式 SHALL 以實際 `glpsol` 產生的輸出樣本測試，不呼叫 `glpsol`；需要 `glpsol` 的端對端測試 SHALL 在找不到它時跳過而不是失敗。`parquet` 的 CCL 介接層 SHALL 以測試自行寫出的 Parquet 檔案測試 `FilterWithCCL`、`ApplyCCL` 與 `parquetContext` 的存取方法。

#### Scenario: GLPK is not installed
- **WHEN** 在沒有 `glpsol` 的機器上執行 `go test ./lp/...`
- **THEN** go-milp 引擎、LP 讀取器與 GLPK 輸出解析的測試照常執行並通過，需要 `glpsol` 的測試標示為跳過

#### Scenario: A CCL filter over a Parquet file
- **WHEN** 對測試寫出的 Parquet 檔執行 `FilterWithCCL`
- **THEN** 回傳的 DataTable 只含符合條件的列
