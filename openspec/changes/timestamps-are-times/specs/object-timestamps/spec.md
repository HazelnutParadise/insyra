## ADDED Requirements

### Requirement: Creation and modification times keep the clock's resolution

`DataList` and `DataTable` SHALL provide `CreationTime() time.Time` and `LastModifiedTime() time.Time`. Both SHALL keep the clock's full resolution, so two changes less than a second apart give different `LastModifiedTime` values whenever the clock reads differently at each.

#### Scenario: Two changes within one second
- **WHEN** 時鐘在 T 建立 list，在 T+100ns 呼叫 `Append`，在 T+200ns 再呼叫 `Append`
- **THEN** `CreationTime()` 為 T，第一次之後 `LastModifiedTime()` 為 T+100ns，第二次之後為 T+200ns

#### Scenario: A table
- **WHEN** 時鐘在 T 建立 table，在 T+5ns 呼叫 `AppendRowsByColIndex`
- **THEN** `CreationTime()` 為 T，`LastModifiedTime()` 為 T+5ns

### Requirement: The modification time never moves backwards

`LastModifiedTime` SHALL NOT decrease when a change happens while the wall clock reads earlier than the recorded time.

#### Scenario: The clock steps back
- **WHEN** list 在 T+1s 被修改後，時鐘退回 T 再修改一次
- **THEN** `LastModifiedTime()` 仍為 T+1s

### Requirement: The second-resolution getters remain for one release

`GetCreationTimestamp()` and `GetLastModifiedTimestamp()` SHALL be marked Deprecated and SHALL return the Unix seconds of `CreationTime()` and `LastModifiedTime()`.

#### Scenario: Old getters
- **WHEN** list 建立於 `time.Unix(1700000000, 500)`
- **THEN** `GetCreationTimestamp()` 為 1700000000，`CreationTime()` 等於 `time.Unix(1700000000, 500)`

### Requirement: IsTheSameAs compares data and name

`DataList.IsTheSameAs` SHALL return true exactly when both lists hold equal data, under the same cell rule as `IsEqualTo`, and have the same name. It SHALL NOT compare creation or modification times. A nil argument SHALL return false.

#### Scenario: A clone made later
- **WHEN** list 建立於 T，在 T+2s 呼叫 `Clone()`
- **THEN** `IsTheSameAs(clone)` 為 true

#### Scenario: Different names
- **WHEN** 兩個 list 資料相同、名稱不同
- **THEN** `IsEqualTo` 為 true，`IsTheSameAs` 為 false
