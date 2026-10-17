# Tasks: parquet-ccl-streamed-sequences

## 1. internal/ccl

- [x] 1.1 The cumulative functions' loop moves into an accumulator shared with their stream; a stream for each built-in sequence function whose output, pushed in any batches, is bit for bit the function's on the whole column (failing tests first)
- [x] 1.2 `ResolveWholeTable` accepts a built-in sequence function at the top of an expression and refuses it anywhere else; a top-level sequence evaluator that reads its column argument from each batch the way the evaluator does (failing tests first)

## 2. parquet

- [x] 2.1 Batch contexts hold columns; `ApplyCCL` runs as a pipeline of statement stages, with sequence statements as stages that may hold rows back; resolution reads the runs leaving the earlier stages (failing tests first)
- [x] 2.2 `FilterWithCCL` streams a sequence filter, deciding held rows when their values arrive (failing tests first)
- [x] 2.3 Written columns are nullable and typed without loss: a column the file had keeps its type when every value fits, otherwise the type `Write` gives, settled from every value with a second pass only when a later value changes a guess; a value or type that cannot be written is an error, never a panic (failing tests first)

## 3. Docs, changelog, ledger

- [x] 3.1 `Docs/parquet.md`: which sequence functions work, where, and what they hold in memory
- [x] 3.2 `CHANGELOG.md` / `CHANGELOG_TW.md`: `parquet`
- [x] 3.3 `AGENTS.md`: the follow-up narrows to the third change; `delivery-status.md`

## 4. Verification

- [x] 4.1 gofmt, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate parquet-ccl-streamed-sequences --strict`
