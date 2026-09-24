# io-error-hygiene Specification

## Purpose
`csvxl` 與 `parquet` 的 I/O 錯誤處理契約：錯誤可 unwrap、目錄權限保守、`parquet.Write` 關閉錯誤不被吞掉、日誌走 Insyra logger。

## Requirements

### Requirement: csvxl errors unwrap and directories are 0755

`csvxl` 的錯誤 SHALL 以 `%w` 包裝底層錯誤；`ExcelToCsv`／`EachExcelToCsv` 建立的目錄權限 SHALL 為 0755。

#### Scenario: Missing workbook
- **WHEN** `ExcelToCsv("/definitely/not/here.xlsx", dir, nil)`
- **THEN** 回傳的錯誤 `errors.Is(err, os.ErrNotExist)` 為 true

#### Scenario: Auto-detection fails
- **WHEN** 以 `Auto` 對一個打得開但讀不了的路徑（例如目錄）呼叫 `ReadCsvToString`
- **THEN** 回傳的錯誤可用 `errors.As` 取得底層的 `*fs.PathError`

#### Scenario: Output directory is not world-writable
- **WHEN** `ExcelToCsv` 建立原本不存在的輸出目錄
- **THEN** 該目錄權限不含其他人的寫入位元

### Requirement: parquet.Write reports close errors and logs through Insyra

`parquet.Write` 在關閉 writer（寫入檔尾）失敗時 SHALL 回傳錯誤，SHALL NOT 只記錄後回傳 nil。`parquet` 套件 SHALL NOT 使用標準 `log` 套件輸出，其他關閉時的錯誤 SHALL 經 `insyra.LogWarning` 記錄。

#### Scenario: Successful write
- **WHEN** `Write(dt, path)` 成功
- **THEN** 回傳 nil，且 `Read` 可讀回同樣的資料

### Requirement: Remote input is bounded

讀取遠端回應 SHALL 經過大小上限，SHALL NOT 直接 `io.ReadAll`。發出的 HTTP 請求 SHALL 設定整體 timeout。

#### Scenario: A reply larger than the cap
- **WHEN** 線上渲染服務回傳超過上限的內容
- **THEN** 回報錯誤，不把它寫進檔案，也不把它整個讀進記憶體

### Requirement: Created directories are not world-writable

程式庫建立的目錄 SHALL NOT 使用 `os.ModePerm`（0777）。

#### Scenario: The Python environment directory
- **WHEN** `py` 建立安裝目錄
- **THEN** 權限是 0o755

### Requirement: A framing layer refuses what it cannot read back

`ipc.WriteMessage` SHALL 在寫入任何位元組之前拒絕超過 `maxMessageSize` 的訊息。寫入端接受的長度 SHALL 都在讀取端接受的範圍內。

#### Scenario: An oversized payload
- **WHEN** 寫入超過上限的訊息
- **THEN** 回傳錯誤，且串流上沒有留下任何位元組
