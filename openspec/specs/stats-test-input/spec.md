# stats-test-input Specification

## Purpose
The hypothesis tests and `CalculateMoment` refuse a cell they cannot read as a finite number, with an error naming the list and the position, both counted from one, and never count it in n; a nil list is an error, not a panic.

## Requirements

### Requirement: Parametric tests refuse unreadable cells

`SingleSampleTTest`、`TwoSampleTTest`、`SingleSampleZTest`、`TwoSampleZTest`、`FTestForVarianceEquality`、`BartlettTest`、`LeveneTest` 與 `CalculateMoment` SHALL 經 `asDataList` 轉換後，以 `numericSlice` 讀取輸入；任一格非數值、nil、NaN 或 Inf 時 SHALL 回傳含標籤與列號的錯誤，SHALL NOT 把該格算進 n。nil 或帶型別的 nil list SHALL 回傳錯誤，SHALL NOT panic。全為有限數值的輸入 SHALL 得到與變更前逐位元相同的統計量。

#### Scenario: Blank cell is refused
- **WHEN** `SingleSampleTTest(NewDataList(1.0, 2.0, nil, 3.0), 0)`
- **THEN** 回傳錯誤，訊息含 `row 3`

#### Scenario: Clean input unchanged
- **WHEN** 既有測試的全數值輸入
- **THEN** 既有測試只改呼叫形式、不改斷言的值即通過

#### Scenario: A nil list
- **WHEN** `SingleSampleTTest(nil, 0)` 或 `CalculateMoment((*insyra.DataList)(nil), 3, true)`
- **THEN** 回傳錯誤，不 panic

### Requirement: Two-sample tests read both samples at one moment

`TwoSampleTTest`、`TwoSampleZTest`、`FTestForVarianceEquality` 與 `PairedTTest` SHALL 在單一 `insyra.AtomicDoAll` 內同時取得兩個樣本的快照，再對快照做數值檢核；SHALL NOT 分兩次各自鎖定一個 list。

#### Scenario: A writer resizes both samples together
- **WHEN** 另一個 goroutine 在 `insyra.AtomicDoAll` 內把兩個 list 一起在 10 與 20 個觀察值之間切換，同時呼叫 `TwoSampleTTest(dl1, dl2, TTestOptions{EqualVariance: true})`
- **THEN** 自由度只會是 18 或 38，不會出現混到兩個時點的 28

### Requirement: Paired, rank and ANOVA tests refuse unreadable cells

`PairedTTest`、`SingleSampleWilcoxon`、`PairedWilcoxon`、`MannWhitneyU`、`OneWayANOVA`、`TwoWayANOVA`、`RepeatedMeasuresANOVA`、`KruskalWallis` 與 `FriedmanTest` SHALL 在計算前檢查每一格；任一格是 nil、文字或其他無法轉成數字的值時，SHALL 回傳 `<名稱> contains a non-numeric value at <位置>: <值>`，任一格是 `NaN`、`+Inf` 或 `-Inf` 時，SHALL 回傳 `<名稱> contains a non-finite value at <位置>: <值>`，SHALL NOT 回傳結果。nil 或帶型別的 nil list SHALL 視為空的 list，由函式原有的空樣本、空組、空格或受試者長度檢查回傳錯誤，SHALL NOT panic，也 SHALL NOT 讓程式結束。全為有限數值的輸入 SHALL 得到與變更前逐位元相同的結果。

#### Scenario: NaN in a rank test is refused
- **WHEN** `KruskalWallis([]insyra.IDataList{NewDataList(1.0, 2.0, math.NaN(), 4.0), NewDataList(1.0, 2.0, 3.0, 4.0)})`
- **THEN** 回傳錯誤 `group 1 contains a non-finite value at row 3: NaN`，結果為 nil

#### Scenario: Infinity in a paired test is refused
- **WHEN** `PairedTTest(NewDataList(1.0, 2.0, math.Inf(1), 4.0), NewDataList(1.5, 2.5, 3.0, 4.5))`
- **THEN** 回傳錯誤 `data1 contains a non-finite value at row 3: +Inf`，結果為 nil

#### Scenario: Blank cell in a two-way ANOVA is refused
- **WHEN** `TwoWayANOVA(2, 2, []insyra.IDataList{a1b1, a1b2, NewDataList(5.0, nil), a2b2})`
- **THEN** 回傳錯誤 `cell (A=2, B=1) contains a non-numeric value at row 2: <nil>`

#### Scenario: A nil group does not end the program
- **WHEN** `OneWayANOVA([]insyra.IDataList{NewDataList(1.0, 2.0), (*insyra.DataList)(nil)})`
- **THEN** 回傳錯誤 `group 2 is empty`，呼叫端的 goroutine 與程式都繼續執行

#### Scenario: A slice cell in a wrapped list stays one cell
- **WHEN** 一個內嵌 `*insyra.DataList`、本身不是 `*insyra.DataList` 的 list 依序存放 `1.0`、`2.0`、`3.0` 與一格 `[]float64{10, 11}`，傳給 `OneWayANOVA([]insyra.IDataList{wrapped, NewDataList(4.0, 5.0, 6.0)})`
- **THEN** 回傳錯誤 `group 1 contains a non-numeric value at row 4: [10 11]`，那一格 SHALL NOT 被拆成兩個觀察值

#### Scenario: Clean input unchanged
- **WHEN** 既有以 R 驗證的測試使用全為有限數值的輸入
- **THEN** 既有測試只改呼叫形式、不改斷言的值即通過

### Requirement: Hypothesis test errors number positions from one

`stats` 的假設檢定，即 `SingleSampleTTest`、`TwoSampleTTest`、`PairedTTest`、`SingleSampleZTest`、`TwoSampleZTest`、`FTestForVarianceEquality`、`LeveneTest`、`BartlettTest`、`SingleSampleWilcoxon`、`PairedWilcoxon`、`MannWhitneyU`、`OneWayANOVA`、`TwoWayANOVA`、`RepeatedMeasuresANOVA`、`KruskalWallis` 與 `FriedmanTest`，錯誤訊息中的每個位置 SHALL 從 1 起算，且 SHALL 寫出它數的是什麼：`row`、`group`、`cell (A=a, B=b)` 的 A 與 B 水準、`subject` 或 `condition`。以 slice 接收多個 list 的函式 SHALL 依 slice 中的順序為組、格或受試者編號，第一個為 1。`TwoWayANOVA` 的 `cells[i*factorBLevels + j]` SHALL 稱為 `A=i+1, B=j+1`。`RepeatedMeasuresANOVA` 與 `FriedmanTest` 的每個 list 是一位受試者，其中的格 SHALL 稱為 `condition`。

#### Scenario: Levene names the second group as group 2
- **WHEN** `LeveneTest([]insyra.IDataList{NewDataList(1.0, 2.0, 3.0), NewDataList(4.0, nil, 6.5)})`
- **THEN** 回傳錯誤 `group 2 contains a non-numeric value at row 2: <nil>`

#### Scenario: Friedman names the subject and the condition
- **WHEN** `FriedmanTest([]insyra.IDataList{NewDataList(1.0, 2.0, 3.0), NewDataList(2.0, math.NaN(), 1.0), NewDataList(3.0, 1.0, 2.0)})`
- **THEN** 回傳錯誤 `subject 2 contains a non-finite value at condition 2: NaN`

#### Scenario: An empty cell is numbered from one
- **WHEN** `TwoWayANOVA(2, 2, []insyra.IDataList{NewDataList(), a1b2, a2b1, a2b2})`
- **THEN** 回傳錯誤 `empty cell at A=1, B=1`

#### Scenario: A mismatched subject is numbered from one
- **WHEN** `FriedmanTest([]insyra.IDataList{NewDataList(1.0, 2.0, 3.0), NewDataList(2.0, 1.0)})`
- **THEN** 回傳錯誤 `subject 2 has 2 observations, expected 3`
