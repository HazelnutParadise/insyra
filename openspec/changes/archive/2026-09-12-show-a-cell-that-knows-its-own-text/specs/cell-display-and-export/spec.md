## ADDED Requirements

### Requirement: A cell that knows its own text is shown as that text

顯示一個沒有專屬處理的值時，系統 SHALL 先詢問它能不能自己轉成文字，能的話就用那段文字。系統 SHALL NOT 對這種值只印出型別名稱。已有專屬處理的型別（浮點數、整數、布林、字串、位元組、時間）SHALL 不受影響。

#### Scenario: A struct that can print itself
- **WHEN** 顯示一個實作了 `fmt.Stringer` 的 struct 值
- **THEN** 顯示的是它的 `String()` 結果
- **AND** 不是 `<型別名>`

#### Scenario: A struct that cannot
- **WHEN** 顯示一個沒有 `String()` 的 struct 值
- **THEN** 仍然顯示 `<型別名>`

#### Scenario: A nil pointer
- **WHEN** 顯示一個 nil 指標，而它的元素型別有值接收者的 `String()`（例如 nil 的 `*time.Time`）
- **THEN** 顯示 `<nil>`，不呼叫 `String()`、不 panic
