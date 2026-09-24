# datalist-numeric-input Specification

## Purpose
同一個值在「是不是數字」的判斷和「轉成 float64」之間得到一致的答案，以數值型別為底的自訂型別也一樣。

## Requirements

### Requirement: A named numeric type is a number everywhere or nowhere

判斷「這是不是數字」與「把它轉成 float64」的兩條路徑 SHALL 對同一個值給出一致的答案。以數值 kind 為底的具名型別（`type Celsius float64`）SHALL 兩邊都接受。

#### Scenario: A user-defined numeric type
- **WHEN** 對 `Celsius(36.6)` 呼叫 `IsNumeric` 與 `ToFloat64Safe`
- **THEN** 兩者都說是數字，且轉換結果為 36.6
