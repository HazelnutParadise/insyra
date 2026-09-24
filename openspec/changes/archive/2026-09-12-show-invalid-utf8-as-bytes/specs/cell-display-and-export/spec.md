## ADDED Requirements

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
