## MODIFIED Requirements

### Requirement: A stand-in key can be looked up and printed

不可雜湊的值在計數結果中 SHALL 以一個可雜湊的替身作為 key。不等於自己的值（NaN，以及任何含有 NaN 的陣列或結構）SHALL 比照辦理，因為用它當 key 會產生永遠取不回來的項目。系統 SHALL 提供由原值取得該替身的方法，讓呼叫端查得到計數。整數以外、可比較且等於自己的值 SHALL 仍以其自身作為 key；整數依「Integers are counted by value」以數值作為 key。列印整份結果時，替身 SHALL 顯示為可讀的簡短形式：值若知道如何把自己寫成文字，SHALL 用它自己的寫法，SHALL NOT 顯示其內部欄位；否則顯示編碼後的內容。SHALL NOT 因為單一巨大值而讓輸出無法閱讀。識別 SHALL 仍然由完整編碼決定，SHALL NOT 改用文字形式，因為文字可能失真而讓不同的值被併成一組。

#### Scenario: Looking up an uncomparable value
- **WHEN** 呼叫端以原值取得替身，再用它索引計數結果
- **THEN** 得到該值的計數

#### Scenario: A value that knows how to write itself
- **WHEN** 格子值實作了 `fmt.Stringer`，例如十進位金額
- **THEN** 列印時顯示它自己的文字，例如 `-340.0221114815`
- **AND** 不顯示其內部欄位

#### Scenario: A value that does not
- **WHEN** 格子值是 `[]byte`、`[]int` 或 map
- **THEN** 維持原本的編碼內容顯示

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

## ADDED Requirements

### Requirement: Integers are counted by value

`Counter` 與 `ToMapKey` SHALL 以數值識別整數，不論儲存的寬度：`int`、`int8`～`int64`、`uint`～`uint64` 與 `uintptr` 中數值相同者 SHALL 是同一個 key，與 `Count` 的比對規則一致。`int` 能容納的整數 SHALL 以 `int` 作為 key，使呼叫端直接寫的整數字面值查得到；`int` 容納不下的 SHALL 以 `int64`（有號）或 `uint64`（無號）作為 key。負數 SHALL NOT 與無號整數合併。浮點數與字串 SHALL NOT 與整數合併。

#### Scenario: A loaded column indexed with a literal
- **WHEN** 從 CSV 讀入一欄 `5, 5, 3`（儲存為 `int64`），對它呼叫 `Counter`
- **THEN** `counter[5]` 為 2

#### Scenario: Widths mixed in one column
- **WHEN** 一欄持有 `int64(5)`、`5`、`uint8(5)`、`int32(5)`、`"5"`、`5.0`
- **THEN** `counter[5]` 為 4，與 `Count(5)` 相同
- **AND** `"5"` 與 `5.0` 各自是獨立的 key

#### Scenario: A value int cannot hold
- **WHEN** 一欄持有 `uint64` 最大值與同值的 `uint`
- **THEN** 兩者合併在型別為 `uint64` 的 key 之下，計數為 2

#### Scenario: Indexing with another width
- **WHEN** 呼叫端以 `insyra.ToMapKey(int64(5))` 索引計數結果
- **THEN** 得到與 `counter[5]` 相同的計數
