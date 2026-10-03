## ADDED Requirements

### Requirement: Integers stay integers

A number literal written as digits alone, with an optional leading minus, that fits an `int64` SHALL be an `int64`; any other number literal SHALL be a `float64`. When both operands are integers — a cell of any Go integer type whose value fits an `int64`, or an integer literal — `+`, `-`, `*`, `%` and unary minus SHALL compute in `int64` and give an `int64`, and `==`, `!=`, `<`, `>`, `<=` and `>=` SHALL compare them exactly. In arithmetic a `nil` next to an integer SHALL be an integer `0`. An integer result outside the `int64` range SHALL be an error that says it overflows. `SUM`, `MIN` and `MAX`, and their streaming forms, SHALL give an `int64` when every value they use is an integer, and an integer `SUM` outside the `int64` range SHALL be an error. `MOD` of two integers SHALL give an `int64`. The row index `#` SHALL be an `int64`. Every other numeric operation SHALL compute in `float64` as before, `/` and `^` included.

#### Scenario: An ID past 2^53 keeps its digits
- **WHEN** 欄 `A` 存 `int64(9007199254740993)`，求值 `A + 0`、`A * 1`、`A - 1`
- **THEN** 分別得到 `int64(9007199254740993)`、`int64(9007199254740993)`、`int64(9007199254740992)`

#### Scenario: Integer literals
- **WHEN** 求值 `1 + 2`、`7 % 3`、`-5`、`1.0`、`1e3`
- **THEN** 分別得到 `int64(3)`、`int64(1)`、`int64(-5)`、`float64(1)`、`float64(1000)`

#### Scenario: Division and powers stay float64
- **WHEN** 求值 `7 / 2`、`6 / 3`、`2 ^ 3`
- **THEN** 分別得到 `float64` 的 3.5、2、8

#### Scenario: Exact comparison
- **WHEN** 欄 `A` 存 `int64(1<<53 + 1)`、欄 `B` 存 `int64(1<<53)`，求值 `A == B` 與 `A > B`
- **THEN** 分別得到 false 與 true

#### Scenario: Overflow is an error
- **WHEN** 欄 `A` 存 `int64(math.MaxInt64)`，求值 `A + 1`、`A * 2`、`SUM(A, 1)`
- **THEN** 都回傳提到 overflow 的錯誤

#### Scenario: Aggregates of integers
- **WHEN** 欄 `A` 依序存 `int64(1<<53 + 1)`、`int64(1<<53)`、nil、`int64(2)`，求值 `SUM(A)`、`MAX(A)`、`MIN(A)`
- **THEN** 分別得到 `int64(1<<54 + 3)`、`int64(1<<53 + 1)`、`int64(2)`；streaming 形式得到相同的值與型別

#### Scenario: A decimal makes it float64
- **WHEN** 欄 `A` 依序存 `int64(1)`、`2.5`、`int64(3)`，求值 `SUM(A)`
- **THEN** 得到 `float64(6.5)`

#### Scenario: nil next to an integer
- **WHEN** 欄 `C` 存 `int16(3)`，求值 `nil + C` 與 `nil + 2.5`
- **THEN** 分別得到 `int64(3)` 與 `float64(2.5)`

## MODIFIED Requirements

### Requirement: Every Go numeric type is a number

CCL SHALL read a cell of every Go integer type (`int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`) and of `float32` as a number, in arithmetic, comparisons, conditions, function arguments, aggregates and row indices, giving the result it gives for the same value held as an `int64`. A `uint64` above the `int64` range SHALL be read as the nearest `float64`. A `float32` NaN SHALL be missing to `ISNA` and `IFNA`.

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
- **THEN** 得到 `int64(1<<53 + 1)`，與 `int64(1<<53 + 1)` 的結果相同

#### Scenario: A float32 NaN is missing
- **WHEN** 欄 `A` 存 `float32(NaN)`，求值 `ISNA(A)` 與 `IFNA(A, 0)`
- **THEN** 分別得到 true 與 0
