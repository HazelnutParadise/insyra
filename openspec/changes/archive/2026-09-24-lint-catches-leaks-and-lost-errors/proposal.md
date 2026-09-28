## Why

`.golangci.yml` ran golangci-lint's default set and nothing else (#280, RP-8). Measured on 2026-09-24 against the whole repository, the candidates came out as: `nilerr` 1, `bodyclose` 1, `rowserrcheck` 4, `sqlclosecheck` 0, `errorlint` 55, `unparam` 63, `noctx` 63, `gosec` 450 — the last dominated by what a data library does on purpose (opening a path the caller names, `math/rand` for sampling, writing files `0644`, running Python).

The owner chose on 2026-09-24 to enable the five that watch for a resource left open and an error lost or hidden, and not `gosec`, `unparam` or `noctx`. Those five are the class of defect this review keeps finding: `ReadSQLStream` holding a connection, `csvxl` reporting the wrong path.

`nilerr`'s one finding is a real defect. `insyra env import` without `--force` must refuse to overwrite a non-empty environment, and decides emptiness in `isEnvironmentEmpty`. When `config.json` exists but cannot be read, that function returned "empty", so an import overwrote the environment without `--force`. The two checks above it, on `state.json` and `history.txt`, ignore every error the same way; `nilerr` did not flag them only because they are written as `err == nil && …`.

## What Changes

- `.golangci.yml` enables `nilerr`, `bodyclose`, `rowserrcheck`, `sqlclosecheck` and `errorlint`, and every finding is resolved.
- `isEnvironmentEmpty` treats a missing file as empty and any other failure to read `state.json`, `history.txt` or `config.json` as a reason to refuse, so the guard in front of an overwrite fails closed.
- `errorlint`: the 50 `fmt.Errorf` calls that formatted an error with `%v` or `%s` wrap it with `%w`, so a caller's `errors.Is`/`errors.As` reaches the cause; the five comparisons and assertions on errors that may be wrapped use `errors.Is`/`errors.As`.
- `rowserrcheck`: the four tests that iterate `*sql.Rows` check `rows.Err()`.
- `bodyclose`: its one finding is a false positive — `readJSONResponse` closes the body — and is marked as such where it is reported.

## Capabilities

### New Capabilities
- `lint-coverage`: which linters CI runs beyond the default set, and why.

### Modified Capabilities
None.

## Impact

- `.golangci.yml`; `cli/env/manager.go`; the files `errorlint` names; `datatable_to_sql_test.go`; `datafetch/twstock.go`.
- Both changelogs (CLI fix; wrapped errors).
- `api-review.md` RP-8, `delivery-status.md`, issue #280 (which stays open for RP-7).
