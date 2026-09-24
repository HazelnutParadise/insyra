## ADDED Requirements

### Requirement: NaN equals NaN in IsEqualTo

`DataList.IsEqualTo` SHALL 把兩個 float64 NaN 視為相等（pandas `equals` 語意），因此任何 list SHALL 與自己的 `Clone()` 相等。

#### Scenario: List with NaN equals its clone
- **WHEN** `dl := NewDataList(1.0, math.NaN()); dl.IsEqualTo(dl.Clone())`
- **THEN** 回傳 true

### Requirement: Cleanup is single pass and Update names itself

`ClearNaNs`、`ClearNils`、`ClearNilsAndNaNs` SHALL 以單趟過濾完成（結果與原本相同）；`Update` 越界時 `Err().FuncName` SHALL 為 `Update`。

#### Scenario: ClearNaNs result
- **WHEN** `NewDataList(1.0, math.NaN(), nil, 2.0).ClearNilsAndNaNs()`
- **THEN** 資料為 `[1, 2]`

#### Scenario: Update out of range
- **WHEN** `NewDataList(1).Update(5, 2)`
- **THEN** `Err().FuncName == "Update"`
