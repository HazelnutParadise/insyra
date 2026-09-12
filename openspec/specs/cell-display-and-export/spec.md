# cell-display-and-export Specification

## Purpose
Most cells hold a primitive the library knows how to render. Some hold a value it has never seen, and the question is what a reader and an export file get for one. This capability answers it: a value that can turn itself into text is shown and written as that text, and a value that can serialise itself is left to do so. It exists because the fallback used to print a type name and export an empty object, which reads as if the cell were empty rather than unfamiliar.
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

### Requirement: Exporting to JSON does not drop a value it could write

匯出 JSON 時，若某個值自己知道怎麼轉成文字，卻不知道怎麼序列化，系統 SHALL 寫出它的文字。系統 SHALL NOT 因為欄位未匯出就寫出空物件。已經實作 `json.Marshaler` 或 `encoding.TextMarshaler` 的型別 SHALL 維持原本的序列化方式。

#### Scenario: A value with only String()
- **WHEN** 匯出含這種值的表格
- **THEN** 該欄位是它的文字，不是 `{}`

#### Scenario: A value that marshals itself
- **WHEN** 匯出含 `time.Time` 的表格
- **THEN** 維持 RFC 3339 形式，不會換成 Go 的預設時間文字

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

