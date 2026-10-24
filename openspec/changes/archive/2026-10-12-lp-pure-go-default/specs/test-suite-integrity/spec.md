## MODIFIED Requirements

### Requirement: Solver-free and bridge code is tested without its external dependency

不需要外部工具就能執行的程式碼 SHALL 有不依賴該工具的測試。

#### Scenario: GLPK is not installed
- **WHEN** 在沒有 `glpsol` 的機器上執行 `go test ./lp/...`
- **THEN** go-milp 引擎、LP 讀取器與解析 GLPK 報告的 `parseGLPKOutputFromFile` 測試照常執行並通過，需要 `glpsol` 的端對端測試標示為跳過而不是失敗

#### Scenario: The GLPK report parser on fixed output
- **WHEN** 執行 `parseGLPKOutputFromFile` 的測試
- **THEN** 測試以固定的輸出樣本驗證解析結果，不呼叫 `glpsol`

#### Scenario: A CCL filter over a Parquet file
- **WHEN** 對測試寫出的 Parquet 檔執行 `FilterWithCCL`
- **THEN** 回傳的 DataTable 只含符合條件的列

#### Scenario: The Parquet CCL bridge helpers
- **WHEN** 以測試自行寫出的 Parquet 檔測試 `ApplyCCL` 與 `parquetContext` 的存取方法
- **THEN** 測試照常執行並通過，不依賴事先存在的檔案
