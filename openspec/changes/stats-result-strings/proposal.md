# Proposal: stats-result-strings

## Why

Only `ChiSquareTestResult` and `FactorAnalysisResult` can print themselves (#245, ST-9 in `api-review.md`). The other thirty-odd result types in `stats` have no `Show` and no `String`, so `fmt.Println(res)` prints Go's raw struct form: pointers as addresses, tables as their internal fields, everything on one line. The two `Show` methods also only write to standard output, so a result cannot go to a log, a file or an HTTP response without copying its fields by hand, and the two print different things in different layouts.

## What Changes

- Every exported result type in `stats` gets `String() string`, and one renderer produces all of them, so they read alike: a title line naming the analysis, then one line per exported field, named as in Go and in declaration order. The fields of an embedded `TestResult` come first, as they are read. A pointer or interface field that is `nil` does not apply and is left out. Numbers follow the rule the library uses for all text output (a plain decimal from 1e-6 up to 1e21). A list, and the rows of a table, print whole up to 60 entries and as the first 20 and last 5 past that, the rule the table views follow. A table field prints as an indented grid with its row and column names. The text carries no colour codes, so it is the same on a terminal, in a file and in a test.
- Every one of those types also gets `Show()`, which prints exactly `String()` and a newline to standard output. `fmt.Fprintln(w, res)` writes the same text to any `io.Writer`, which is what the review asked for, without a second method per type.
- `ChiSquareTestResult.Show()` and `FactorAnalysisResult.Show()` print the new text. `FactorAnalysisResult.Show` keeps its optional row range: with a range it shows each table through that range as before.
- A test fails the build of the test suite if an exported `…Result` struct type in `stats` lacks its own `String` or `Show`, so a new result type cannot skip them.
- `Docs/stats.md` explains printing once, under Common Result Structure; both CHANGELOGs, `api-review.md` (the `Show`/`String` part of ST-9), `delivery-status.md` and the `insyra` skill's rule on reading results are updated.

### Why `Show` is kept and defined through `String`

`String` is the Go convention: `fmt`, `log`, `%v`, test failure messages and any `io.Writer` use it with no extra API. `Show` is insyra's own verb, the one `DataList` and `DataTable` use, and people who work with the library call `res.Show()` as they call `dt.Show()`; a result that has no `Show` breaks that habit for no reason. Defining `Show` as printing `String` means the two can never disagree and there is only one layout to maintain. A `ShowTo(w)` per type was considered and rejected: `fmt.Fprintln(w, res)` already does it.

## Capabilities

### New Capabilities

- `stats-result-text`: how every `stats` result prints itself.

### Modified Capabilities

(none)

## Impact

- Code: a new `stats/result_text.go` (the renderer) and a new `stats/result_strings.go` holding every `String` and `Show`, kept apart from the files that declare the types so that work on those files does not collide with this change; `stats/chi_square.go` drops its `Show` and `stats/factor_analysis.go` rewrites the no-range path of its own.
- Tests: the renderer's rules (field order, nil fields, number text, list and table elision, tables), one output test per family of result, and the test that every result type has both methods.
- Docs: `Docs/stats.md`, `CHANGELOG.md`, `CHANGELOG_TW.md`, `api-review.md`, `delivery-status.md`, `skills/insyra/SKILL.md`.
- No new dependencies.
- Lands after `chisq-observed-expected`, which gives `ChiSquareTestResult` the tables this prints.
