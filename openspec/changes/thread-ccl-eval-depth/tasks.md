# Tasks: thread-ccl-eval-depth

## 1. Pin current behavior before touching it

- [x] 1.1 Reproduce the #191 A/B on this host: benchmark `AddColUsingCCL("C", "SQRT(A*A + B*B) * 2 + A / B - 1")` at 10k and 100k rows with the current guards and with both guards stripped (locally, not committed), recording the session's own baseline and upper bound.
  - Same session, Apple M3, `go test -run '^$' -bench 'BenchmarkIssue191_AddColUsingCCL' -benchtime=3x -count=1 .`: current global guards `18.534 ms` (10k) / `141.641 ms` (100k); guards stripped locally `3.519 ms` / `34.320 ms`. The temporary stripped patch was restored before continuing.
- [x] 1.2 Write the eval-depth limit test before refactoring: build an AST deeper than `maxEvalDepth` directly (the compiler's `checkASTDepth` blocks the parse route), assert the existing error message. It must pass against the current implementation.
- [x] 1.3 Write the func-call guard test before refactoring, via the internal seam if the limit is unreachable from the public API; assert the existing error message. It must pass against the current implementation.

## 2. Thread the parameters

- [x] 2.1 Change `evaluateWithContext` to take a `depth int`, pass `depth+1` at every recursive site including the comparison and fold helpers, start every public entry at 0, and compare against `maxEvalDepth` with the unchanged error message.
- [x] 2.2 Give `callFunction` its own threaded counter along the funcCallNode path with `maxFuncCallDepth` and its unchanged error message, per design Decision 2.
- [x] 2.3 Delete `evalDepthByGoid`, `funcCallDepthByGoid`, both reset helpers and their call sites, and the `goid` import from `internal/ccl`. Do NOT touch `go.mod` — `internal/core/atomic.go` still uses `goid`; leave a note in the commit recording that the issue's dependency-removal claim stops at the package boundary.

## 3. Verification

- [x] 3.1 The 1.2 and 1.3 limit tests still pass unmodified — both guards trip with identical messages.
- [x] 3.2 Full `go test ./internal/ccl/...` and the root-package CCL suites (`AddColUsingCCL`, `EditCol*UsingCCL`, `ExecuteCCL` tests) pass with identical results.
- [x] 3.3 A race-detector test evaluates expressions concurrently across goroutines and reports clean.
- [x] 3.4 Re-run the 1.1 benchmark with the threaded implementation in the same session; record threaded vs global vs stripped times and the recovery fraction in this tasks file.
  - Same Apple M3 session, `-benchtime=10x`: global guard `15.512 ms` / `143.714 ms`; threaded `3.429 ms` / `23.851 ms`; guards stripped `3.466 ms` / `23.992 ms` at 10k / 100k rows. Recovery `(global-threaded)/(global-stripped)` was `100.3%` / `100.1%` respectively, within benchmark noise of the stripped upper bound.

## 4. Docs, changelog, skills

- [x] 4.1 Add the entry under `## Unreleased` in both `CHANGELOG.md` and `CHANGELOG_TW.md` under `### Core`, stating the measured speedup range for CCL column expressions.
- [x] 4.2 Check `Docs/ccl.md` (and any CCL doc pages) and `skills/insyra/` for claims about the guard mechanism or CCL evaluation cost; sync only what mentions them. No matching mechanism or cost claim required a documentation change.

## 5. Bookkeeping

- [x] 5.1 Record the decision delta in `delivery-status.md`, including that the pre-change "CCL transform chains 11x on GPU" measurement is invalidated as a comparison baseline (single-core interpreter with the old guards), which bears on whether CCL value expressions are ever worth accelerating.
- [x] 5.2 `openspec validate thread-ccl-eval-depth --strict` passes before handoff.
