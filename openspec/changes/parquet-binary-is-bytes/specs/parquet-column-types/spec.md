## MODIFIED Requirements

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

### Requirement: A binary column survives a round trip

把含二進位欄位的檔案讀進來再寫出去時，該欄 SHALL 仍然是 Arrow 的二進位型別，SHALL NOT 變成字串欄。

#### Scenario: Read then write
- **WHEN** 讀取含 `Binary` 欄的檔案，未經修改直接寫回
- **THEN** 寫出的檔案中該欄仍是二進位型別，且位元組相同
