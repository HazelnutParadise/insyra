# Proposal: lp-never-returns-a-nil-table

## Why

`lp.SolveFromFile` and `lp.SolveModel` return `nil` as their first DataTable on every failure path, and `Docs/lp.md`'s own example calls `result.Show()` on it. A nil `*DataTable` panics on `Show()`; measured on 2026-09-12. So the documented way to use the package crashes the program whenever the solve does not work — a timeout, a solver error, an unwritable temp file, or more than one `timeoutSeconds` argument, which returns `nil, nil`.

There is also a path that reports success and hands back nothing: when `glpsol` runs cleanly but the solution file cannot be read, `parseGLPKOutputFromFile` returns `nil` while the info table still says `Status: Success`.

This is one symptom of **LP-2**, already in `api-review.md` as a Med finding whose error shape is marked "待決" (pending a decision). LP-2 is larger than the nil: it also says the result table is GLPK's output copied line by line, with the variable values never parsed into columns, and that `timeoutSeconds ...int` should not be variadic. Its suggested fix is a redesign to `(*Solution, error)` carrying `Status`, `Objective` and `Variables map[string]float64`.

**This change does not do that redesign, on purpose.** Two reasons:

- **Its hard part cannot be verified here.** Parsing GLPK's output into variables means writing a parser for a format that can only be checked against the real `glpsol`, and `glpsol` is not installed on this machine and CI does not install it either. Writing it blind would ship a plausible-looking wrong answer, which is the failure mode this review exists to remove.
- **It depends on a decision that is still open.** LP-1 (#257, high) asks whether the runtime GLPK download-and-compile stays. If it goes, "glpsol not found" becomes an ordinary error path and shapes whatever error type LP-2 introduces. Adding an `error` return now and redesigning later would break every caller twice for one function.

So: take the crash out today without touching a signature, and leave the redesign to be decided once, with LP-1, by the owner.

## What Changes

- Every failure path returns an empty but usable `*DataTable` carrying the reason on `Err()`, instead of `nil`. `result.Show()` prints `(empty)`, `result.ToCSV(...)` writes an empty file, and `result.Err()` says what happened. Both returns are non-nil on every path, including the `nil, nil` ones.
- `parseGLPKOutputFromFile` does the same rather than returning `nil`.
- A solve that succeeds but whose solution file cannot be read reports `Status: Error` in the info table with the reason in its warnings, instead of `Status: Success` beside an empty result.
- `Docs/lp.md`'s example checks `Err()` before using the result, and the returns are documented as never nil.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `error-philosophy`: gains the rule that a function returning a `*DataTable` hands back a usable one carrying the reason, rather than a nil the caller's next line dereferences.

## Impact

- `lp/lp.go`, `Docs/lp.md`, tests.
- **BREAKING in the same sense the DataList transforms were**: a caller checking `result == nil` to detect failure will no longer see it and must check `result.Err()` instead. Nil was never documented as a return value, so nothing that follows the docs changes. Both changelogs get an entry saying so.
- No signature changes, so LP-2's redesign can still take whatever shape the owner decides.
- LP-2 has no GitHub issue, unlike LP-1 (#257) and LP-3 (#258). It should get one before it is decided.
