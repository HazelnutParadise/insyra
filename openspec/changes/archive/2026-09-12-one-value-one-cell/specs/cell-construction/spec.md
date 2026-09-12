## ADDED Requirements

### Requirement: A value can be marked to become exactly one cell

建構清單時，被標記的值 SHALL 成為單一格子，不論它是不是切片。同一次呼叫中未被標記的值 SHALL 維持原本的行為，切片仍然被攤平。格子中 SHALL 存放原本的值本身，SHALL NOT 存放標記。

#### Scenario: A marked slice among ordinary values
- **WHEN** 以 `NewDataList(Cell([]int{1,2}), 3, "a")` 建構
- **THEN** 得到 3 個格子
- **AND** 第一格是 `[]int{1,2}` 本身，型別未被包裝

#### Scenario: An unmarked slice
- **WHEN** 以 `NewDataList([]int{1,2})` 建構
- **THEN** 仍然攤平成 2 個格子

### Requirement: The mark is accepted everywhere a value is accepted

任何接受呼叫端傳入值的入口 SHALL 接受被標記的值並取出其內容。標記 SHALL NOT 被存進格子。搜尋與比對的入口 SHALL 以相同方式處理，使標記與未標記的寫法得到相同結果。

#### Scenario: Appending a marked value
- **WHEN** 以 `Append(Cell(x))` 加入
- **THEN** 結果與 `Append(x)` 相同

#### Scenario: Searching with a marked value
- **WHEN** 以 `Count(Cell(x))` 搜尋
- **THEN** 結果與 `Count(x)` 相同

#### Scenario: Updating and replacing
- **WHEN** 以被標記的值呼叫 `Update`、`InsertAt`、`ReplaceAll` 或 `UpdateElement`
- **THEN** 格子中存放的是原值，不是標記
