# cell-display-and-export Specification

## Purpose
Most cells hold a primitive the library knows how to render. Some hold a value it has never seen, and the question is what a reader gets for one. This capability answers it for display: a value that can turn itself into text is shown as that text. It exists because the fallback used to print a type name, which hides the value a cell actually holds.

## Requirements
### Requirement: A cell that knows its own text is shown as that text

顯示一個沒有專屬處理的值時，系統 SHALL 先詢問它能不能自己轉成文字，能的話就用那段文字。系統 SHALL NOT 對這種值只印出型別名稱。已有專屬處理的型別（浮點數、整數、布林、字串、位元組、時間）SHALL 不受影響。

#### Scenario: A struct that can print itself
- **WHEN** 顯示一個實作了 `fmt.Stringer` 的 struct 值
- **THEN** 顯示的是它的 `String()` 結果
- **AND** 不是 `<型別名>`

#### Scenario: A struct that cannot
- **WHEN** 顯示一個沒有 `String()` 的 struct 值
- **THEN** 仍然顯示 `<型別名>`
