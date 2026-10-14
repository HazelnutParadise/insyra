## MODIFIED Requirements

### Requirement: Chainable methods never return nil

`DataList` 上回傳 `*DataList` 的方法 SHALL NOT 回傳 nil。會產生新 list 的方法失敗時 SHALL 回傳可用的空 `DataList`，錯誤 SHALL 同時記錄在該結果與接收者上。直接修改接收者的方法失敗時 SHALL 回傳接收者本身、內容不變，錯誤記錄在接收者上。

#### Scenario: Invalid window keeps the chain alive
- **WHEN** 對 3 個元素的 list 呼叫 `MovingAverage(0).Sort()`
- **THEN** 不 panic，結果長度為 0，接收者的 `Err()` 非 nil

#### Scenario: A failed in-place method keeps the data
- **WHEN** 對 `[1.0, 2.0, "x"]` 呼叫 `dl = dl.Normalize()`
- **THEN** `dl` 仍是原本的 list，內容為 `[1.0, 2.0, "x"]`，`Err()` 非 nil

#### Scenario: A failed sample carries its error
- **WHEN** 呼叫 `Sample(0, false)`、`Sample(9, false)`（3 個元素，不放回）或 `SampleFrac(2, false)`
- **THEN** 回傳長度 0 的新 list，其 `Err()` 非 nil，接收者的 `Err()` 也非 nil

#### Scenario: Two optional values do not return the receiver
- **WHEN** 呼叫 `Rank(true, false)` 或 `Shift(1, 0, 0)`
- **THEN** 回傳的不是接收者，而是長度 0、`Err()` 非 nil 的新 list，接收者內容不變

#### Scenario: Sampling an empty list is not a failure
- **WHEN** 對空 list 呼叫 `Sample(1, false)` 或 `SampleFrac(0.5, false)`
- **THEN** 回傳空 list，結果與接收者的 `Err()` 皆為 nil
