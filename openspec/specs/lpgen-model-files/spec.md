# lpgen-model-files Specification

## Purpose
How `lpgen` writes a model in LP format and reads a LINGO model: `WriteLP`, `GenerateLPFile`, the strict `ParseLingo` and `ParseLingoFile`, and the lenient deprecated readers.

## Requirements

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

### Requirement: The LINGO readers refuse what they do not read

`ParseLingo` 與 `ParseLingoFile` SHALL 對無法辨識的敘述、無法讀出變數的宣告、非線性的目標函數或限制式，以及未以 `;` 結束的最後一句，回傳 nil 模型與含行號和敘述的錯誤。

#### Scenario: A last statement without its semicolon
- **WHEN** `ParseLingo("MODEL:\nMIN= X1 + X2\nEND")`
- **THEN** 回傳 nil 模型與錯誤，錯誤含行號 2 與 `MIN= X1 + X2`

#### Scenario: A statement nothing reads
- **WHEN** `ParseLingo` 讀到 `hello world;`
- **THEN** 回傳 nil 模型與錯誤，錯誤含該敘述與它所在的行號

#### Scenario: A relation or objective that is not linear
- **WHEN** `ParseLingo` 讀到 `@UNKNOWN(X)=1;` 或 `MIN= ;`
- **THEN** 回傳 nil 模型與錯誤，錯誤指出第 2 行

#### Scenario: A comment is skipped
- **WHEN** 模型含 `! the plant has two lines;`
- **THEN** 這一句被略過，其餘照常讀取，沒有錯誤

#### Scenario: A statement after a comment on the same line
- **WHEN** 模型含 `! note; X >= 5;`
- **THEN** 註解只到第一個 `;` 為止，`X >= 5` 照常讀入 `Bounds`

#### Scenario: The deprecated readers still drop it
- **WHEN** `ParseLingoModel_str("MODEL:\nMIN= X1 + X2\nEND")`
- **THEN** 回傳目標函數為空的模型，與過去相同，沒有錯誤

### Requirement: The LINGO readers read the variable declarations LINGO has

除 `@BIN` 與 `@INT` 外，`ParseLingo` 與 `ParseLingoFile` SHALL 讀取 `@GIN(x)`、`@FREE(x)` 與 `@BND(l, x, u)` 宣告。

#### Scenario: The three declarations
- **WHEN** 模型含 `@GIN(Y);`、`@FREE(X);` 與 `@BND(-5, Z, 2.5);`
- **THEN** `IntegerVars` 含 `Y`，`Bounds` 含 `X free` 與 `-5 <= Z <= 2.5`

#### Scenario: A free variable reaches the solver
- **WHEN** 以 `lp.Solve` 求解 `ParseLingo` 讀出的 `MIN= X; @FREE(X); X + Y >= -3; Y <= 0;`
- **THEN** 目標值為 -3，而不是把 X 當成非負時的 0
