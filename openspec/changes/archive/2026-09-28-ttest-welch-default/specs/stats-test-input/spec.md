## MODIFIED Requirements

### Requirement: Two-sample tests read both samples at one moment

`TwoSampleTTest`、`TwoSampleZTest`、`FTestForVarianceEquality` 與 `PairedTTest` SHALL 在單一 `insyra.AtomicDoAll` 內同時取得兩個樣本的快照，再對快照做數值檢核；SHALL NOT 分兩次各自鎖定一個 list。

#### Scenario: A writer resizes both samples together
- **WHEN** 另一個 goroutine 在 `insyra.AtomicDoAll` 內把兩個 list 一起在 10 與 20 個觀察值之間切換，同時呼叫 `TwoSampleTTest(dl1, dl2, TTestOptions{EqualVariance: true})`
- **THEN** 自由度只會是 18 或 38，不會出現混到兩個時點的 28
