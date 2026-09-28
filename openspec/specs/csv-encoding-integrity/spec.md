# csv-encoding-integrity Specification

## Purpose
How a CSV reader turns a named or detected encoding into text: a charset it can decode, or a common alias of one, is decoded; any other name is refused; undecoded bytes never become cells.

## Requirements
### Requirement: Text is decoded or refused

讀取 CSV 時，指定或偵測到的編碼若無對應解碼器，SHALL 回傳錯誤並列出支援的編碼，SHALL NOT 將未解碼的位元組原樣放進 DataTable。支援清單 SHALL 至少包含 UTF-8／UTF-16、Big5、GB18030、Shift-JIS、EUC-JP、EUC-KR、Windows-1250／1251／1252、ISO-8859-1／2／15。

#### Scenario: Latin-1 file
- **WHEN** 讀取以 ISO-8859-1 編碼的 CSV
- **THEN** 儲存格是有效的 UTF-8 文字

### Requirement: Common aliases of a supported charset are accepted

編碼名稱是受支援字元集的常見別名時，讀取 SHALL 以該字元集解碼，SHALL NOT 回傳 unsupported-encoding 錯誤。`utf-8-sig` 與 `utf-8-bom` SHALL 讀成 UTF-8 並去掉開頭的位元組順序標記；`big5-hkscs`、`csbig5`、`cn-big5`、`x-x-big5` SHALL 讀成 Big5；`x-gbk`、`gb_2312-80`、`csgb2312`、`csiso58gb231280`、`chinese`、`iso-ir-58` SHALL 讀成 GBK。名稱中的大小寫與分隔符號 SHALL NOT 影響比對。不屬於任何受支援字元集的名稱 SHALL 仍回傳錯誤。

#### Scenario: A file written as utf-8-sig
- **WHEN** 以 `utf-8-sig` 讀取開頭有 UTF-8 位元組順序標記的 CSV
- **THEN** 讀取成功，內容不含位元組順序標記

#### Scenario: A Big5 file named big5-hkscs
- **WHEN** 以 `BIG5-HKSCS` 讀取 Big5 編碼的 CSV
- **THEN** 儲存格是正確解碼的 UTF-8 文字

#### Scenario: A name no charset owns
- **WHEN** 以 `klingon-1` 讀取
- **THEN** 回傳 unsupported-encoding 錯誤

### Requirement: Byte-order marks and detector failures

`DetectEncoding` SHALL 在比對 UTF-16 BOM 之前先比對 UTF-32 BOM。偵測器無法判定字元集時 SHALL 記錄警告並回傳 `utf-8`，SHALL NOT 讓整個讀取失敗。

#### Scenario: UTF-32LE BOM
- **WHEN** 檔案以 `FF FE 00 00` 開頭
- **THEN** `DetectEncoding` 回傳 `utf-32le`

