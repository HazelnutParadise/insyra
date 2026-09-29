## ADDED Requirements

### Requirement: Saving a model reports its failure

`(*LPModel).GenerateLPFile(filename string) error` SHALL return an error when the file cannot be created or written and when the model's objective type is not one `WriteLP` accepts, and SHALL NOT log in their place. It SHALL write through a temporary file in the directory of `filename` that is renamed into place only after the whole model was written, so a failed call SHALL NOT leave a file at `filename` or a temporary file behind, and a file already at `filename` SHALL keep its content. A successful call SHALL save exactly the text `WriteLP` writes.

#### Scenario: An unknown objective type
- **WHEN** 一個 `ObjectiveType` 為 `"sideways"` 的模型呼叫 `GenerateLPFile(path)`
- **THEN** 回傳錯誤，`path` 不存在，目錄裡也沒有暫存檔

#### Scenario: An existing file and a failed save
- **WHEN** `path` 已有一個 LP 檔，再以未知的目標型別呼叫 `GenerateLPFile(path)`
- **THEN** 回傳錯誤，`path` 的內容與呼叫前相同

#### Scenario: A directory that does not exist
- **WHEN** 呼叫 `GenerateLPFile` 的路徑位於不存在的目錄
- **THEN** 回傳錯誤，錯誤訊息含該路徑

#### Scenario: A successful save
- **WHEN** 一個合法的模型呼叫 `GenerateLPFile(path)`
- **THEN** 回傳 nil，檔案內容與 `WriteLP` 寫出的文字相同

### Requirement: A LINGO model is read by ParseLingo and ParseLingoFile

`ParseLingo(model string) (*LPModel, error)` SHALL read a LINGO model from text and `ParseLingoFile(path string) (*LPModel, error)` SHALL read one from a file, and the two SHALL return equal models for the same text. A file that cannot be opened, and text that cannot be read, such as a line longer than the reader accepts, SHALL return a nil model and an error; an error from opening a file SHALL still match `fs.ErrNotExist` with `errors.Is` when the file is missing. Everything the old parser read SHALL be read the same way: the objective, constraints, bounds, `@BIN` and `@INT` declarations.

#### Scenario: The same model from text and from a file
- **WHEN** 同一段 LINGO 模型分別以 `ParseLingo` 讀字串、以 `ParseLingoFile` 讀檔
- **THEN** 兩者都回傳 nil 錯誤，模型相同，且與 `ParseLingoModel_str` 讀出的模型相同

#### Scenario: A missing file
- **WHEN** `ParseLingoFile` 讀取不存在的檔案
- **THEN** 回傳 nil 模型與錯誤，`errors.Is(err, fs.ErrNotExist)` 為 true

#### Scenario: A line the reader cannot hold
- **WHEN** `ParseLingo` 讀取的文字有一行超過讀取器的上限
- **THEN** 回傳 nil 模型與錯誤

### Requirement: The old LINGO names keep their meaning for one release

`ParseLingoModel_str` and `ParseLingoModel_txt` SHALL remain for one release, each with a doc comment whose `Deprecated:` paragraph names its replacement, and SHALL return what they returned before: the model, or `nil` with a warning when the text or file cannot be read.

#### Scenario: The deprecated file reader on a missing file
- **WHEN** `ParseLingoModel_txt` 讀取不存在的檔案
- **THEN** 回傳 nil，並記錄警告

#### Scenario: The deprecation notices
- **WHEN** 讀取 `ParseLingoModel_str` 與 `ParseLingoModel_txt` 的 doc comment
- **THEN** 各自有指名 `ParseLingo`／`ParseLingoFile` 的 `Deprecated:` 段落
