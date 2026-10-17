# Tasks: env-create-leaves-nothing-behind

## 1. Tests first

- [x] 1.1 `cli/env/create_cleanup_test.go`: a directory at a default file path, and a second Create

## 2. Implementation

- [x] 2.1 `Create` and `writeDefaultFiles` in `cli/env/manager.go`; `EnsureDefaultEnvironment` checks `ErrAlreadyExists`

## 3. Changelog

- [x] 3.1 `CHANGELOG.md` and `CHANGELOG_TW.md`: CLI

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate env-create-leaves-nothing-behind --strict`
