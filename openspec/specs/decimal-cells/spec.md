# decimal-cells Specification

## Purpose
A fixed-point decimal is what money looks like in a cell, and the library puts one there from two directions: an amortization schedule and a Parquet decimal column. This capability says it is a number — readable by every numeric path, sorted among the other numbers, exact when compared with another decimal and rounded only where a float was asked for — and that being one is decided by the shape of the value rather than by any particular decimal package. It exists because a column of money used to answer NaN, and say nothing.

## Requirements
### Requirement: A decimal in a cell is a number

固定小數點的十進位值 SHALL 被視為數值。數值讀取路徑 SHALL 讀得到它，`IsNumeric` SHALL 回報 true，兩者 SHALL NOT 不一致。系統 SHALL NOT 對十進位欄位安靜地回答 `NaN`。

#### Scenario: Averaging a column of money
- **WHEN** 對一欄十進位金額呼叫 `Mean`
- **THEN** 得到該欄的平均值，不是 `NaN`

#### Scenario: The two halves agree
- **WHEN** 對同一個十進位值呼叫 `IsNumeric` 與數值讀取
- **THEN** 兩者得到一致的答案

#### Scenario: A value wider than a float64
- **WHEN** 十進位值的有效位數超過 `float64` 能表示的範圍
- **THEN** 轉換後的浮點數為四捨五入的結果，而格子中的值仍然精確

### Requirement: A decimal sorts among numbers

十進位值 SHALL 與其他數值一同依大小排序，SHALL NOT 被分到數值之外的群組。兩個十進位值之間的比較 SHALL 保持精確，不經由浮點數。

#### Scenario: A column mixing decimals and floats
- **WHEN** 排序一欄同時含十進位與浮點數的資料
- **THEN** 結果依數值大小交錯，而不是分成兩段

#### Scenario: Two decimals
- **WHEN** 比較兩個十進位值
- **THEN** 以十進位本身的比較決定，超出 `float64` 精度的差異也分辨得出

### Requirement: Being a decimal is decided by shape

系統 SHALL 以「能報出自己的文字與小數位數，且該文字可解析為數字」判斷一個值是不是十進位，SHALL NOT 依賴任何特定十進位套件的具體型別。

#### Scenario: Another library's decimal
- **WHEN** 格子中的值來自另一個同樣形狀的十進位套件
- **THEN** 同樣被視為數值

#### Scenario: A type that only looks similar
- **WHEN** 某型別剛好有同樣的方法，但其文字不是數字
- **THEN** 不被視為數值

