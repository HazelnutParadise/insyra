## ADDED Requirements

### Requirement: A streaming filter returns every matching row

`parquet.FilterWithCCL` SHALL return every row of the file that satisfies the expression, whatever the file's size relative to the internal batch size. It SHALL NOT append into a value obtained from `DataTable.GetColByNumber`, which returns a copy.

#### Scenario: A file larger than one batch
- **WHEN** 對一個 2500 列的檔案套用每列都成立的條件
- **THEN** 回傳 2500 列，而不是第一批的 1000 列

#### Scenario: Matches spread across batches
- **WHEN** 條件只在第 500 列與第 2000 列成立
- **THEN** 兩列都出現在結果中，順序與檔案相同

### Requirement: A read error is reported, never traded for a partial result

`streamAsArrowRecord` 的錯誤通道與紀錄通道由同一個 goroutine 一起關閉，因此兩者同時就緒時 `select` 會隨機挑一個。紀錄通道關閉後，`FilterWithCCL`、`ApplyCCL` 與 `Stream` SHALL 先讀取錯誤通道再結束，不得讓 `select` 在兩者之間挑選。讀取失敗時 SHALL 回傳錯誤，SHALL NOT 回傳只含部分資料而 error 為 nil 的結果。

#### Scenario: The producer finishes before the consumer looks again
- **WHEN** 讀取在最後一批之後失敗，而消費端正在處理前一批
- **THEN** 該錯誤被回報，不會被關閉的紀錄通道蓋掉

### Requirement: A successful streaming read logs nothing

串流讀取路徑關閉檔案時 SHALL 忽略 `os.ErrClosed`，與 `Read` 和 `Inspect` 一致。成功的 `Stream`、`FilterWithCCL` 或 `ApplyCCL` 呼叫 SHALL NOT 產生關檔警告。

#### Scenario: A successful filter
- **WHEN** `FilterWithCCL` 正常完成
- **THEN** 沒有 `failed to close file` 警告
