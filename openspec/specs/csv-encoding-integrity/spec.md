# csv-encoding-integrity Specification

## Purpose
CSV 讀取時的文字解碼契約：偵測器能回報的字元集與常見別名都會被解碼成 UTF-8，不會把可解碼的位元組原樣當成資料；無法辨識的名稱維持既有讀法。

## Requirements
### Requirement: Known charsets are decoded

讀取 CSV 時，指定或偵測到的編碼名稱（忽略大小寫與 `-`、`_`、空白、`.`、`:`）若屬於解碼表，SHALL 解碼成 UTF-8。解碼表 SHALL 涵蓋偵測器可能回報的每一種字元集，至少包含 UTF-8／16／32、Big5、GB18030、Shift-JIS、EUC-JP、EUC-KR、Windows-1250 至 1258、ISO-8859 各分部、KOI8-R。不在表內的名稱 SHALL 沿用既有的子字串規則（`utf-8`、`big5`、`gb`、`utf-16`），都不符合時 SHALL 原樣讀取且不回傳錯誤。

#### Scenario: Latin-1 file
- **WHEN** 以 `iso-8859-1` 讀取以 ISO-8859-1 編碼的 CSV
- **THEN** 儲存格是有效的 UTF-8 文字

#### Scenario: Legacy name outside the table
- **WHEN** 以 `big5-hkscs` 讀取 Big5 編碼的 CSV
- **THEN** 內容以 Big5 解碼

### Requirement: UTF-32 byte-order marks are recognised

`DetectEncoding` SHALL 在比對 UTF-16 BOM 之前先比對 UTF-32 BOM。

#### Scenario: UTF-32LE BOM
- **WHEN** 檔案以 `FF FE 00 00` 開頭
- **THEN** `DetectEncoding` 回傳 `utf-32le`
