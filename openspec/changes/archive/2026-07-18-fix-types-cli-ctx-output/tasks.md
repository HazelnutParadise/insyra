# Tasks — route `types` through ctx.Output

## 1. Core writer-aware methods (show.go)

- [x] 1.1 `(*DataTable) ShowTypesRangeTo(w io.Writer, startEnd ...any)`: move the body of `ShowTypesRange` here; convert every `fmt.Print*` to `fmt.Fprint*(w, ...)`.
- [x] 1.2 `(*DataTable) ShowTypes()` → `ShowTypesTo(os.Stdout)`; `ShowTypesTo(w)` → `ShowTypesRangeTo(w)`; `ShowTypesRange(startEnd...)` → `ShowTypesRangeTo(os.Stdout, startEnd...)`.
- [x] 1.3 `printTypeRows(w io.Writer, ...)`: add writer param, convert prints, update both call sites.
- [x] 1.4 Same four-method transform for `(*DataList)` (incl. nil-guard writing to `w`).

## 2. CLI wiring

- [x] 2.1 `cli/commands/types.go`: both cases call `ShowTypesTo(ctx.Output)`.

## 3. Tests

- [x] 3.1 New `cli/commands/types_test.go`: `types` on a DataList and a DataTable var writes "Type Info" into the captured `ctx.Output` buffer; unknown var still errors. Style of `m20_output_test.go`.

## 4. Docs & follow-up cleanup (same change)

- [x] 4.1 `Docs/DataTable.md` + `Docs/DataList.md`: add `ShowTypesTo`/`ShowTypesRangeTo` lines to the existing `ShowTypes`/`ShowTypesRange` signature blocks with the standard "same output, written to w" comment and description sentence.
- [x] 4.2 Delete the resolved `[2026-07-10]` `types` entry from `AGENTS.md` Follow-ups.

## 5. Verification

- [x] 5.1 `go build ./...`; `go test ./cli/... .` green; `gofmt -l` clean on touched files; `go vet ./...` clean.
