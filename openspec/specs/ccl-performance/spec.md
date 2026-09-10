# ccl-performance Specification

## Purpose
A CCL expression does not repeat work whose answer cannot change, and no optimisation changes a result. The two go together: the value a folded aggregate yields is the one the row loop computed each time, and a rolling window still receives the same floats in the same order, so its sums are bit-identical.

## Requirements

### Requirement: A row-invariant aggregate is evaluated once

An aggregate function call that does not reference the current row (`#`) SHALL be evaluated once per expression, not once per row. An aggregate that does reference `#` SHALL remain row-dependent.

#### Scenario: A z-score over a large column
- **WHEN** 對 20,000 列求值 `(A - AVG(A)) / STDEV(A)`
- **THEN** 結果與逐列重算相同，且耗時與 `A / 1` 同一個數量級，而不是數千倍

#### Scenario: An aggregate that depends on the row
- **WHEN** 求值一個引數含有 `#` 的聚合
- **THEN** 它仍然逐列求值，各列結果不同

### Requirement: Optimisations do not change results

Every value produced by an optimised path SHALL equal the value the unoptimised path produced, including the order in which floating-point operations are applied.

#### Scenario: Rolling windows keep their summation order
- **WHEN** 對同一欄計算 `ROLLING_SUM`、`ROLLING_MEAN` 與 `ROLLING_STD`
- **THEN** 每個視窗收到的 `[]float64` 內容與順序不變，結果與逐視窗重新轉換時完全相同

#### Scenario: A string that is not a date
- **WHEN** 對文字欄求值 `B & 'x'`
- **THEN** 結果不變，且不再對每個值嘗試日期解析

### Requirement: A compiled pattern is reused

`REGEX_MATCH` SHALL compile a given pattern once and reuse it. The cache SHALL be bounded so that patterns built per row cannot grow it without limit, and an invalid pattern SHALL still be reported.

#### Scenario: The same pattern over many rows
- **WHEN** 對 100,000 列求值 `REGEX_MATCH(B, 'item-[0-9]+')`
- **THEN** 該樣式只編譯一次，結果與每列重新編譯相同
