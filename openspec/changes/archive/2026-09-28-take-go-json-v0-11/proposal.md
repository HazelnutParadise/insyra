# Proposal: take-go-json-v0-11

## Why

`ReadJSON`, `ReadJSON_File`, `ToJSON` and its variants, `py`'s result exchange and two `datafetch` clients read and write JSON through `github.com/goccy/go-json`. Its v0.11 line, released on 2026-09-27, rewrites the encoder and decoder to behave like `encoding/json` and is faster. The pre-release refresh (refresh-deps-for-0-3-4) held it at v0.10.6 because it changes what insyra returns, which the refresh rule sends to a change of its own. The owner decided on 2026-09-28 to take it in Huashan v0.3.4.

Measured on the M3 against v0.10.6, on a 100,000 × 5 table whose JSON is 14 MB, best of 5: `ToJSON_Bytes` 66 → 45 ms, `ReadJSON` 129 → 90 ms; go-json's own `MarshalIndent` 57 → 35 ms and `Decoder` with `UseNumber` 66 → 28 ms.

## What Changes

- `github.com/goccy/go-json` v0.10.6 → v0.11.1. It declares `go 1.21` and adds no module to the graph.
- What insyra returns changes, each time to what `encoding/json` does, measured on 2026-09-28:
  - **BREAKING**: `ReadJSON` and `ReadJSON_File` refuse input that is not JSON, such as a number with a leading zero (`{"a":01}`), which v0.10.6 read as 1.
  - A number outside `float64`'s range, such as `1e400`, is read as the text it was written as, which is what the existing fallback in `coerceJSONNumber` returns and what `ReadCSV` does with the same text. v0.10.6 failed the whole read.
  - Invalid UTF-8 inside a JSON string becomes U+FFFD instead of passing through as raw bytes.
  - `ToJSON`, `ToJSON_Bytes` and `ToJSON_String` write what `encoding/json` writes, byte for byte. A small exponent is now `1e-7` instead of `1e-07`; the value is the same.
  - Decoding errors are worded as `encoding/json` words them.
- The go-json follow-up in `AGENTS.md` is resolved and removed. The number-to-text follow-up is updated, because `ToJSON` no longer writes the same exponent form as `ToCSV`.

## Capabilities

### New Capabilities

- `json-codec-conformance`: insyra's JSON reading and writing behave as `encoding/json` does.

### Modified Capabilities

(none)

## Impact

- `go.mod`, `go.sum`, a test file in the root package, `Docs/DataTable.md`, both changelogs, `AGENTS.md`.
- Every downstream program resolves go-json to at least v0.11.1.
