## ADDED Requirements

### Requirement: WriteLP writes the LP text that GenerateLPFile saves

`(*LPModel).WriteLP(w io.Writer) error` SHALL 把模型寫成 LP 格式的文字，且 `GenerateLPFile` SHALL 維持沒有回傳值的簽名並寫出與 `WriteLP` 相同的位元組。

#### Scenario: The same bytes
- **WHEN** 同一個模型分別以 `WriteLP` 與 `GenerateLPFile` 輸出
- **THEN** 兩者位元組相同，並與 0.3.x 寫出的檔案相同

#### Scenario: An unknown objective type
- **WHEN** 一個 `ObjectiveType` 為 `"sideways"` 的模型呼叫 `WriteLP`
- **THEN** 回傳錯誤

### Requirement: ParseLingo and ParseLingoFile read a LINGO model and return an error when they cannot

`ParseLingo(model string)` 與 `ParseLingoFile(path string)` SHALL 回傳 `(*LPModel, error)`，對同一段文字回傳相同的模型，並讀取目標函數、限制式、邊界與 `@BIN`、`@INT` 宣告，無法讀取的輸入則回傳 nil 模型與錯誤。

#### Scenario: The same model from text and from a file
- **WHEN** 同一段 LINGO 模型分別以 `ParseLingo` 讀字串、以 `ParseLingoFile` 讀檔
- **THEN** 兩者都回傳 nil 錯誤，模型相同

#### Scenario: A missing file
- **WHEN** `ParseLingoFile` 讀取不存在的檔案
- **THEN** 回傳 nil 模型與錯誤，`errors.Is(err, fs.ErrNotExist)` 為 true

#### Scenario: A line the reader cannot hold
- **WHEN** `ParseLingo` 讀取的文字有一行超過讀取器的上限（64 KiB）
- **THEN** 回傳 nil 模型與錯誤

### Requirement: The deprecated LINGO names keep their meaning until 0.4

`ParseLingoModel_str` 與 `ParseLingoModel_txt` SHALL 保留既有行為直到 0.4 移除，並各自在 doc comment 的 `Deprecated:` 段落指名替代函式。

#### Scenario: The deprecated file reader on a missing file
- **WHEN** `ParseLingoModel_txt` 讀取不存在的檔案
- **THEN** 回傳 nil，並記錄警告

#### Scenario: The deprecation notices
- **WHEN** 讀取 `ParseLingoModel_str` 與 `ParseLingoModel_txt` 的 doc comment
- **THEN** 各自有指名 `ParseLingo`／`ParseLingoFile` 的 `Deprecated:` 段落
