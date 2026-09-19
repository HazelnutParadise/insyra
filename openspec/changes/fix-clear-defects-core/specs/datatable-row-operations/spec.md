## ADDED Requirements

### Requirement: Unnamed columns merge by position

垂直合併 SHALL 以名稱對齊有名稱的欄位，以「在無名欄中的位置」對齊沒有名稱的欄位。空字串 SHALL NOT 被視為重複的名稱。

#### Scenario: Two tables built without column names
- **WHEN** 兩張以 `NewDataTable(NewDataList(...), NewDataList(...))` 建立的表垂直合併
- **THEN** 合併成功，兩欄各自依位置接起來，而不是回報「重複欄名 ""」
