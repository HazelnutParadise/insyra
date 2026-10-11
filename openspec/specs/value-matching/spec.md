# value-matching Specification

## Purpose
Defines how the library decides that a cell holds the value being looked for: integers match by value across Go integer types, a float never matches an integer, a value Go cannot compare matches by type and content, every other value matches as it did before, and whole-list comparison stays type-strict.

## Requirements

### Requirement: Integers match by value across Go integer types

When the library looks for a value in a DataList or DataTable, whether counting, finding, replacing or dropping by value, a cell holding an integer SHALL match a searched integer of equal value regardless of their Go integer types. A negative signed integer SHALL NOT match any unsigned integer. A floating-point value SHALL NOT match an integer.

#### Scenario: Counting a CSV column with a Go literal
- **WHEN** 對從 CSV 讀入的整數欄（int64）呼叫 `Count(2)`
- **THEN** 回傳欄中 2 的個數

#### Scenario: A float is not an integer
- **WHEN** 對存有 `2.0` 的欄呼叫 `Count(2)`
- **THEN** 回傳 0

#### Scenario: Replacing from the CLI
- **WHEN** 在 CLI 對整數欄是 int64 的表執行 `replace t 2 0`
- **THEN** 原本的 2 都變成 0

### Requirement: Other values match as they did before

Floats, strings, bools, nil and NaN SHALL match exactly as they did before integers matched by value. A float64 NaN SHALL match a float64 NaN in every lookup except `FindRowsIfContainsAll`, which SHALL keep treating NaN as unequal. A cell Go cannot compare with `==` SHALL match a searched value of the same type and content, and SHALL NOT cause a panic.

#### Scenario: Searching for NaN
- **WHEN** 對含 NaN 的 DataList 呼叫 `Count(math.NaN())`
- **THEN** 回傳 NaN 的個數

#### Scenario: An uncomparable cell
- **WHEN** 對含有無法以 `==` 比較之值的 DataList 呼叫 `FindAll` 搜尋型別與內容相同的值
- **THEN** 不 panic，且該格算相符

### Requirement: Whole-list comparison stays type-strict

`IsEqualTo` and `IsTheSameAs` SHALL keep treating cells of different Go types as different.

#### Scenario: int versus int64 lists
- **WHEN** 呼叫 `NewDataList(1).IsEqualTo(NewDataList(int64(1)))`
- **THEN** 回傳 false

### Requirement: Group keys compare integers by value

`GroupBy`, `Pivot`, `Merge` keys and `OpNUnique` SHALL put integers of equal value in one group whatever their Go width: `int`, `int8` through `int64`, `uint` through `uint64` and `uintptr`. `float32` and `float64` values SHALL group by value among themselves. An integer SHALL NOT share a group with a float or a string of the same value.

#### Scenario: Eleven widths and two floats

- **WHEN** a column holds the value 1 in each of the eleven integer widths, as `float64` and as `float32`, and as the string `"1"`, and is grouped
- **THEN** there are three groups, of sizes 11, 2 and 1

#### Scenario: A distinct count

- **WHEN** `OpNUnique` runs over `1`, `int64(1)`, `uintptr(1)`, `1.0` and `"1"`
- **THEN** it returns 3
