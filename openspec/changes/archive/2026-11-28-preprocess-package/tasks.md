# Tasks: preprocess-package (0.3.x)

## 1. Package

- [x] 1.1 Port `preprocess/` and `internal/prep/` from `0.4`; add `internal/utils/text.go`; read JSON through `github.com/goccy/go-json`.
- [x] 1.2 Decode a scaler JSON `"ref"` written as a plain string, name first then index, and write it back unchanged.
- [x] 1.3 Mark the core names Deprecated, naming the replacement; `//nolint:staticcheck` with a reason on the uses that stay.
- [x] 1.4 `ml` accepts the `preprocess` types in `transformerInputTypes`, `applyONNXTransformer` and the `Transformer` assertions; `allpkgs` registers the package.

## 2. Tests

- [x] 2.1 The `0.4` package tests, adapted to `0.3.x`.
- [x] 2.2 `preprocess/scale_legacy_json_test.go` and `scale_core_json_test.go`: core JSON loads, transforms like the core scaler, name first, empty reference refused.
- [x] 2.3 `ml/onnx_export_preprocess_test.go`: the same ONNX bytes for a pipeline of `preprocess` types as for the core types.

## 3. Docs

- [x] 3.1 `Docs/preprocess.md`; deprecation notes in `Docs/DataTable.md`; `Docs/ml.md`; rows in `README.md`, `README_TW.md`, `Docs/README.md`.
- [x] 3.2 `CHANGELOG.md` and `CHANGELOG_TW.md` under `## Unreleased`.

## 4. Verify

- [x] 4.1 `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run`, `openspec validate preprocess-package --strict`.
