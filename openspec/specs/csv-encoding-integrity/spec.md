# csv-encoding-integrity Specification

## Purpose
CSV 讀取時的文字解碼契約：解碼表內的字元集與常見別名會被解碼成 UTF-8，不會把可解碼的位元組原樣當成資料；表外的名稱，包括偵測器可能回報的少數字元集，在 `ReadCsvToString` 回傳錯誤，在其他讀取函式維持既有讀法。
## Requirements
### Requirement: Known charsets are decoded

讀取 CSV 時，指定或偵測到的編碼名稱（忽略大小寫與 `-`、`_`、空白、`.`、`:`）若屬於解碼表，SHALL 解碼成 UTF-8。解碼表 SHALL 包含 UTF-8／16／32、Big5、GB18030（含 GBK、GB2312）、Shift-JIS、EUC-JP、ISO-2022-JP、EUC-KR、Windows-1250 至 1258、ISO-8859-1 至 10 與 13 至 16、KOI8-R／U、IBM866、Macintosh。不在表內的名稱 SHALL 沿用既有的子字串規則，依序比對 `utf8`、`big5`、`gb`、`utf16`；比對 SHALL 在同樣忽略大小寫與分隔符號的正規化名稱上進行，大小寫 SHALL NOT 影響結果。都不符合時，`ReadCsvToString` 以外的讀取函式 SHALL 原樣讀取且不回傳錯誤；偵測器也可能回報的 ISO-2022-KR、ISO-2022-CN、IBM424 與 IBM420 屬於這種情況。

#### Scenario: Latin-1 file
- **WHEN** 以 `iso-8859-1` 讀取以 ISO-8859-1 編碼的 CSV
- **THEN** 儲存格是有效的 UTF-8 文字

#### Scenario: A detected charset outside the table
- **WHEN** `DetectEncoding` 回報 `ibm424_rtl`，以 `ReadCSV_File` 自動偵測讀取該 CSV
- **THEN** 內容原樣讀取，不回傳錯誤

#### Scenario: Legacy name outside the table
- **WHEN** 以 `big5-hkscs` 讀取 Big5 編碼的 CSV
- **THEN** 內容以 Big5 解碼

#### Scenario: Legacy name in mixed case
- **WHEN** 以 `BIG5-HKSCS`、`Utf-8-Sig` 或 `X-GBK` 讀取對應編碼的 CSV
- **THEN** 內容的解碼結果與小寫拼法相同

### Requirement: UTF-32 byte-order marks are recognised

`DetectEncoding` SHALL 在比對 UTF-16 BOM 之前先比對 UTF-32 BOM。

#### Scenario: UTF-32LE BOM
- **WHEN** 檔案以 `FF FE 00 00` 開頭
- **THEN** `DetectEncoding` 回傳 `utf-32le`

### Requirement: ReadCsvToString returns UTF-8 or an error

`csvxl.ReadCsvToString` SHALL 回傳 UTF-8 內容。指定或自動偵測到的編碼既不在解碼表、也不符合子字串規則時，SHALL 回傳列出支援編碼的錯誤，SHALL NOT 回傳未解碼的位元組。錯誤訊息 SHALL 同時列出解碼表的鍵與四組子字串家族，使讀者能分辨自己輸入的別名是否可用。

#### Scenario: An unknown name
- **WHEN** 以 `klingon-1` 呼叫 `ReadCsvToString`
- **THEN** 回傳錯誤，訊息含 `klingon-1`、支援的編碼與 `utf8`、`big5`、`gb`、`utf16` 四組家族

#### Scenario: A detected charset nothing decodes
- **WHEN** 自動偵測回報 `ibm424_rtl`，以 `ReadCsvToString` 讀取
- **THEN** 回傳錯誤，不回傳內容

### Requirement: An encoding argument means the same in csvxl as in the core readers

`csvxl`'s CSV readers SHALL detect the encoding when given an empty string or `"auto"` in any case, as the core CSV readers do.

#### Scenario: Uppercase AUTO on a Big5 file

- **WHEN** `ReadCsvToString(path, "AUTO")` reads a Big5 file
- **THEN** the text is decoded from Big5 instead of the call failing

### Requirement: An explicit encoding nothing decodes is an error in every CSV reader

`ReadCSV`, `ReadCSV_FileWithOptions`, `StreamCSV`, `csvxl.CsvToExcel` and `csvxl.AppendCsvToExcel` SHALL return an error listing the supported encodings when given an encoding name that is neither in the decoder table nor matched by its substring rules, and SHALL NOT read the bytes as they are. A charset the auto-detector reports SHALL be read as before.

#### Scenario: A mistyped name

- **WHEN** a Big5 file is read with the encoding "big6"
- **THEN** the call returns an error that names "big6" and the supported encodings

#### Scenario: A converter writes nothing

- **WHEN** CsvToExcel is given the encoding "big6"
- **THEN** it returns the error and creates no output file

### Requirement: Windows code page names for Big5 and GBK are accepted

The decoder table SHALL accept `cp950` as Big5 and `cp936` as GB18030, as it accepts `cp932` and `cp949`.

#### Scenario: cp950

- **WHEN** a Big5 file is read with the encoding "cp950"
- **THEN** its text is decoded from Big5

