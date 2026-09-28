# Proposal: one-json-library

## Why

insyra reads and writes JSON through two libraries. `github.com/goccy/go-json` serves `ReadJSON`, `ToJSON`, `py` and two `datafetch` clients; `encoding/json` serves the other ten non-test files: the CLI's config and environment state, the scalers' JSON methods, SafeTensors headers, TWSE responses and the accel probe. Two libraries can disagree: until go-json v0.11 the same small number was `1e-07` on one path and `1e-7` on the other. The owner decided on 2026-09-28 that every JSON path uses one library, the fastest one that behaves like `encoding/json`, and that a faster library found later replaces it everywhere at once.

## What Changes

- The ten non-test files that import `encoding/json` import `github.com/goccy/go-json` instead. Every type they use from it (`Number`, `RawMessage`, `Delim`, `Token`, the error types) is an alias of the `encoding/json` type in go-json, and the decoder methods they call (`Token`, `UseNumber`, `More`) exist there, so no signature changes. `Marshaler` and `Unmarshaler` are go-json's own interfaces with the same method sets, so the scalers still satisfy `encoding/json`'s, which the existing scaler tests exercise through `encoding/json`.
- `nn/safetensors.go` checks for a trailing value after the header with `errors.Is(err, io.EOF)` instead of `==`. errorlint accepts `==` only for standard-library functions known to return `io.EOF` unwrapped, so it flags the go-json decoder. go-json returns `io.EOF` itself at a clean end, which the existing SafeTensors tests show, so nothing changes.
- `golangci-lint` gains a `depguard` rule that fails on `encoding/json`, and on the other common JSON libraries, in any non-test file. Tests may still import `encoding/json`: several use it as the independent reference insyra's output is checked against, which is what the `json-codec-conformance` requirements ask for.
- `BenchmarkJSONPaths` measures `ToJSON_Bytes` and `ReadJSON` on a fixed 100,000 × 5 table, so a candidate library is compared on insyra's own paths.
- `AGENTS.md` states the rule, the library it currently names, and how a switch is made.
- No output changes: go-json v0.11 writes what `encoding/json` writes, which `json-codec-conformance` pins. No changelog entry.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `json-codec-conformance`: one library for every JSON path, enforced by lint; a faster conforming library replaces it everywhere in one change.

## Impact

- `accel/native_probe.go`, `cli/commands/config.go`, `cli/env/{cell_codec,config,manager,state,variable_codec}.go`, `datafetch/twstock.go`, `datatable_scale.go`, `nn/safetensors.go`, `.golangci.yml`, a benchmark file, `AGENTS.md`.
- Builds on `take-go-json-v0-11`.
