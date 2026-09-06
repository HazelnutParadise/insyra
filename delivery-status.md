# Delivery Status

## Current Phase
Production-readiness hardening (2026-09-06). Every package's exported surface was reviewed symbol by symbol, then the whole repository was reviewed beyond exported symbols (CLI command behaviour, CCL semantics, internal packages, security, test quality, repo/CI hygiene). The ledger is `api-review.md`; every open finding is a GitHub issue labelled `api-review` with a `severity:*` label (#205–#368). Fixes land in numbered OpenSpec batches: `fix-api-review-batch-1` through `-4` are archived.

`insyra/nn` phase 2 (training), `insyra/ml` v1, and the acceleration package are merged and stable; no feature workstream is active while the review backlog is worked down.

## Stage Objective
Close the `severity:high` issues first, then medium, with one OpenSpec change per batch. Findings that need an API decision (return shapes, `LogFatal`, name-vs-index, `IDataList` with unexported methods, `parallel`, `lp` auto-install, the Google Maps crawler) stay open until the owner decides; they are listed in the issue tracker, not here.

## Active Workstreams
None in flight. Next pickup is the remaining `severity:high` issues that carry a decision fork:
- #205 `LogFatal` removal (K-1), #206 `LogLevelError` (K-3), #216 `DataList` nil returns (D-3), #249 Google Maps crawler (DF-1), #257 `lp` runtime GLPK install (LP-1), #267 `csvxl` batch error policy (C-1), #271 `parallel` package future (P-1/P-4), #302 CVXPY reference run in CI (TS-4), #303 GPU/MNIST verifications in CI (TS-5).

## Latest Milestones
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
Owner decision needed before proposing: pick from the decision-fork list above. Without a decision, the next decision-free work is the `severity:med` backlog (`gh issue list --label severity:med`).

## Decision Delta Since Previous Handoff
- `AtomicDoAll` inside an `AtomicDo` on one of its instances now runs inline without locking the others (trust-zone rule), instead of deadlocking.
- CCL keywords are case-insensitive; out-of-range Excel references are errors; aggregates skip `NaN`; `MapContext` orders columns by name.
- `plot.SavePNG` online fallback is opt-in (batch 3).
