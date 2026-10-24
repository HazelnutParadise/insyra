# Tasks: timestamps-are-times

## 1. Tests first

- [x] 1.1 `timestamps_test.go`: with an injected clock, two modifications within one second are told apart, the modification time does not move back, the old getters return Unix seconds, and `IsTheSameAs` still compares whole seconds
- [x] 1.2 `TestDataListGetCreationTimestamp` sets the clock instead of sleeping

## 2. Implementation

- [x] 2.1 an unexported `clock`; every place that records a time stores `clock().UnixNano()`
- [x] 2.2 `CreationTime` and `LastModifiedTime` on `DataList` and `DataTable`; the old getters Deprecated and returning Unix seconds
- [x] 2.3 `interfaces.go`: both interfaces gain the new methods
- [x] 2.4 `ml/models_test.go`: `bitwiseEqual` skips the two recorded-time fields

## 3. Docs and changelog

- [x] 3.1 `Docs/DataList.md` and `Docs/DataTable.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate timestamps-are-times --strict`
