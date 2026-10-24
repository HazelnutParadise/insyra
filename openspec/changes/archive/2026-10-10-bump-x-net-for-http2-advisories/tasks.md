# Tasks: bump-x-net-for-http2-advisories

## 1. Bump

- [x] 1.1 `go get golang.org/x/net@v0.60.0` and `go mod tidy`; the `go` directive stays at 1.26.9
- [x] 1.2 Check every module that moved against GitHub's advisory database (none inside a range)

## 2. Verify

- [x] 2.1 `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` on go1.26.9
- [x] 2.2 `govulncheck ./...` on go1.26.9 reports no reachable vulnerability

## 3. Records

- [x] 3.1 No changelog entry: nothing a user of the library sees changes
- [x] 3.2 `delivery-status.md` drops the red-scan blocker and records the milestone
- [x] 3.3 `openspec validate bump-x-net-for-http2-advisories --strict` passes
