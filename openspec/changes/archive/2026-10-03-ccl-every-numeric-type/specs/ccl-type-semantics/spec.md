## ADDED Requirements

### Requirement: Every Go numeric type is a number

CCL SHALL read a cell of every Go integer type (`int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`) and of `float32` as a number, in arithmetic, comparisons, conditions, function arguments, aggregates and row indices, giving the result it gives for the same value held as an `int64`. A `uint64` above 2^53 SHALL be read as the nearest `float64`. A `float32` NaN SHALL be missing to `ISNA` and `IFNA`.

#### Scenario: Aggregates over narrow integers
- **WHEN** 欄 `A` 依序存 `int16(3)`、`int16(4)`、`uint8(5)`，求值 `SUM(A)`、`MAX(A)`、`AVG(A)`
- **THEN** 分別得到 12、5、4，不回傳錯誤

#### Scenario: Comparing a narrow integer
- **WHEN** 欄 `A` 存 `uint8(1)`，逐列求值 `A == 1`
- **THEN** 得到 true

#### Scenario: Arithmetic and conditions on a narrow integer
- **WHEN** 欄 `A` 存 `int8(3)`，逐列求值 `A * 2` 與 `IF(A, 'y', 'n')`
- **THEN** 分別得到 6 與 `'y'`

#### Scenario: A row index held as an integer column
- **WHEN** 欄 `B` 存 `int64(1)`，逐列求值 `A.B`
- **THEN** 得到 `A` 第 1 列（從 0 數）的值；`B` 存其他整數型別或 `float32(1)` 時結果相同，存 `float32(1.5)` 時回傳錯誤，與 `float64` 的列號規則相同

#### Scenario: A uint64 past the float64 range of exact integers
- **WHEN** 欄 `A` 存 `uint64(1<<53 + 1)`，求值 `A + 0`
- **THEN** 得到 `float64(1<<53)`，與 `int64(1<<53 + 1)` 的結果相同

#### Scenario: A float32 NaN is missing
- **WHEN** 欄 `A` 存 `float32(NaN)`，求值 `ISNA(A)` 與 `IFNA(A, 0)`
- **THEN** 分別得到 true 與 0
