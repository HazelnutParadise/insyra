# Tasks: shuffle-in-place

## 1. Tests first

- [x] 1.1 `shuffled_test.go`: `Shuffled` leaves a list and a table as they were, agrees with `Shuffle` for one seed, and keeps rows whole

## 2. Implementation

- [x] 2.1 `Shuffled` on `DataList` and `DataTable` sharing one body with `Shuffle`
- [x] 2.2 `interfaces.go` and the CLI's `shuffle`

## 3. Docs and changelog

- [x] 3.1 `Docs/DataList.md` and `Docs/DataTable.md`
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate shuffle-in-place --strict`
