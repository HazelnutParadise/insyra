# Delivery Status

> **You are on `0.4` (2026-09-09).** This branch carries the whole-repo API review and its breaking changes: the library no longer terminates on a fatal, `Err()` is sticky, `gplot.SaveChart` returns an error, several methods stopped returning `nil`. `dev` stays on the 0.3.x line so feature releases can ship while the review continues, and `api-review.md`, the review's OpenSpec changes and every fix batch live here only. Non-breaking fixes are backported to `dev` as ordinary commits; merge `dev` into `0.4` after each 0.3.x change so this line keeps them.

## Current Phase
Production-readiness hardening (2026-09-06). Every package's exported surface was reviewed symbol by symbol, then the whole repository was reviewed beyond exported symbols (CLI command behaviour, CCL semantics, internal packages, security, test quality, repo/CI hygiene). The ledger is `api-review.md`; every open finding is a GitHub issue labelled `api-review` with a `severity:*` label (#205–#368). Fixes land in numbered OpenSpec batches: `fix-api-review-batch-1` through `-8` are archived; 114 findings remain open (7 high, 58 medium, 49 low).

`insyra/nn` phase 2 (training), `insyra/ml` v1, and the acceleration package are merged and stable; no feature workstream is active while the review backlog is worked down.

## Stage Objective
Close the `severity:high` issues first, then medium, with one OpenSpec change per batch. Findings that need an API decision (return shapes, `LogFatal`, name-vs-index, `IDataList` with unexported methods, `parallel`, `lp` auto-install, the Google Maps crawler) stay open until the owner decides; they are listed in the issue tracker, not here.

## Active Workstreams
None in flight. Every remaining `severity:high` issue carries a decision fork and needs the owner:
- #249 Google Maps crawler removal (DF-1), #257 `lp` runtime GLPK install (LP-1), #267 `csvxl` batch error policy (C-1), #271 `parallel` package future (P-1/P-4), #302 CVXPY reference run in CI (TS-4), #303 GPU/MNIST verifications in CI (TS-5), #341 CCL name-vs-index resolution (CCL-1, same decision as #225/T-11).

Also awaiting a ruling: `CSVWriteOptions.SanitizeFormulas`'s default (commented on #285), and what to do with two unmerged legacy branches (`copilot/fix-aa06638c-…`, `lingo-to-lp`).

## Latest Milestones
- 2026-09-09 `fix-api-review-batch-8`: nine CCL defects that produced a value for an expression that could not mean what it said. `&&`/`||`/`CASE` short-circuit; argument lists require commas; strings compare as text and a word against a number is an error; `nil` concatenates as the empty string; `&` binds looser than `+`/`-` and the docs gained a precedence table; a bare range and `LAG(@, …)` are errors; `AND()`/`OR()` check their arguments (the unreachable duplicate implementations are deleted); indices and windows must be whole numbers; fractional days keep sub-hour precision.
- 2026-09-08 `fix-api-review-batch-7`: six CLI defects where a command reported success after doing nothing. Shared target pre-checks, closed-set option parsing, `config` key and value validation, usage text matching the command, rejected extra arguments, range checks.
- 2026-09-08 `fix-api-review-batch-6`: I/O corruption. Every charset the detector can report now has a decoder and an undecodable file is an error rather than raw bytes in a cell; UTF-32 BOM recognised; SQLite identifiers quoted; `ToCSVWithOptions` with opt-in formula sanitising.
- 2026-09-07 `fix-api-review-batch-5`: six silently-wrong-answer defects. Exact integer ordering above 2^53, the Hermite basis, `KMeans` initial centres, `TryParseTime` layouts, `ShowTypes` column order, `Close()` dropping queued work. The non-breaking ones are backported to `dev`.
- 2026-09-07 `make-errors-non-terminating`: the library never terminates or panics by default; `Config.SetPanicOnError(true)` is the opt-in; `Err()` is sticky with `PopErr()`; chainable transforms never return `nil`.
- 2026-09-06 `fix-api-review-batch-4`: 21 high-severity, decision-free issues closed (CCL correctness and panics, atomic `ToCSV`/`ToJSON`, `AtomicDoAll` nested deadlock, csvxl sheet-name traversal, CLI nil-variable crash / NaN persistence / root flags / script REPL / password masking, empty and never-failing tests, CI `-run` pattern, tracked test binary).
- 2026-09-06 `bump-vulnerable-deps`: thrift, grpc, x/image, x/crypto past their advisories; `go` directive unchanged.
- 2026-09-06 `fix-api-review-batch-3`, `-2`; 2026-09-05 `-1`.
- 2026-09-05 quant: risk metrics, beta/CAPM, factor model, options pricing, portfolio optimisation, block bootstrap; CLI `quant` forms; datafetch TWSE adjusted prices; timeseries basics and CLI commands.
- 2026-08-06 `add-accel-execution-logging`, `add-nn-sequential-fit` archived.

## Blockers
- Multi-GPU wall clock and non-Apple GPU parity still need hardware nobody has (see AGENTS.md follow-ups).
- Cross-language, GPU, MNIST and CVXPY verifications run only where their toolchains exist; #302/#303 track making that a scheduled job.

## Next Verifiable Output
The next batch's tests green under `go test ./...`, `go test -race` on the touched packages, and `golangci-lint run`, with the corresponding issues closed and `api-review.md` rows marked.

## Next OpenSpec Change
Owner decision needed before proposing: pick from the decision-fork list above. Without a decision, the next decision-free work is the `severity:med` backlog (`gh issue list --label severity:med`) — the largest remaining coherent group is CCL error reporting and performance (#354, #355) and the CCL documentation gaps (#356).

## Decision Delta Since Previous Handoff
- CCL boolean coercion stays and the documentation was corrected, rather than the reverse: `IF`, `AND`, `OR`, `CASE`, `&&` and `||` all read their condition the same way, so a strict `&&` would have split the language (batch 8).
- CCL comparison resolves in one order — numbers, then text, then an error — so `'10' > '9'` keeps its documented numeric reading while `'abc' < 'abd'` gains a real one (batch 8).
- `ColumnRange`/`RowRange` stay internal and an expression that ends with one is an error, instead of exporting them so callers can type-assert an implementation detail (batch 8).
- A CLI check reads `ColNames()`/`RowNames()` rather than the `Get*` lookups, because a check must not record an error on the user's table; `checkTableErr` uses `PopErr` so a sticky error cannot fail the next REPL command (batch 7).
- `CSVWriteOptions.SanitizeFormulas` defaults off: it changes the value written, and `ToCSV`'s output stays as it was (batch 6, flagged on #285 for the owner to overrule).
- `AtomicDoAll` inside an `AtomicDo` on one of its instances now runs inline without locking the others (trust-zone rule), instead of deadlocking.
- CCL keywords are case-insensitive; out-of-range Excel references are errors; aggregates skip `NaN`; `MapContext` orders columns by name.
- `plot.SavePNG` online fallback is opt-in (batch 3).
