# Tasks: take-deps-held-by-go-1-25

## 1. Bump

- [x] 1.1 `go get` each module whose newest version declares `go 1.26`, at `@latest`, then `go mod tidy`; the `go` directive stays at 1.26.9
- [x] 1.2 `chromedp` at v0.16.0 with the `cdproto` it requires, after v0.17.0 to v0.19.1 (`go 1.27`) and v0.20.x (`snapshot-chromedp` no longer compiles) were tried
- [x] 1.3 Check every module that moved against GitHub's advisory database (none inside a range)

## 2. Verify

- [x] 2.1 On go1.26.9: `go build ./...`, `go vet ./...`, `go test -count=1 ./...`, `golangci-lint run` (0 issues)
- [x] 2.2 govulncheck v1.3.0 built with go1.26.9 completes on the new `chromedp` chain and reports no vulnerabilities
- [x] 2.3 A bar chart saved with `plot.SavePNG` through local Chrome is byte-identical with `chromedp` v0.12.1 and v0.16.0
- [x] 2.4 A table of integers, floats, strings, booleans and timestamps with nulls written with `parquet.Write` is byte-identical with `thrift` v0.24.0 and v0.25.0, and each build reads the other's file to the same values

## 3. Records

- [x] 3.1 No changelog entry: nothing a user of the library sees changes
- [x] 3.2 AGENTS.md: the Go 1.25 follow-up is replaced by one for `chromedp`, and the refresh rule names a broken build as a reason to hold back
- [x] 3.3 `delivery-status.md` handoff note
- [x] 3.4 `openspec validate take-deps-held-by-go-1-25 --strict` passes
