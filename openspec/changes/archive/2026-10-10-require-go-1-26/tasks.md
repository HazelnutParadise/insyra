# Tasks: require-go-1-26

## 1. Version

- [x] 1.1 `go.mod`'s `go` directive becomes `1.26.9`; `go mod tidy` changes no dependency version
- [x] 1.2 The govulncheck workflow asks for `1.26.x` with `check-latest: true`

## 2. Docs and records

- [x] 2.1 `Docs/README.md` and every tutorial say Go 1.26+ instead of Go 1.25+
- [x] 2.2 BREAKING entry in both changelogs
- [x] 2.3 `delivery-status.md` decision log entry

## 3. Verify

- [x] 3.1 On go1.26.9: `gofmt -l .`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`
- [x] 3.2 `govulncheck ./...` on go1.26.9 reports only the five `x/net` advisories, which `bump-x-net-for-http2-advisories` fixes
- [x] 3.3 `openspec validate require-go-1-26 --strict` passes
