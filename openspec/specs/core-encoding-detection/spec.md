# core-encoding-detection Specification

## Purpose
`DetectEncoding` 的取樣邊界契約：合法 UTF-8 檔案不因多位元組字元被取樣切開而誤判。

## Requirements
### Requirement: Sample boundary does not change the detected encoding

`DetectEncoding` 對合法 UTF-8 檔案 SHALL 回傳 `utf-8`，即使多位元組字元恰好被 8192 位元組的取樣邊界切開。

#### Scenario: Multi-byte character straddles the boundary
- **WHEN** 檔案前 8191 個位元組為 ASCII，接著是一個三位元組的中文字，再接更多內容
- **THEN** `DetectEncoding` 回傳 `utf-8`

