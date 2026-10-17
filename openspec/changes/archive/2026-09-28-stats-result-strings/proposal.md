# Proposal: stats-result-strings

## Why

Only `ChiSquareTestResult` and `FactorAnalysisResult` could print themselves (#245). The other result types printed Go's raw struct form (pointers as addresses, tables as internal fields), and the two `Show` methods wrote only to standard output in two different layouts. Ported from `0.4`; purely additive except that those two `Show` methods now print the shared layout.

## What Changes

- Every exported `…Result` type in `stats` gets `String()` and `Show()`, produced by one renderer: a title line, then one line per exported field in declaration order, with the embedded `TestResult` first. A `nil` pointer or interface field is left out. Numbers print as a plain decimal from 1e-6 up to 1e21. A list or table prints whole up to 60 entries, otherwise the first 20 and last 5. No colour codes.
- `Show()` prints `String()` and a newline. `fmt.Fprintln(w, res)` writes to any `io.Writer`.
- `ChiSquareTestResult.Show()` and `FactorAnalysisResult.Show()` print the new text; `FactorAnalysisResult.Show` keeps its row-range form.
- A test fails when an exported `…Result` struct lacks its own `String` or `Show`.

## Capabilities

### New Capabilities

- `stats-result-text`: how every `stats` result prints itself.

### Modified Capabilities

None.

## Impact

- `stats/result_text.go`, `stats/result_strings.go` and their tests, `stats/chi_square.go`, `stats/factor_analysis.go`, `Docs/stats.md`, both changelogs.
