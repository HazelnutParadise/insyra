# Tasks: group-keys-integers-by-value

## 1. Tests first

- [x] 1.1 `groupby_integer_keys_test.go`: grouping the value 1 in eleven integer widths, two float widths and as text gives groups of 11, 2 and 1; `OpNUnique` over `1`, `int64(1)`, `uintptr(1)`, `1.0`, `"1"` is 3. Against the old code the groups were `[10 1 2 1]` and the count 4.

## 2. Implementation

- [x] 2.1 `datatable_groupby.go` (`encodeGroupKey`, `uniqueKey`) and `cell_identity.go` (`encodeCell`): `uintptr` joins the integer arm.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`: the key rule at `GroupBy`.
- [x] 3.2 Skills: no change.
- [x] 3.3 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core.
- [x] 3.4 `api-review.md`: T-17's row names this change for its grouping half. `delivery-status.md`: a Latest Milestones entry.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` (0 issues), `openspec validate group-keys-integers-by-value --strict`.
