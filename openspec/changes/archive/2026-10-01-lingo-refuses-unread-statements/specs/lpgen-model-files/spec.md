## ADDED Requirements

### Requirement: The LINGO readers refuse what they do not read

`ParseLingo` and `ParseLingoFile` SHALL return a nil model and an error for a statement they do not recognise, for a declaration whose variable they cannot read, and for a last statement left without its closing `;`. The error SHALL name the line the statement starts on and the statement. A statement starting with `!` is a LINGO comment and SHALL be skipped. `ParseLingoModel_str` and `ParseLingoModel_txt` SHALL keep dropping such statements, as they did before.

#### Scenario: A last statement without its semicolon
- **WHEN** `ParseLingo("MODEL:\nMIN= X1 + X2\nEND")`
- **THEN** 回傳 nil 模型與錯誤，錯誤含行號 2 與 `MIN= X1 + X2`

#### Scenario: A statement nothing reads
- **WHEN** `ParseLingo` 讀到 `hello world;`
- **THEN** 回傳錯誤，錯誤含該敘述與它所在的行號

#### Scenario: A comment
- **WHEN** 模型含 `! the plant has two lines;`
- **THEN** 這一句被略過，其餘照常讀取，沒有錯誤

#### Scenario: The deprecated reader
- **WHEN** `ParseLingoModel_str("MODEL:\nMIN= X1 + X2\nEND")`
- **THEN** 回傳目標函數為空的模型，與過去相同

### Requirement: The LINGO readers read the variable declarations LINGO has

Besides `@BIN` and `@INT`, `ParseLingo` and `ParseLingoFile` SHALL read `@GIN(x)` as a general integer variable, `@FREE(x)` as the bound `x free`, and `@BND(l, x, u)`, with `l` and `u` numbers, as the bound `l <= x <= u`.

#### Scenario: The three declarations
- **WHEN** 模型含 `@GIN(Y);`、`@FREE(X);` 與 `@BND(-5, Z, 2.5);`
- **THEN** `IntegerVars` 含 `Y`，`Bounds` 含 `X free` 與 `-5 <= Z <= 2.5`

#### Scenario: A free variable reaches the solver
- **WHEN** 以 `lp.Solve` 求解 `ParseLingo` 讀出的 `MIN= X; @FREE(X); X + Y >= -3; Y <= 0;`
- **THEN** 目標值為 -3，而不是把 X 當成非負時的 0
