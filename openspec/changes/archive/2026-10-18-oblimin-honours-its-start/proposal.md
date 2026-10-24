# Proposal: oblimin-honours-its-start

## Why

`FaRotations`'s oblimin arm built its own identity start on every pass and ignored the one it was handed, so `Restarts: 20` ran the same computation twenty times (202 ms to return the 9.9 ms answer) while `Docs/stats.md` promised a search. psych's `faRotations` passes `Tmat = initial` to `GPArotation::oblimin` with no exception for oblimin, and GPArotation has `randomStarts`. SPSS is single-start, which is `Restarts: 1`.

## What Changes

- Oblimin rotates from each start like the other methods. `Restarts: 1` stays bit-identical (the only start is the identity); at `Delta` 0 it equals quartimin for the same `Restarts`.
- BREAKING for `Restarts` above 1, the default included: results can move to a lower-criterion basin.

## Capabilities

### Modified Capabilities

- `factor-analysis`: see What Changes.

## Impact

`stats/internal/fa/psych_faRotations.go`, `stats/internal/fa/rotation_starts_test.go`, `stats/verify_more_test.go`, `Docs/stats.md`, both CHANGELOGs, `AGENTS.md` follow-up removed.
