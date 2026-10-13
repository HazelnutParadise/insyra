## ADDED Requirements

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
