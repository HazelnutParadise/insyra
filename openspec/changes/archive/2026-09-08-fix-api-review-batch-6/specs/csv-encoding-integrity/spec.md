## ADDED Requirements

### Requirement: Text is decoded or refused

讀取 CSV 時，指定或偵測到的編碼若無對應解碼器，SHALL 回傳錯誤並列出支援的編碼，SHALL NOT 將未解碼的位元組原樣放進 DataTable。支援清單 SHALL 至少包含 UTF-8／UTF-16、Big5、GB18030、Shift-JIS、EUC-JP、EUC-KR、Windows-1250／1251／1252、ISO-8859-1／2／15。

#### Scenario: Latin-1 file
- **WHEN** 讀取以 ISO-8859-1 編碼的 CSV
- **THEN** 儲存格是有效的 UTF-8 文字

### Requirement: Byte-order marks and detector failures

`DetectEncoding` SHALL 在比對 UTF-16 BOM 之前先比對 UTF-32 BOM。偵測器無法判定字元集時 SHALL 記錄警告並回傳 `utf-8`，SHALL NOT 讓整個讀取失敗。

#### Scenario: UTF-32LE BOM
- **WHEN** 檔案以 `FF FE 00 00` 開頭
- **THEN** `DetectEncoding` 回傳 `utf-32le`
