## MODIFIED Requirements

### Requirement: A LINGO model is read by ParseLingo and ParseLingoFile

`ParseLingo(model string) (*LPModel, error)` SHALL read a LINGO model from text and `ParseLingoFile(path string) (*LPModel, error)` SHALL read one from a file, and the two SHALL return equal models for the same text. Everything the old parser read SHALL be read the same way: the objective, constraints, bounds, `@BIN` and `@INT` declarations.

#### Scenario: The same model from text and from a file
- **WHEN** 同一段 LINGO 模型分別以 `ParseLingo` 讀字串、以 `ParseLingoFile` 讀檔
- **THEN** 兩者都回傳 nil 錯誤，模型相同

## REMOVED Requirements

### Requirement: The old LINGO names keep their meaning for one release
**Reason**: `ParseLingoModel_str` and `ParseLingoModel_txt` were removed in 0.4, the release after the one that deprecated them.
**Migration**: Use `ParseLingo` and `ParseLingoFile`, which return an error where the old functions returned `nil` with a warning.

### Requirement: The LINGO readers refuse what they do not read
**Reason**: It also described the removed `ParseLingoModel_str` and `ParseLingoModel_txt`, and its scenarios cannot be edited in place; restated below as "The LINGO readers refuse statements they cannot read".
**Migration**: See the requirement "The LINGO readers refuse statements they cannot read".

## ADDED Requirements

### Requirement: The LINGO readers refuse statements they cannot read

`ParseLingo` and `ParseLingoFile` SHALL return a nil model and an error for a statement they do not recognise, for a declaration whose variable they cannot read, and for a last statement left without its closing `;`. The error SHALL name the line the statement starts on and the statement. A statement starting with `!` is a LINGO comment and SHALL be skipped.

#### Scenario: A last statement without its semicolon
- **WHEN** `ParseLingo("MODEL:\nMIN= X1 + X2\nEND")`
- **THEN** 回傳 nil 模型與錯誤，錯誤含行號 2 與 `MIN= X1 + X2`

#### Scenario: A statement nothing reads
- **WHEN** `ParseLingo` 讀到 `hello world;`
- **THEN** 回傳錯誤，錯誤含該敘述與它所在的行號

#### Scenario: A comment
- **WHEN** 模型含 `! the plant has two lines;`
- **THEN** 這一句被略過，其餘照常讀取，沒有錯誤
