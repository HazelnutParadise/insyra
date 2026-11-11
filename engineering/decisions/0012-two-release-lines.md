# 0012 — Breaking changes go on `0.4`; `dev` stays on the 0.3.x line

**Status:** accepted
**Date:** 2026-09-14

## Context
The whole-repository API review carries breaking changes: the library no longer terminates on a fatal error, `Err()` is sticky, `gplot.SaveChart` returns an error, and several methods stopped returning `nil`. Feature releases still had to ship while the review continued.

## Decision
The API review and every breaking change live on `0.4`. `dev` stays on the 0.3.x line. A `0.4` change reaches `dev` only when code or data that worked on v0.3.2 keeps its result type, returned values, nil and error behaviour, written files and defaults. A fix whose old behaviour was a panic, a process exit, lost data or an answer wrong by any reading qualifies. Each port commit names its `0.4` source and what it left behind. `dev` is merged into `0.4` periodically so the review line keeps the 0.3.x fixes.

The owner decided five cases the rule left open: rotation changes at `Restarts >= 2`, `IsEqualTo` treating NaN as equal, and temp-file-and-rename writes stay on `0.4`; Parquet `Binary` columns read as `[]byte` and `ReadJSON_File` reads integers as `int64` on `dev`.

## Options rejected
- Cherry-pick `0.4` commits as they are. Most mix breaking and non-breaking parts, and later fixes are written on top of sticky `Err()` and `fail()`, so they are ported by hand against the rule above.

## Consequences
`api-review.md` stays on `0.4`. Engineering documents on the two lines can differ where the lines differ.
