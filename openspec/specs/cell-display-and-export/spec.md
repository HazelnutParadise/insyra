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

#### Scenario: A nil pointer
- **WHEN** 顯示一個 nil 指標，而它的元素型別有值接收者的 `String()`（例如 nil 的 `*time.Time`）
- **THEN** 顯示 `<nil>`，不呼叫 `String()`、不 panic

### Requirement: A string that is not text is shown as bytes

顯示字串時，若它不是合法的 UTF-8，系統 SHALL 以十六進位顯示其位元組，與 `[]byte` 的顯示方式相同，超過 20 個位元組 SHALL 截斷並附上總長度。系統 SHALL NOT 把這種值當成文字加引號輸出。合法的 UTF-8 字串 SHALL 不受影響。

#### Scenario: Raw bytes in a string cell
- **WHEN** 顯示一個含 `0x00` 與 `0xff` 的字串
- **THEN** 顯示的是十六進位，例如 `00ff41`
- **AND** 結果只含 ASCII，所以欄寬可以精確計算，表格對得齊

#### Scenario: Text is still text
- **WHEN** 顯示一個合法的 UTF-8 字串，含中日韓文字或 emoji
- **THEN** 照原本加引號的方式顯示

#### Scenario: A byte sequence containing a newline byte
- **WHEN** 一段非法 UTF-8 的位元組裡剛好有 `0x0a`
- **THEN** 仍以十六進位顯示，不會被當成多行字串只顯示第一行
