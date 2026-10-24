## ADDED Requirements

### Requirement: Column index conversions name both ends

The root package SHALL export `ColIndexToNumber(colIndex string) (int, bool)`, which turns an Excel-style column index (letters, either case) into its 0-based number, and `ColNumberToIndex(colNumber int) (string, bool)`, which does the reverse. Each SHALL report `false` for input that is not a column. Their doc comments SHALL state that the number counts from 0.

#### Scenario: Converting both ways
- **WHEN** 呼叫 `ColIndexToNumber("A")`、`ColIndexToNumber("AA")`、`ColIndexToNumber("zz")`
- **THEN** 依序回傳 `0, true`、`26, true`、`701, true`，且 `ColNumberToIndex` 對 `0`、`26`、`701` 依序回傳 `"A"`、`"AA"`、`"ZZ"`

#### Scenario: A round trip
- **WHEN** 對 0 到 20000 的每個數字先呼叫 `ColNumberToIndex` 再呼叫 `ColIndexToNumber`
- **THEN** 回到原本的數字

#### Scenario: Input that is not a column
- **WHEN** 呼叫 `ColIndexToNumber("")`、`ColIndexToNumber("A1")`，或 `ColNumberToIndex(-1)`
- **THEN** 前兩個回傳 `-1, false`，最後一個回傳 `"", false`

### Requirement: The coloured-output getter mirrors its setter

`Config.GetUseColoredOutput()` SHALL report the value last given to `Config.SetUseColoredOutput`, and `SetDefaultConfig` SHALL make it `true`.

#### Scenario: Turning coloured output off
- **WHEN** 呼叫 `Config.SetUseColoredOutput(false)`
- **THEN** `Config.GetUseColoredOutput()` 回傳 `false`；呼叫 `SetDefaultConfig()` 後回傳 `true`

### Requirement: The replaced names keep their meaning for one release

`ParseColIndex`, `CalcColIndex` and `Config.GetDoesUseColoredOutput` SHALL remain as wrappers that return exactly what `ColIndexToNumber`, `ColNumberToIndex` and `Config.GetUseColoredOutput` return. Each SHALL have a doc comment whose `Deprecated:` paragraph names its replacement and says it is removed in 0.4.

#### Scenario: The deprecation notices
- **WHEN** 讀取三個舊名稱的 doc comment
- **THEN** 各自有指名新名稱的 `Deprecated:` 段落與移除時程

#### Scenario: An old name gives the same answer
- **WHEN** 對 0 到 20000 的每個數字與其欄位索引比較新舊函式，並在開關彩色輸出後比較兩個 getter
- **THEN** 新舊名稱的回傳值完全相同

### Requirement: The thread-safety switch keeps its underscore on purpose

`Config.Dangerously_TurnOffThreadSafety` SHALL keep its name, and its doc comment SHALL say that the underscore is a deliberate warning that makes the call site look wrong.

#### Scenario: The thread-safety switch says the underscore is deliberate
- **WHEN** 讀取 `Dangerously_TurnOffThreadSafety` 的 doc comment
- **THEN** 註解說明底線是刻意的警告

### Requirement: isr keeps Of, From and Push

`isr`'s `DL.Of`/`DT.Of` and `DL.From`/`DT.From` SHALL both remain and build the same value. `isr`'s `Push` methods SHALL remain: `dl.Push` returns the `isr` list so a chain can continue, and `dt.Push` appends a row or a column depending on the type it is given.

#### Scenario: Of and From build the same list and table
- **WHEN** 以相同參數呼叫 `isr.DL.Of` 與 `isr.DL.From`，以及 `isr.DT.Of` 與 `isr.DT.From`
- **THEN** 兩者的型別與內容相同

#### Scenario: Push keeps the isr chain and takes rows
- **WHEN** 呼叫 `isr.DL.From(1, 2, 3).Push(4).At(3)`，以及對一列一欄的 `isr.DT` 呼叫 `Push(isr.Row{...})`
- **THEN** 前者回傳 `4`，後者使表格多一列且欄數不變
