# json-codec-conformance Specification

## Purpose
Makes insyra's JSON reading and writing behave the way Go's `encoding/json` does: `ReadJSON` accepts only valid JSON, keeps a number beyond `float64` as its text and replaces invalid UTF-8 as `encoding/json` does, and `ToJSON` writes the bytes `encoding/json.MarshalIndent` writes. insyra reads and writes JSON through `github.com/goccy/go-json` for speed; these requirements keep that choice from changing results.

## Requirements

### Requirement: ReadJSON accepts only JSON

`ReadJSON` and `ReadJSON_File` SHALL refuse input that is not valid JSON under RFC 8259, returning an error and no table.

#### Scenario: A leading zero is refused

- **WHEN** `ReadJSON` 讀 `[{"a":01}]`
- **THEN** 回傳錯誤，表為 nil

#### Scenario: A trailing comma is refused

- **WHEN** `ReadJSON` 讀 `[{"a":1,}]`
- **THEN** 回傳錯誤，表為 nil

### Requirement: A number outside float64's range keeps its text

`ReadJSON` SHALL read a JSON number that is neither an `int64` nor a finite `float64` as the string it was written as, the way `ReadCSV` reads the same text.

#### Scenario: 1e400

- **WHEN** `ReadJSON` 讀 `[{"a":1e400,"b":1}]`
- **THEN** 讀取成功，`a` 那格是字串 `"1e400"`，`b` 那格是 `int64(1)`

### Requirement: Invalid UTF-8 is replaced as encoding/json replaces it

`ReadJSON` SHALL replace each invalid UTF-8 byte inside a JSON string with U+FFFD, as `encoding/json` does.

#### Scenario: A stray 0xFF byte

- **WHEN** `ReadJSON` 讀的字串值裡有一個 0xFF 位元組
- **THEN** 那格的字串在該位置是 U+FFFD，與 `encoding/json` 解碼同一份輸入的結果相同

### Requirement: ToJSON writes what encoding/json writes

`ToJSON_Bytes` SHALL return, byte for byte, what `encoding/json.MarshalIndent` returns with a two-space indent for the same rows, and `ToJSON` and `ToJSON_String` SHALL write the same bytes.

#### Scenario: Small exponents, escapes and mixed cells

- **WHEN** 一張表含 `1e-7`、`2.5e-8`、`1e21`、`int64` 極值、`<a&b>`、U+2028、中文、`nil`、`bool`、`time.Time` 與 `float32(0.1)`
- **THEN** `ToJSON_Bytes(true)` 與 `ToJSON_Bytes(false)` 都與 `encoding/json.MarshalIndent` 對同樣的列輸出逐位元組相同，其中 `1e-7` 寫成 `1e-7`

### Requirement: Every JSON path uses one library

Every non-test Go file in the module SHALL read and write JSON through the one library `AGENTS.md` names, currently `github.com/goccy/go-json`. `golangci-lint run` SHALL fail when a non-test file imports `encoding/json` or another JSON library. Test files MAY import `encoding/json`, so it can serve as the independent reference insyra's output is checked against.

#### Scenario: A non-test file imports encoding/json

- **WHEN** 一個非測試的 `.go` 檔 import `encoding/json`
- **THEN** `golangci-lint run` 失敗，訊息指向 `AGENTS.md` 的規則

#### Scenario: A test file uses encoding/json as the reference

- **WHEN** 一個 `_test.go` 檔 import `encoding/json`，拿它的輸出當參考答案
- **THEN** `golangci-lint run` 不報錯

### Requirement: A faster conforming library replaces it everywhere

When a JSON library measures faster than the current one on `BenchmarkJSONPaths` and passes every `json-codec-conformance` test, every non-test file SHALL move to it in one change. That change SHALL update the lint rule and `AGENTS.md` to name it, and SHALL record both libraries' `BenchmarkJSONPaths` results. A library that is faster only under settings that break a `json-codec-conformance` test SHALL NOT be adopted with those settings.

#### Scenario: A faster library is found

- **WHEN** 新的 JSON 庫在 `BenchmarkJSONPaths` 上比現用的快，而且通過所有 `json-codec-conformance` 測試
- **THEN** 同一個變更把所有非測試檔換成它，lint 規則與 `AGENTS.md` 改成它的名字，並記錄兩者的測速結果

#### Scenario: Faster only by departing from encoding/json

- **WHEN** 某個庫只有在不排序 map 鍵、不跳脫 HTML 等偏離 `encoding/json` 的設定下才比較快
- **THEN** 不採用那組設定
