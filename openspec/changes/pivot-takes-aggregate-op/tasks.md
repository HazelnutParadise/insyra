# Tasks: pivot-takes-aggregate-op

## 1. Tests first

- [x] 1.1 `datatable_pivot_test.go`: the existing aggregation tests take `new(OpSum)`, `new(OpMean)`, `new(OpCount)`, `new(OpCustom)` and `new(AggregateOp(99))`; a nil `AggFunc` refuses duplicates while `new(OpSum)` sums them; every op from `OpSum` to `OpNUnique` is accepted. Against the old code the file does not compile: `AggFunc` was a string.
- [x] 1.2 `isr/pivot_test.go`: `Agg: new(insyra.OpSum)`.
- [x] 1.3 `cli/commands/pivot_test.go`: `agg` reads `sum`, `mean`, `avg`, `MAX`, `count` and `stddev`, and refuses `custom`, `average` and `wat` with `unknown aggregate op`. Against the old code `custom` failed with `AggFunc "custom" requires Custom func` and `wat` with `unknown AggFunc "wat"`, and `average` was accepted.

## 2. Implementation

- [x] 2.1 `datatable_pivot.go`: `AggFunc *AggregateOp`; a value past `OpCustom` or below `OpSum` is refused; `parseAggOpName` deleted.
- [x] 2.2 `isr/pivot.go`: `Agg *insyra.AggregateOp`. `cli/commands/pivot.go`: `agg` parsed with `parseAggregateOp`, stored as `*insyra.AggregateOp`.

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/DataTable.md`: `AggFunc` and `Custom` fields and the Pivot example. `Docs/cli-dsl.md` already says pivot's ops match groupby's; no change.
- [x] 3.2 Skills: no change. Neither skill names a Pivot field or a CLI aggregate name.
- [x] 3.3 `CHANGELOG.md` and `CHANGELOG_TW.md`: Core (BREAKING), `isr` (BREAKING), CLI.
- [x] 3.4 `api-review.md`: T-17 marked fixed, with `group-keys-integers-by-value` for its grouping half.
- [x] 3.5 `delivery-status.md`: a Latest Milestones entry.

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run` (0 issues), `openspec validate pivot-takes-aggregate-op --strict`.
