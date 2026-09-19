# datalist-equality-and-cleanup Specification

## Purpose
`DataList` 相等比較與清理方法的語意：無法用 `==` 比較的格子以型別與內容比較、不會讓相等比較 panic、清理方法單趟完成且結果不變、`Err()` 記錄正確的方法名。

## Requirements
### Requirement: Equality never panics

`DataList.IsEqualTo` 與 `IsTheSameAs` 遇到 Go 無法用 `==` 比較的格子 SHALL 以型別與內容比較，SHALL NOT panic：內容相同的這種格子相等，內容不同則不相等；兩個純量 float64 NaN SHALL 維持不相等。

#### Scenario: Uncomparable cell
- **WHEN** `dl := NewDataList(1.0, uncomparableCell{s: []int{1}}); dl.IsEqualTo(dl.Clone())`
- **THEN** 回傳 true，不 panic

#### Scenario: Uncomparable cell with different content
- **WHEN** `NewDataList(1.0, uncomparableCell{s: []int{1}}).IsEqualTo(NewDataList(1.0, uncomparableCell{s: []int{2}}))`
- **THEN** 回傳 false，不 panic

#### Scenario: NaN stays unequal
- **WHEN** `dl := NewDataList(1.0, math.NaN()); dl.IsEqualTo(dl.Clone())`
- **THEN** 回傳 false

### Requirement: Cleanup is single pass and Update names itself

`ClearNaNs`、`ClearNils`、`ClearNilsAndNaNs`、`ClearNumbers`、`DropAll`、`ClearStrings` SHALL 以單趟過濾完成，結果與原本相同：`ClearNumbers` 只移除內建數值型別，`DropAll` 只在要刪除的值含 NaN 時刪除 NaN。`Update` 越界時 `Err().FuncName` SHALL 為 `Update`。

#### Scenario: ClearNilsAndNaNs result
- **WHEN** `NewDataList(1.0, math.NaN(), nil, 2.0).ClearNilsAndNaNs()`
- **THEN** 資料為 `[1, 2]`

#### Scenario: ClearNumbers keeps named numeric types
- **WHEN** `NewDataList(1, 3.5, "x", time.Second).ClearNumbers()`
- **THEN** 資料為 `["x", time.Second]`

#### Scenario: Update out of range
- **WHEN** `NewDataList(1).Update(5, 2)`
- **THEN** `Err().FuncName == "Update"`
