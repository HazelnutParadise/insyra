# cell-identity Specification

## Purpose
Most cell values are things Go can compare and hash, so counting, searching and grouping them is free. Some are not: a slice, a map, anything holding one. This capability says what happens to those. They are identified by their type and their content, so they can be counted, matched and grouped like anything else, and the library never crashes on them. It exists because the two halves used to disagree and one of them was fatal: counting a binary column killed the process while searching it reported that a value sitting in two rows was not there.
## Requirements
### Requirement: A cell Go cannot compare never crashes the library

對 Go 無法比較的格子值進行計數、搜尋、比對或分組時，系統 SHALL NOT panic，SHALL NOT 讓程式終止。系統 SHALL 以該值的型別與內容識別它。

#### Scenario: Counting a column of binary values
- **WHEN** 對含二進位值（例如從 SQL BLOB 欄讀入的 `[]byte`）的欄位呼叫 `Counter`
- **THEN** 回傳計數結果，不 panic
- **AND** 內容相同的兩列被算成同一項，計數為 2

#### Scenario: A value whose type looks comparable but is not
- **WHEN** 格子值是元素為 `any` 的陣列，或欄位為 `any` 的 struct，而其中裝著切片
- **THEN** 一樣不 panic，並以型別與內容識別

#### Scenario: A self-referential value
- **WHEN** 格子值是一個包含自己的切片
- **THEN** 編碼在固定深度停止，不會耗盡堆疊

### Requirement: Counting and searching give the same answer

同一個無法比較的值，`Counter` 認為它出現幾次，`Count` SHALL 回報同樣的次數。`FindAll`、`Replace`、`DropAll`、`IsEqualTo` SHALL 依同一套識別規則判斷相等。系統 SHALL NOT 讓一個方法找得到而另一個找不到。

#### Scenario: The same binary value through both methods
- **WHEN** 欄位有兩列持有相同內容的 `[]byte`，以該值呼叫 `Count`
- **THEN** 回傳 2，與 `Counter` 對該項的計數一致

#### Scenario: Different content
- **WHEN** 以內容不同的值搜尋
- **THEN** 找不到，不會誤判為相等

### Requirement: A stand-in key can be looked up and printed

不可雜湊的值在計數結果中 SHALL 以一個可雜湊的替身作為 key。不等於自己的值（NaN，以及任何含有 NaN 的陣列或結構）SHALL 比照辦理，因為用它當 key 會產生永遠取不回來的項目。系統 SHALL 提供由原值取得該替身的方法，讓呼叫端查得到計數。可比較且等於自己的值 SHALL 仍以其自身作為 key。列印整份結果時，替身 SHALL 顯示為可讀的簡短形式，SHALL NOT 因為單一巨大值而讓輸出無法閱讀。

#### Scenario: Looking up an uncomparable value
- **WHEN** 呼叫端以原值取得替身，再用它索引計數結果
- **THEN** 得到該值的計數

#### Scenario: A column holding several NaNs
- **WHEN** 欄位中有三個 NaN，對它呼叫 `Counter`
- **THEN** 它們合併成一個項目，計數為 3
- **AND** 以原值取得的替身查得到那個計數
- **AND** 該計數與 `Count` 對同一個值的回答一致

#### Scenario: A value that contains a NaN
- **WHEN** 格子值是含 NaN 的陣列或結構
- **THEN** 同樣以替身作為 key，不會產生取不回來的項目

#### Scenario: Ordinary values are unchanged
- **WHEN** 結果中含一般的數字與字串
- **THEN** 仍能以 `counter[1]`、`counter["a"]` 直接取得

#### Scenario: Printing a counter that holds a large binary value
- **WHEN** 以 `%v` 列印整份計數結果
- **THEN** 每個替身顯示為型別加上截斷後的內容

### Requirement: Identity descends into composite values

編碼 SHALL 遞迴進入切片、陣列、map 與 struct 的元素，並對每個元素套用同一套型別規則。系統 SHALL NOT 讓型別不同但列印結果相同的巢狀值被視為同一個值。map 的編碼 SHALL 與其迭代順序無關。

#### Scenario: A nested integer and a nested string
- **WHEN** 比較 `[]any{1}` 與 `[]any{"1"}`
- **THEN** 兩者的識別不同，分組與計數都分開

#### Scenario: Grouping is unaffected for scalars
- **WHEN** 以一般的字串、整數、浮點數、布林值分組
- **THEN** 產生的 key 與此變更前逐位元組相同

