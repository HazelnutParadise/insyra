## ADDED Requirements

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
