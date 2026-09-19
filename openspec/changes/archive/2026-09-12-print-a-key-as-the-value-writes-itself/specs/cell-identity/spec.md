## MODIFIED Requirements

### Requirement: A stand-in key can be looked up and printed

不可雜湊的值在計數結果中 SHALL 以一個可雜湊的替身作為 key。系統 SHALL 提供由原值取得該替身的方法，讓呼叫端查得到計數。可比較的值 SHALL 仍以其自身作為 key。列印整份結果時，替身 SHALL 顯示為可讀的簡短形式：值若知道如何把自己寫成文字，SHALL 用它自己的寫法，SHALL NOT 顯示其內部欄位；否則顯示編碼後的內容。SHALL NOT 因為單一巨大值而讓輸出無法閱讀。識別 SHALL 仍然由完整編碼決定，SHALL NOT 改用文字形式，因為文字可能失真而讓不同的值被併成一組。

#### Scenario: Looking up an uncomparable value
- **WHEN** 呼叫端以原值取得替身，再用它索引計數結果
- **THEN** 得到該值的計數

#### Scenario: A value that knows how to write itself
- **WHEN** 不可比較的格子值實作了 `fmt.Stringer`
- **THEN** 列印時顯示它自己的文字
- **AND** 不顯示其內部欄位

#### Scenario: A value that does not
- **WHEN** 格子值是 `[]byte`、`[]int` 或 map
- **THEN** 維持原本的編碼內容顯示

#### Scenario: Ordinary values are unchanged
- **WHEN** 結果中含一般的數字與字串
- **THEN** 仍能以 `counter[1]`、`counter["a"]` 直接取得

#### Scenario: Printing a counter that holds a large binary value
- **WHEN** 以 `%v` 列印整份計數結果
- **THEN** 每個替身顯示為型別加上截斷後的內容
