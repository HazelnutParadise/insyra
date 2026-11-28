# Tasks: preprocess-package

## 1. Package

- [x] 1.1 `preprocess/scale.go`, `encode.go`, `imputer.go`, `enum_names.go`, moved from the root package; functions take the table first.
- [x] 1.2 `internal/prep`: the selector rule, the type-7 quantile, the missing-value tests, used by the core and `preprocess`.

## 2. Callers

- [x] 2.1 Remove the root names and the `IDataTable` methods.
- [x] 2.2 `ml`, `nn` tests, `internal/dsl/commands` (`scale`, `encode`, `show`), `internal/dsl/env` and `allpkgs` use `preprocess`.

## 3. Tests

- [x] 3.1 Move the root tests to `preprocess/`.
- [x] 3.2 Benchmarks on 100,000 rows for `StandardScale`, `MinMaxScale`, `OneHotEncode`, `LabelEncode` and `SimpleImputer`.

## 4. Docs

- [x] 4.1 `Docs/preprocess.md`; pointers in `Docs/DataTable.md`; names in `Docs/DataList.md`, `Docs/ml.md`, `Docs/nn.md`; rows in `README.md`, `README_TW.md`, `Docs/README.md`.
- [x] 4.2 `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`, BREAKING marked.

## 5. Verify

- [x] 5.1 `go build ./...`, `go vet ./...`, `go test ./...`, `openspec validate preprocess-package --strict`.
