# Tasks: mode-returns-every-mode

## 1. Tests first

- [x] 1.1 Add `datalist_mode_test.go`. Against the old code, `Mode([1 2 3])`, `[1 1 2 2]`, `[7]` and `[NaN NaN 1]` return nothing, and `[3 3 1 1 2]` returns `[3 1]`.

## 2. Implementation

- [x] 2.1 `datalist.go` `Mode`:
  - drop the rule that returns nil when every value ties;
  - skip `NaN`;
  - sort the result.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataList.md`: the `Mode` section.
- [x] 3.2 `CHANGELOG.md`/`CHANGELOG_TW.md`: a BREAKING entry under Core, and a note under CLI for `mode`.
- [x] 3.3 `api-review.md` D-13, and `delivery-status.md`. These are shared with `datalist-quantile`.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate mode-returns-every-mode --strict`
