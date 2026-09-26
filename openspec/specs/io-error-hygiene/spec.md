# io-error-hygiene Specification

## Purpose
`csvxl` 與 `parquet` 的 I/O 錯誤處理契約：錯誤可 unwrap、目錄權限保守、寫檔不留截斷檔、日誌走 Insyra logger。
## Requirements
### Requirement: csvxl errors unwrap and directories are 0755

`csvxl` 的錯誤 SHALL 以 `%w` 包裝底層錯誤；`ExcelToCsv`／`EachExcelToCsv` 建立的目錄權限 SHALL 為 0755。

#### Scenario: Missing CSV
- **WHEN** `CsvToExcel([]string{"/nope.csv"}, nil, out)` 
- **THEN** 回傳的錯誤 `errors.Is(err, os.ErrNotExist)` 為 true

### Requirement: parquet.Write is atomic and logs through Insyra

`parquet.Write` SHALL 先寫暫存檔再 rename；`parquet` 套件 SHALL NOT 使用標準 `log` 套件輸出。

#### Scenario: Successful write leaves no temp file
- **WHEN** `Write(dt, path)` 成功
- **THEN** 目錄中只有 `path`

### Requirement: A named sheet that is not there is an error

`ExcelToCsv` 收到 `onlyContainSheets` 時，其中不存在於檔案的名稱 SHALL 回報錯誤並列出檔案實際有的工作表，SHALL NOT 靜默略過。

#### Scenario: A misspelled sheet name
- **WHEN** `onlyContainSheets` 含一個檔案裡沒有的名稱
- **THEN** 回傳錯誤，指出缺少哪些、檔案有哪些

### Requirement: Remote and compressed input is bounded

讀取遠端回應 SHALL 經過大小上限，SHALL NOT 直接 `io.ReadAll`。發出的 HTTP 請求 SHALL 設定整體 timeout。讀取 Excel SHALL 設定 `UnzipSizeLimit`，不得沿用 excelize 的 16 GB 預設值。

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

### Requirement: An accept loop does not spin

IPC 伺服器的 accept 迴圈在監聽器已關閉時 SHALL 結束，僅在逾時錯誤時 SHALL 繼續，其餘錯誤 SHALL 回報一次後結束。連線 SHALL 設定讀寫期限，socket 檔 SHALL 在行程結束時移除。

#### Scenario: A permanently failing listener
- **WHEN** `Accept` 持續失敗
- **THEN** 迴圈結束並回報一次，而不是每次迭代印一行警告

