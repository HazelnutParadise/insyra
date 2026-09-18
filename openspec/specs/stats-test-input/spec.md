# stats-test-input Specification

## Purpose
`stats` 參數檢定與 `CalculateMoment` 的輸入契約：無法讀成有限數字的格子回傳指出序列與列號的錯誤，不計入 n；nil 的 list 回傳錯誤而不是 panic。

## Requirements
### Requirement: Parametric tests refuse unreadable cells

`SingleSampleTTest`、`TwoSampleTTest`、`SingleSampleZTest`、`TwoSampleZTest`、`FTestForVarianceEquality`、`BartlettTest`、`LeveneTest` 與 `CalculateMoment` SHALL 經 `asDataList` 轉換後，以 `numericValues` 檢核每一格再使用；任一格非數值、nil、NaN 或 Inf 時 SHALL 回傳含標籤與列號的錯誤，SHALL NOT 把該格算進 n。nil 或帶型別的 nil list SHALL 回傳錯誤，SHALL NOT panic。全為有限數值的輸入 SHALL 得到與變更前逐位元相同的統計量。

#### Scenario: Blank cell is refused
- **WHEN** `SingleSampleTTest(NewDataList(1.0, 2.0, nil, 3.0), 0)`
- **THEN** 回傳錯誤，訊息含 `row 3`

#### Scenario: Clean input unchanged
- **WHEN** 既有測試的全數值輸入
- **THEN** 既有測試不修改即通過

#### Scenario: A nil list
- **WHEN** `SingleSampleTTest(nil, 0)` 或 `CalculateMoment((*insyra.DataList)(nil), 3, true)`
- **THEN** 回傳錯誤，不 panic

### Requirement: Two-sample tests read both samples at one moment

`TwoSampleTTest`、`TwoSampleZTest`、`FTestForVarianceEquality` 與 `PairedTTest` SHALL 在單一 `insyra.AtomicDoAll` 內同時取得兩個樣本的快照，再對快照做數值檢核；SHALL NOT 分兩次各自鎖定一個 list。

#### Scenario: A writer resizes both samples together
- **WHEN** 另一個 goroutine 在 `insyra.AtomicDoAll` 內把兩個 list 一起在 10 與 20 個觀察值之間切換，同時呼叫 `TwoSampleTTest(dl1, dl2, true)`
- **THEN** 自由度只會是 18 或 38，不會出現混到兩個時點的 28
