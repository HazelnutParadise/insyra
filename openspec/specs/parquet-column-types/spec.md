# parquet-column-types Specification

## Purpose
A Parquet file written by another tool carries column types this library never writes, and reading one is the main thing a Parquet reader is for. This capability fixes what each Arrow column type becomes in Go, insists that a cell is the value of its own row, and says what happens to a type with no faithful representation: nothing that looks like data, and a reason the caller can find. It exists because the reader used to answer an unknown type with the string form of the whole column, repeated in every row, and report success.
## Requirements
### Requirement: A column never reads back as a value it does not hold

讀取 Parquet 時，每一格 SHALL 是該列自己的值。系統 SHALL NOT 把整欄的字串傾印當成單一格的值，也 SHALL NOT 忽略列索引。

#### Scenario: A column type the reader does not handle
- **WHEN** 讀到沒有對應 Go 表示法的 Arrow 欄位型別
- **THEN** 該欄每一格為 `nil`
- **AND** 每一列的值互相獨立，不會全部相同

### Requirement: Every Arrow type with a faithful Go representation gets one

以下 Arrow 型別 SHALL 讀成對應的 Go 值：`Date32` 與 `Date64` 讀成 `time.Time`；`Int8`、`Int16`、`Uint8`、`Uint16`、`Uint32`、`Uint64` 讀成同名的 Go 整數型別；`Binary`、`LargeBinary`、`FixedSizeBinary` 讀成 `[]byte`；`LargeString` 讀成 `string`；`Decimal128` 與 `Decimal256` 讀成保留原始係數與 scale 的十進位值。轉換 SHALL NOT 捨入或改變數值。二進位欄位的格子 SHALL NOT 與文字欄位的格子無法區分。

#### Scenario: A date column written by another tool
- **WHEN** 讀取 `Date32` 或 `Date64` 欄位
- **THEN** 每一格是對應日期的 `time.Time`

#### Scenario: A decimal column
- **WHEN** 讀取 `Decimal128` 欄位
- **THEN** 每一格的值與檔案中的未縮放整數及 scale 完全相符，包含超出 `int64` 範圍的 38 位數值與負數

#### Scenario: An integer width the writer never emits
- **WHEN** 讀取 `Int8`、`Int16` 或任一無號整數欄位
- **THEN** 每一格是該列的整數值，且與其他 Go 整數型別以數值相等比較

#### Scenario: A binary column beside a text column
- **WHEN** 同一份檔案同時有 `Binary` 欄與 `String` 欄，且內容相同
- **THEN** 兩者的格子型別不同，分辨得出哪一欄是二進位
- **AND** 二進位欄的位元組完整保留，整欄以十六進位顯示，不因某一列剛好是合法 UTF-8 而改變顯示方式

### Requirement: A column that cannot be read says so

遇到沒有對應表示法的欄位型別時，系統 SHALL 在回傳的 DataTable 上記錄錯誤，內容指出欄位名稱與 Arrow 型別。系統 SHALL 仍然回傳其餘欄位可用的表格，SHALL NOT 因為單一欄位而讓整份檔案讀不了。

#### Scenario: A file mixing readable and unreadable columns
- **WHEN** 檔案同時含巢狀欄位與一般欄位
- **THEN** 一般欄位的值正確
- **AND** `Err()` 指出那一個欄位的名稱與型別

#### Scenario: Nothing is wrong
- **WHEN** 每一欄的型別都有對應表示法
- **THEN** `Err()` 為 nil

### Requirement: A decimal orders by value

十進位值 SHALL 以數值大小排序，SHALL NOT 以其文字形式的字典順序排序。

#### Scenario: Sorting a decimal column
- **WHEN** 比較 `9.5` 與 `10.2` 兩個十進位值
- **THEN** `9.5` 排在前面

### Requirement: A binary column survives a round trip

把含二進位欄位的檔案讀進來再寫出去時，該欄 SHALL 仍然是 Arrow 的二進位型別，SHALL NOT 變成字串欄。

#### Scenario: Read then write
- **WHEN** 讀取含 `Binary` 欄的檔案，未經修改直接寫回
- **THEN** 寫出的檔案中該欄仍是二進位型別，且位元組相同

