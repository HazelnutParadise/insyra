# csv-read-options Delta Spec

## MODIFIED Requirements

### Requirement: Options-based CSV loading API

根套件 SHALL 提供 `CSVReadOptions` 結構與 `ReadCSV_FileWithOptions(filePath string, opts CSVReadOptions)`、`ReadCSV_StringWithOptions(csvString string, opts CSVReadOptions)` 兩個函數。`CSVReadOptions` SHALL 包含 `FirstColToRowNames bool`、`FirstRowToColNames bool`、`Encoding string`（僅檔案輸入有效，空字串或 `"auto"` 表示自動偵測）、`RawStrings bool`、`AllowRaggedRows bool` 與 `TrimLeadingSpace bool`。零值 options SHALL 產生與現有 `ReadCSV_File(path, false, false)` / `ReadCSV_String(s, false, false)` 完全相同的結果。

#### Scenario: Zero-value options match legacy behavior

- **WHEN** 呼叫 `ReadCSV_StringWithOptions(csv, CSVReadOptions{})`
- **THEN** 結果與 `ReadCSV_String(csv, false, false)` 相同，欄位型別推斷照常執行

#### Scenario: Row/col name flags map to legacy parameters

- **WHEN** 呼叫 `ReadCSV_FileWithOptions(path, CSVReadOptions{FirstColToRowNames: true, FirstRowToColNames: true})`
- **THEN** 結果與 `ReadCSV_File(path, true, true)` 相同

#### Scenario: Zero-value options stay strict on ragged input

- **WHEN** 以零值 options 載入各列欄數不一致的 CSV
- **THEN** 回傳 `wrong number of fields` 類錯誤，與現行行為相同

## ADDED Requirements

### Requirement: Ragged-rows tolerance

當 `CSVReadOptions.AllowRaggedRows` 為 true 時，系統 SHALL 接受各列欄數與首列不一致的 CSV：短列缺少的尾端欄位 SHALL 補為空字串 `""` 以維持列對齊；長列多出的 cell SHALL 保留，為其動態新增自動命名的欄位，且先前已讀入的各列在新欄位中 SHALL 為空字串。系統 SHALL NOT 截斷或丟棄任何 cell。`FirstRowToColNames` 為 true 時，欄名僅來自首列，動態新增的欄位使用自動命名；`FirstColToRowNames` 為 true 且該列僅有一個欄位時，該欄位 SHALL 作為列名，其餘資料欄為空字串。

#### Scenario: Trailer note row is padded

- **WHEN** 以 `AllowRaggedRows: true, FirstRowToColNames: true` 載入表尾含單欄註記列 `以上資料僅供參考` 的三欄 CSV
- **THEN** 載入成功，該列第一欄為註記文字，其餘欄為空字串

#### Scenario: Trailing comma produces one empty extra column

- **WHEN** 以 `AllowRaggedRows: true` 載入含 `2330,1000,` 這種列尾補逗號的 CSV（首列僅兩欄）
- **THEN** 載入成功，多出的 cell 進入自動新增的第三欄，該欄其他列為空字串

#### Scenario: Extra non-empty cells are kept

- **WHEN** 以 `AllowRaggedRows: true` 載入某列比首列多兩個非空 cell 的 CSV
- **THEN** 兩個 cell 皆保留於自動新增的欄位中，無資料遺失

#### Scenario: Works with RawStrings

- **WHEN** 以 `AllowRaggedRows: true, RawStrings: true` 載入欄數不齊的 CSV
- **THEN** 載入成功且所有 cell（含補入的空欄）皆為字串

### Requirement: Leading-space tolerance

當 `CSVReadOptions.TrimLeadingSpace` 為 true 時，系統 SHALL 忽略欄位的前導空白，包含引號欄位前的空白（對應 `encoding/csv` 的 `TrimLeadingSpace`）。預設 false 時行為 SHALL 與現行相同。

#### Scenario: Space before quoted field parses

- **WHEN** 以 `TrimLeadingSpace: true` 載入含 `2330, "1,000",600.86` 的 CSV
- **THEN** 載入成功，第二欄 cell 為 `1,000`（引號欄位正常解析）

#### Scenario: Default still rejects bare quote

- **WHEN** 以零值 options 載入同一列
- **THEN** 回傳 `bare " in non-quoted-field` 類錯誤，與現行行為相同

### Requirement: isr CSV input supports tolerance options

`isr` 套件的 `CSV_inOpts` SHALL 提供 `AllowRaggedRows bool` 與 `TrimLeadingSpace bool` 欄位，`DT.From(CSV{...})` SHALL 將其傳遞至 options 版讀取函數。

#### Scenario: isr passes tolerance options through

- **WHEN** 呼叫 `DT.From(CSV{String: csv, InputOpts: CSV_inOpts{AllowRaggedRows: true, TrimLeadingSpace: true}})` 載入含短列與引號前空白的 CSV
- **THEN** 載入成功，行為與直接呼叫 options 版函數相同
