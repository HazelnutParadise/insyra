# datalist-conversions Specification

## Purpose
How `DataList` converts values between types in place: `ParseNumbers` types the numbers it reads with the same rule the CSV reader uses for a column, so the same text gets the same Go type either way, and `Capitalize` title-cases with the language-neutral casing rules.

## Requirements

### Requirement: ParseNumbers types the list the way the CSV reader types a column

`DataList.ParseNumbers` SHALL read each string after trimming surrounding white space, with `strconv.ParseInt(s, 10, 64)` for an integer and `strconv.ParseFloat(s, 64)` otherwise, the two calls the CSV reader's column inference uses. A value that is already a Go integer (any integer kind, named types included) and fits `int64` SHALL count as an integer; a larger unsigned integer and any Go float SHALL count as a decimal. When every counted value is an integer and no string is empty (after trimming), every counted value SHALL become `int64`; otherwise every counted value SHALL become `float64` and every empty string SHALL become `NaN`. `nil`, and a value that is a number but not a Go integer or float (such as a decimal), SHALL be left unchanged without an error. A string that is not a number and any other value SHALL be left unchanged, and the call SHALL record one error that gives how many values were left and names the first by its 1-based row. For a list of strings without surrounding white space and with no unreadable value, the result SHALL equal what the CSV reader produces for the same column.

#### Scenario: An integer list keeps every digit
- **WHEN** `NewDataList("9007199254740993", "1").ParseNumbers()`
- **THEN** 結果為 `[int64(9007199254740993), int64(1)]`，`Err()` 為 nil

#### Scenario: One decimal makes every number a float
- **WHEN** `NewDataList("1", "2.5", "3").ParseNumbers()`
- **THEN** 結果為 `[1.0, 2.5, 3.0]`，全為 `float64`

#### Scenario: Numbers already in the list take part
- **WHEN** `NewDataList("1", 2, "3", int8(8)).ParseNumbers()`
- **THEN** 結果為四個 `int64`：`[1, 2, 3, 8]`

#### Scenario: An empty string is a missing number
- **WHEN** `NewDataList("1", "", "3").ParseNumbers()`
- **THEN** 結果為 `[1.0, NaN, 3.0]`，全為 `float64`，`Err()` 為 nil

#### Scenario: Unreadable values stay and are reported once
- **WHEN** `NewDataList("1", "hello", "3", true, nil).ParseNumbers()`
- **THEN** 結果為 `[int64(1), "hello", int64(3), true, nil]`
- **AND** `Err()` 記錄一筆錯誤，說明有 2 個值未轉換，並指出第一個是第 2 列的 `"hello"`

#### Scenario: The same text gets the same types as the CSV reader
- **WHEN** 一份 CSV 以 `RawStrings: true` 讀入後對每欄呼叫 `ParseNumbers`，另一次以預設選項讀入同一份 CSV
- **THEN** 對每一欄沒有無法解析的值的欄位，兩者每一格的值與 Go 型別都相同（NaN 視為相同）

### Requirement: Capitalize follows the root casing rules

`DataList.Capitalize` SHALL title-case each string with the language-neutral rules (`language.Und`), after lowering it, and SHALL leave non-string values unchanged.

#### Scenario: No language tailoring
- **WHEN** `NewDataList("ijssel", "istanbul", "hello world", 3).Capitalize()`
- **THEN** 結果為 `["Ijssel", "Istanbul", "Hello World", 3]`
