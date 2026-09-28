## ADDED Requirements

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
