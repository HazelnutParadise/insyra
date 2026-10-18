# core-encoding-detection Specification

## Purpose
`DetectEncoding` 的取樣邊界契約：合法 UTF-8 檔案不因多位元組字元被取樣切開而誤判。

## Requirements
### Requirement: Sample boundary does not change the detected encoding

`DetectEncoding` 對合法 UTF-8 檔案 SHALL 回傳 `utf-8`，即使多位元組字元恰好被 8192 位元組的取樣邊界切開。只有填滿取樣的讀取結尾、且是某個字元尚未讀完的合法開頭時，才 SHALL 忽略；其他非法結尾 SHALL 照實判斷。

#### Scenario: Multi-byte character straddles the boundary
- **WHEN** 檔案前 8191 個位元組為 ASCII，接著是一個三位元組的中文字，再接更多內容
- **THEN** `DetectEncoding` 回傳 `utf-8`

#### Scenario: A short file with an invalid tail
- **WHEN** 檔案比 8192 位元組短，結尾是 Latin-1 位元組，例如 `name\nJos\xe9\n`
- **THEN** `DetectEncoding` 不回傳 `utf-8`，因為取樣就是整個檔案，結尾沒有被切開

### Requirement: Detection gives the same answer for the same bytes
`DetectEncoding` and the reader-based detection SHALL return the same charset every time they are given the same bytes, breaking a tie between equally confident recognizers by charset name.

#### Scenario: A short Big5 file where recognizers tie
- **WHEN** the same 24-byte Big5 sample is detected 300 times
- **THEN** every call returns the same charset
