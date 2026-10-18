## MODIFIED Requirements

### Requirement: A CSV that fails leaves the workbook as it was

`csvxl.CSVToExcel`、`csvxl.AppendCSVToExcel` 與 `csvxl.CSVDirToExcel` SHALL 在建立或取代工作表之前完整讀取並解碼該 CSV。讀取失敗或工作表無法建立的 CSV SHALL NOT 在輸出中留下工作表，工作簿裡既有的同名工作表 SHALL 維持原內容。批次中其他 CSV SHALL 照常轉換。

#### Scenario: One file in a batch is missing
- **WHEN** `CSVToExcel([]SheetFromCSV{{Path: "good.csv"}, {Path: "missing.csv"}}, out)`
- **THEN** `out` 只有 `good` 工作表，錯誤指名 `missing.csv`，且 `errors.Is(err, os.ErrNotExist)` 為真

#### Scenario: Appending a CSV that cannot be read
- **WHEN** 工作簿的 `data` 工作表有內容，`AppendCSVToExcel` 要以一個不存在的 CSV 取代 `data`，並以另一個可讀的 CSV 取代 `other`
- **THEN** 回傳錯誤，`data` 內容不變，`other` 換成新 CSV 的內容

#### Scenario: A sheet name Excel does not allow
- **WHEN** `CSVToExcel` 其中一個工作表名稱含有 `:`
- **THEN** 錯誤指名該 CSV，其他檔案照常寫入輸出

### Requirement: The workbook is saved only when something was converted

至少一個 CSV 成功時 SHALL 存檔。全部失敗時，`CSVToExcel` SHALL NOT 建立輸出檔，`AppendCSVToExcel` SHALL NOT 改寫工作簿。

#### Scenario: Every file fails
- **WHEN** `CSVToExcel` 的每個 CSV 都讀取失敗
- **THEN** 回傳錯誤，輸出路徑上沒有檔案

#### Scenario: Every appended file fails
- **WHEN** `AppendCSVToExcel` 的每個 CSV 都讀取失敗
- **THEN** 回傳錯誤，工作簿檔案的位元組與呼叫前相同
