# Proposal: ccl-numbers-as-plain-text

## Why

Every path that wrote a number as text used Go's default formatting, so a float outside roughly 1e-4..1e21 or with enough digits came out in exponent form: `'x' & 0.0000001` gave `x1e-07`, and `LEN(1000000)` was 5 while the same value read from an integer column gave 7. Inside a string built by CCL, `id-1e-07` reads back as nothing useful.

## What Changes

- `&`, `CONCAT` and the string functions that convert a number write a finite `float64` or `float32` below 1e21 as the shortest plain decimal. Larger values, NaN and the infinities keep Go's spelling.
- `ToCSV`, `ToJSON` and `Show` are unchanged: a number cell in exponent form reads back as the same value.
- BREAKING for expressions that build text from a small or many-digit float.

## Capabilities

### Modified Capabilities

- `ccl-type-semantics`: see What Changes.

## Impact

`internal/ccl/stdlib_string.go`, `stdlib.go`, `ccl_evaluator.go`, `internal/ccl/float_text_test.go`, `Docs/CCL.md`, both CHANGELOGs, `AGENTS.md` follow-up removed.
