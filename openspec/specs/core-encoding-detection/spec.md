# core-encoding-detection Specification

## Purpose
How `DetectEncoding` and every reader that detects an encoding decide it. A valid UTF-8 file is not misjudged because the sample cuts a character in two, a guess the input does not clearly support is refused with `ErrEncodingAmbiguous` instead of being made, and the same input always gets the same answer.

## Requirements

### Requirement: Sample boundary does not change the detected encoding

`DetectEncoding` 對合法 UTF-8 檔案 SHALL 回傳 `utf-8`，即使多位元組字元恰好被 8192 位元組的取樣邊界切開。只有填滿取樣的讀取結尾、且是某個字元尚未讀完的合法開頭時，才 SHALL 忽略；其他非法結尾 SHALL 照實判斷。

#### Scenario: Multi-byte character straddles the boundary
- **WHEN** 檔案前 8191 個位元組為 ASCII，接著是一個三位元組的中文字，再接更多內容
- **THEN** `DetectEncoding` 回傳 `utf-8`

#### Scenario: A short file with an invalid tail
- **WHEN** 檔案比 8192 位元組短，內容是 ASCII，最後一行是 Latin-1 的 `Jos\xe9`
- **THEN** `DetectEncoding` 不回傳 `utf-8`，因為取樣就是整個檔案，結尾沒有被切開

### Requirement: An unclear guess is refused

When detection gets past the byte-order mark and the UTF-8 check, it SHALL use `chardet`'s best charset only when two things hold:
- its confidence is at least 50;
- no other charset has the same confidence.

Otherwise detection SHALL return an error that wraps `ErrEncodingAmbiguous`, lists the likeliest charsets with their confidence, and says to name the encoding. It SHALL return no charset with the error.

#### Scenario: Tied CJK charsets
- **WHEN** `DetectEncoding` reads a Big5 file holding `名稱,數量`, `蘋果,3` and `香蕉,5`, 24 bytes in all
- **THEN** it returns an error that wraps `ErrEncodingAmbiguous`
- **AND** the error names `big5`, `gb-18030` and `shift_jis`

#### Scenario: A Chinese header over numbers
- **WHEN** `ReadCSVFile` reads a Big5 file whose header is `名稱,數量,價格` over 400 rows of numbers, without an encoding
- **THEN** it returns an error that wraps `ErrEncodingAmbiguous`
- **AND** with `Encoding: "big5"` it reads the header as `名稱`

#### Scenario: A weak single guess
- **WHEN** `DetectEncoding` reads the Latin-1 file `name,city` / `café,Zürich`, which chardet scores 27
- **THEN** it returns an error that wraps `ErrEncodingAmbiguous`

#### Scenario: The CLI says how to read it
- **WHEN** `load` reads that 24-byte Big5 file without `encoding`
- **THEN** the error wraps `ErrEncodingAmbiguous` and contains `add encoding <name> to the load command`
- **AND** `load <file> encoding big5` reads the header as `名稱`

### Requirement: Detection gives the same answer every time

For the same input, detection SHALL return the same charset or the same error on every call. It SHALL NOT depend on the order in which `chardet`'s recognisers finish.

#### Scenario: Repeated calls on an ambiguous file
- **WHEN** `DetectEncoding` is called 50 times on the 24-byte Big5 file
- **THEN** every call returns the `ErrEncodingAmbiguous` error

#### Scenario: Repeated calls on a clear file
- **WHEN** `DetectEncoding` is called 20 times on a Big5 file of ten rows of Chinese text
- **THEN** every call returns `big5`
